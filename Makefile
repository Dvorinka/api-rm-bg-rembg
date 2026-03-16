.PHONY: up down logs rebuild clean status health test help

# Default target - start services
all: up

# Start services
up:
	docker compose up -d --build

# Stop services
down:
	docker compose down

# View logs
logs:
	docker compose logs -f

# Rebuild and restart
rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d

# Clean Docker resources
clean:
	docker compose down -v
	docker system prune -f
	docker image prune -f

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
test:
	@echo "Testing API..."
	@curl -s -H "Authorization: Bearer $${RMBG_API_KEY:-dev-rmbg-key}" -H "Content-Type: application/json" -d '{"file_base64":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="}' http://localhost:30019/v1/rmbg/remove/base64 | jq .

# Help
help:
	@echo "Available commands:"
	@echo "  up       - Build and start all services"
	@echo "  down     - Stop all services"
	@echo "  logs     - View logs"
	@echo "  rebuild  - Rebuild and restart services"
	@echo "  clean    - Clean Docker resources"
	@echo "  status   - View service status"
	@echo "  health   - Check service health"
	@echo "  test     - Test API endpoints"
	@echo "  help     - Show this help"
