# Railway deployment Dockerfile - Single service approach
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y wget curl golang && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy Python requirements
COPY python_service/requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy Go modules
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY python_service/ ./python_service/

# Build Go server
RUN CGO_ENABLED=0 go build -o go-server cmd/server/main.go

# Create non-root user
RUN useradd --create-home --shell /bin/bash appuser
RUN chown -R appuser:appuser /app
USER appuser

# Expose port
EXPOSE 30019

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:30019/healthz || exit 1

# Start both services
CMD ["sh", "-c", "chmod +x start.sh && ./start.sh"]
