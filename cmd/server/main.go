package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type PythonService struct {
	Host string
	Port int
}

type RemoveBackgroundRequest struct {
	FileBase64 string `json:"file_base64"`
}

type RemoveBackgroundResponse struct {
	Data struct {
		OutputFilename string `json:"output_filename"`
		OutputMime     string `json:"output_mime"`
		SizeBytes      int    `json:"size_bytes"`
		ProcessingMs   int    `json:"processing_ms"`
		OutputBase64   string `json:"output_base64"`
	} `json:"data"`
}

type ErrorResponse struct {
	Detail string `json:"detail"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Configuration
	port := getEnv("PORT", "30019")
	apiKey := getEnv("RMBG_API_KEY", "dev-rmbg-key")
	pythonHost := getEnv("PYTHON_SERVICE_HOST", "localhost")
	pythonPort := getEnvInt("PYTHON_SERVICE_PORT", 30020)

	// Initialize Python service
	pythonService := &PythonService{
		Host: pythonHost,
		Port: pythonPort,
	}

	// Gin setup
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Middleware
	r.Use(authMiddleware(apiKey))
	r.Use(corsMiddleware())
	r.Use(loggingMiddleware())

	// Routes
	r.GET("/healthz", healthCheck(pythonService))
	r.POST("/v1/rmbg/remove", handleFileUpload(pythonService))
	r.POST("/v1/rmbg/remove/base64", handleBase64(pythonService))

	// Start server
	log.Printf("Starting server on port %s", port)
	log.Printf("Python service at %s:%d", pythonHost, pythonPort)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func authMiddleware(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		apiKeyHeader := c.GetHeader("X-API-Key")

		var key string
		if apiKeyHeader != "" {
			key = strings.TrimSpace(apiKeyHeader)
		} else if authHeader != "" {
			auth := strings.ToLower(strings.TrimSpace(authHeader))
			if strings.HasPrefix(auth, "bearer ") {
				key = strings.TrimSpace(authHeader[7:])
			}
		}

		if key != expectedKey {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

func healthCheck(pythonService *PythonService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check Python service health
		pythonURL := fmt.Sprintf("http://%s:%d/healthz", pythonService.Host, pythonService.Port)
		resp, err := http.Get(pythonURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "degraded",
				"warning": "python service unavailable",
				"detail":  err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "degraded",
				"warning": "python service unhealthy",
				"detail":  fmt.Sprintf("python service returned %d", resp.StatusCode),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func handleFileUpload(pythonService *PythonService) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "file is required"})
			return
		}

		// Read file content
		fileContent, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "failed to read file"})
			return
		}
		defer fileContent.Close()

		// Convert to base64
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, fileContent); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "failed to process file"})
			return
		}

		base64Content := base64.StdEncoding.EncodeToString(buf.Bytes())

		// Call Python service
		result, err := callPythonService(pythonService, base64Content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}

		// Add filename info
		result.Data.OutputFilename = "output.png"
		if file.Filename != "" {
			// Keep original filename info if needed
		}

		c.JSON(http.StatusOK, gin.H{
			"data": result.Data,
		})
	}
}

func handleBase64(pythonService *PythonService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RemoveBackgroundRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid JSON"})
			return
		}

		if req.FileBase64 == "" {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "file_base64 is required"})
			return
		}

		// Validate base64
		if _, err := base64.StdEncoding.DecodeString(req.FileBase64); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid file_base64"})
			return
		}

		// Call Python service
		result, err := callPythonService(pythonService, req.FileBase64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": result.Data,
		})
	}
}

func callPythonService(pythonService *PythonService, base64Content string) (*RemoveBackgroundResponse, error) {
	start := time.Now()

	// Prepare request
	req := RemoveBackgroundRequest{
		FileBase64: base64Content,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Python service
	pythonURL := fmt.Sprintf("http://%s:%d/v1/rmbg/remove/base64", pythonService.Host, pythonService.Port)
	resp, err := http.Post(pythonURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call python service: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("python service returned %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("python service error: %s", errResp.Detail)
	}

	// Parse success response
	var result RemoveBackgroundResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Add processing time
	result.Data.ProcessingMs = int(time.Since(start).Milliseconds())

	return &result, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
