.PHONY: build run dev test clean docker-build docker-up docker-down docker-logs docker-rebuild

# Default target - build and run with Docker
all: docker-up

# Docker build
docker-build:
	docker build -f Dockerfile.go -t rm-bg-rembg-go:latest .
	docker build -f Dockerfile.python -t rm-bg-rembg-python:latest .

# Docker up (main command)
docker-up:
	docker compose up -d --build

# Docker down
docker-down:
	docker compose down

# Docker logs
docker-logs:
	docker compose logs -f

# Docker rebuild
docker-rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d

# Clean Docker resources
docker-clean:
	docker compose down -v
	docker system prune -f
	docker image prune -f

# Quick development setup
dev: docker-up

# Production deployment
deploy:
	ENVIRONMENT=production docker compose up -d --build

# View status
status:
	docker compose ps

# Health check
health:
	@echo "Checking Go server health..."
	@curl -s -H "Authorization: Bearer $${RMBG_API_KEY:-dev-rmbg-key}" http://localhost:30019/healthz | jq . || echo "Go server not ready"
	@echo "Checking Python service health..."
	@curl -s http://localhost:30020/healthz | jq . || echo "Python service not ready"

# Test API
test-api:
	@echo "Testing API..."
	@curl -s -H "Authorization: Bearer $${RMBG_API_KEY:-dev-rmbg-key}" -H "Content-Type: application/json" -d '{"file_base64":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="}' http://localhost:30019/v1/rmbg/remove/base64 | jq .

# Help
help:
	@echo "Available commands:"
	@echo "  docker-up      - Build and start all services"
	@echo "  docker-down    - Stop all services"
	@echo "  docker-logs    - View logs"
	@echo "  docker-rebuild - Rebuild and restart services"
	@echo "  docker-clean   - Clean Docker resources"
	@echo "  dev            - Quick development setup"
	@echo "  deploy         - Production deployment"
	@echo "  status         - View service status"
	@echo "  health         - Check service health"
	@echo "  test-api       - Test API endpoints"
	@echo "  help           - Show this help"
