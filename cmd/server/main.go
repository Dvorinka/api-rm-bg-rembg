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

	// Health check endpoint (no auth required)
	r.GET("/healthz", healthCheck(pythonService))

	// Landing page (no auth required)
	r.GET("/", landingPage())
	r.GET("/api", apiDocumentation())
	r.GET("/services", serviceList())

	// Middleware for protected routes
	r.Use(authMiddleware(apiKey))
	r.Use(corsMiddleware())
	r.Use(loggingMiddleware())

	// Protected Routes
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

func landingPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>RM BG Rembg API</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
               min-height: 100vh; color: #333; }
        .container { max-width: 800px; margin: 0 auto; padding: 40px 20px; }
        .card { background: white; border-radius: 16px; padding: 40px; 
                box-shadow: 0 20px 40px rgba(0,0,0,0.1); margin-bottom: 20px; }
        h1 { color: #667eea; margin-bottom: 20px; font-size: 2.5em; text-align: center; }
        h2 { color: #764ba2; margin-bottom: 15px; font-size: 1.5em; }
        .endpoint { background: #f8f9fa; border-left: 4px solid #667eea; 
                    padding: 15px; margin: 10px 0; border-radius: 4px; }
        .method { display: inline-block; padding: 4px 8px; border-radius: 4px; 
                  font-weight: bold; font-size: 0.9em; margin-right: 10px; }
        .get { background: #28a745; color: white; }
        .post { background: #007bff; color: white; }
        .code { background: #f1f3f4; padding: 2px 6px; border-radius: 3px; 
                font-family: 'Courier New', monospace; font-size: 0.9em; }
        .btn { display: inline-block; background: #667eea; color: white; 
               padding: 12px 24px; text-decoration: none; border-radius: 8px; 
               margin: 10px 5px; transition: all 0.3s; }
        .btn:hover { background: #5a6fd8; transform: translateY(-2px); }
        .status { padding: 8px 16px; border-radius: 20px; font-size: 0.9em; 
                 font-weight: bold; display: inline-block; margin: 5px; }
        .online { background: #d4edda; color: #155724; }
        .feature { background: #e7f3ff; padding: 20px; border-radius: 8px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <h1>🎨 RM BG Rembg API</h1>
            <p style="text-align: center; font-size: 1.2em; color: #666; margin-bottom: 30px;">
                Remove backgrounds from images using AI-powered background removal
            </p>
            
            <div style="text-align: center; margin-bottom: 30px;">
                <span class="status online">🟢 Service Online</span>
                <a href="/api" class="btn">📚 API Documentation</a>
                <a href="/healthz" class="btn">💊 Health Check</a>
            </div>

            <div class="feature">
                <h2>✨ Features</h2>
                <ul style="margin-left: 20px; line-height: 1.8;">
                    <li>Remove backgrounds from images automatically</li>
                    <li>Support for file upload and base64 input</li>
                    <li>Fast processing with Python backend</li>
                    <li>RESTful API with authentication</li>
                    <li>Health monitoring and status tracking</li>
                </ul>
            </div>

            <h2>🚀 Quick Start</h2>
            <div class="endpoint">
                <span class="method post">POST</span>
                <code>/v1/rmbg/remove</code>
                <p>Upload an image file to remove background</p>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <code>/v1/rmbg/remove/base64</code>
                <p>Send base64 image data to remove background</p>
            </div>

            <h2>📝 Authentication</h2>
            <p>All API endpoints (except health check and landing page) require authentication:</p>
            <div style="background: #f8f9fa; padding: 15px; border-radius: 8px; margin: 10px 0;">
                <code>Header: X-API-Key: dev-rmbg-key</code>
            </div>

            <h2>🔧 Try It Out</h2>
            <p style="background: #fff3cd; padding: 15px; border-radius: 8px; border-left: 4px solid #ffc107;">
                <strong>Example:</strong> Use Postman or curl to test the API with your image files.
            </p>
        </div>
    </div>
</body>
</html>`
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	}
}

func apiDocumentation() gin.HandlerFunc {
	return func(c *gin.Context) {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Documentation - RM BG Rembg</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
               background: #f8f9fa; line-height: 1.6; color: #333; }
        .container { max-width: 1000px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
                  color: white; padding: 40px; border-radius: 12px; margin-bottom: 30px; }
        .endpoint { background: white; border-radius: 8px; padding: 25px; 
                   margin-bottom: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .method { display: inline-block; padding: 6px 12px; border-radius: 4px; 
                  font-weight: bold; color: white; margin-right: 10px; }
        .get { background: #28a745; }
        .post { background: #007bff; }
        .code { background: #f1f3f4; padding: 15px; border-radius: 6px; 
                font-family: 'Courier New', monospace; margin: 10px 0; overflow-x: auto; }
        .param { background: #e7f3ff; padding: 10px; margin: 5px 0; border-radius: 4px; }
        .response { background: #d4edda; padding: 15px; border-radius: 6px; margin: 10px 0; }
        .error { background: #f8d7da; padding: 15px; border-radius: 6px; margin: 10px 0; }
        .nav { margin-bottom: 20px; }
        .nav a { background: #667eea; color: white; padding: 10px 20px; 
                 text-decoration: none; border-radius: 6px; margin-right: 10px; }
        .nav a:hover { background: #5a6fd8; }
        h1, h2 { margin-bottom: 15px; }
        h3 { margin: 20px 0 10px 0; color: #667eea; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/">← Back to Home</a>
        </div>
        
        <div class="header">
            <h1>📚 API Documentation</h1>
            <p>Complete API reference for RM BG Rembg service</p>
        </div>

        <div class="endpoint">
            <h2><span class="method get">GET</span> Health Check</h2>
            <code>GET /healthz</code>
            <p>Check the health status of the API and Python service.</p>
            
            <h3>Response:</h3>
            <div class="response">
<code>{
  "status": "ok"
}</code>
            </div>
            
            <h3>Status Codes:</h3>
            <ul>
                <li><strong>200</strong> - Service healthy</li>
                <li><strong>503</strong> - Python service unavailable</li>
            </ul>
        </div>

        <div class="endpoint">
            <h2><span class="method post">POST</span> Remove Background (File Upload)</h2>
            <code>POST /v1/rmbg/remove</code>
            <p>Upload an image file to remove its background.</p>
            
            <h3>Authentication:</h3>
            <div class="param">
                <strong>Header:</strong> X-API-Key: dev-rmbg-key
            </div>
            
            <h3>Request:</h3>
            <div class="param">
                <strong>Form Data:</strong> file (multipart/form-data)
            </div>
            
            <h3>Response:</h3>
            <div class="response">
<code>{
  "data": {
    "output_filename": "output.png",
    "output_mime": "image/png",
    "size_bytes": 12345,
    "processing_ms": 150,
    "output_base64": "iVBORw0KGgoAAAANSUhEUgAA..."
  }
}</code>
            </div>
        </div>

        <div class="endpoint">
            <h2><span class="method post">POST</span> Remove Background (Base64)</h2>
            <code>POST /v1/rmbg/remove/base64</code>
            <p>Send base64 encoded image data to remove background.</p>
            
            <h3>Authentication:</h3>
            <div class="param">
                <strong>Header:</strong> X-API-Key: dev-rmbg-key
            </div>
            
            <h3>Request Body:</h3>
            <div class="code">
<code>{
  "file_base64": "iVBORw0KGgoAAAANSUhEUgAA..."
}</code>
            </div>
            
            <h3>Response:</h3>
            <div class="response">
<code>{
  "data": {
    "output_filename": "output.png",
    "output_mime": "image/png",
    "size_bytes": 12345,
    "processing_ms": 150,
    "output_base64": "iVBORw0KGgoAAAANSUhEUgAA..."
  }
}</code>
            </div>
        </div>

        <div class="endpoint">
            <h2>🔧 Error Responses</h2>
            <p>All endpoints may return these error responses:</p>
            
            <h3>401 Unauthorized</h3>
            <div class="error">
<code>{
  "detail": "unauthorized"
}</code>
            </div>
            
            <h3>400 Bad Request</h3>
            <div class="error">
<code>{
  "detail": "file is required"
}</code>
            </div>
            
            <h3>500 Internal Server Error</h3>
            <div class="error">
<code>{
  "detail": "python service error: processing failed"
}</code>
            </div>
        </div>

        <div class="endpoint">
            <h2>🚀 Quick Examples</h2>
            
            <h3>cURL - File Upload:</h3>
            <div class="code">
<code>curl -X POST \
  http://localhost:30019/v1/rmbg/remove \
  -H "X-API-Key: dev-rmbg-key" \
  -F "file=@/path/to/your/image.jpg"
</code>
            </div>
            
            <h3>cURL - Base64:</h3>
            <div class="code">
<code>curl -X POST \
  http://localhost:30019/v1/rmbg/remove/base64 \
  -H "X-API-Key: dev-rmbg-key" \
  -H "Content-Type: application/json" \
  -d '{"file_base64": "iVBORw0KGgoAAAANSUhEUgAA..."}'
</code>
            </div>
            
            <h3>JavaScript - Base64:</h3>
            <div class="code">
<code>const response = await fetch('http://localhost:30019/v1/rmbg/remove/base64', {
  method: 'POST',
  headers: {
    'X-API-Key': 'dev-rmbg-key',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    file_base64: base64ImageData
  })
});

const result = await response.json();
console.log(result.data.output_base64);
</code>
            </div>
        </div>
    </div>
</body>
</html>`
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	}
}

func serviceList() gin.HandlerFunc {
	return func(c *gin.Context) {
		services := []map[string]interface{}{
			{
				"name":        "RM BG Rembg API",
				"version":     "v1",
				"description": "AI-powered background removal service",
				"endpoints": []map[string]string{
					{"method": "GET", "path": "/", "description": "Landing page"},
					{"method": "GET", "path": "/healthz", "description": "Health check"},
					{"method": "GET", "path": "/api", "description": "API documentation"},
					{"method": "GET", "path": "/services", "description": "Service list"},
					{"method": "POST", "path": "/v1/rmbg/remove", "description": "Remove background (file upload)"},
					{"method": "POST", "path": "/v1/rmbg/remove/base64", "description": "Remove background (base64)"},
				},
				"authentication": map[string]string{
					"type":   "API Key",
					"header": "X-API-Key",
					"value":  "dev-rmbg-key",
				},
				"ports": map[string]int{
					"go_server":      30019,
					"python_service": 30020,
				},
				"status": "active",
			},
		}

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{"services": services})
	}
}
