# RmBgRembg API

Part of the API Services Collection - A comprehensive set of specialized APIs for modern applications.

**Architecture**: Go web server with Python microservice for ML processing

## � Docker-First Deployment

This project is designed to run primarily with Docker Compose for consistency across environments.

## �🚀 Quick Start

### Prerequisites
- Docker and Docker Compose installed
- Port 30019 available

### Development
```bash
# Clone the repository
git clone https://github.com/your-username/api-rm-bg-rembg.git
cd api-rm-bg-rembg

# Copy environment file
cp .env.example .env

# Edit .env with your API keys (optional for development)
vim .env

# Build and start all services
make docker-up

# Or directly with docker-compose
docker compose up -d
```

### Production (RapidAPI)
```bash
# Set production environment
export ENVIRONMENT=production
export RAPIDAPI_PROXY_SECRET=your-secret-here

# Deploy to production
make deploy

# Deploy to Coolify
# Use the coolify.yaml configuration
```

## 📋 API Documentation

- **Local**: http://localhost:30019/healthz
- **Health Check**: http://localhost:30019/healthz
- **Base URL**: http://localhost:30019/v1/rmbg/

## 🔐 Authentication

### Development Mode
Use Bearer token authentication:
```bash
curl -H "Authorization: Bearer dev-rmbg-key" \
     http://localhost:30019/v1/rmbg/remove
```

### Production Mode (RapidAPI)
Requests must include both headers:
```bash
curl -H "X-RapidAPI-Proxy-Secret: your-secret" \
     -H "Authorization: Bearer your-api-key" \
     https://your-api.p.rapidapi.com/v1/rmbg/remove
```

**Security Layers:**
1. RapidAPI authentication (user keys, quotas, billing)
2. Proxy secret validation (prevents bypass attacks)
3. Service API key validation

## 🐳 Docker Deployment

### Primary Method: Docker Compose
```bash
# Build and start all services
make docker-up

# View logs
make docker-logs

# Check service status
make status

# Health check
make health

# Test API
make test-api

# Stop services
make docker-down

# Rebuild and restart
make docker-rebuild

# Clean Docker resources
make docker-clean
```

### Individual Docker Commands
```bash
# Build images
make docker-build

# Production deployment
ENVIRONMENT=production docker compose up -d --build
```

## ☁️ Coolify Deployment

### Automatic Deployment
The `coolify.yaml` configuration includes:
- Docker image building
- Health checks
- Resource limits
- Environment variables
- Monitoring setup

### Manual Coolify Setup
1. Create Application → Docker → Git Repository
2. Repository: `your-username/api-rm-bg-rembg`
3. Configure environment variables
4. Set health check path: `/healthz`

## 📊 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 30019 |
| `ENVIRONMENT` | development/production | development |
| `RMBG_API_KEY` | Service API key | dev-rmbg-key |
| `PYTHON_SERVICE_HOST` | Python service host | localhost |
| `PYTHON_SERVICE_PORT` | Python service port | 30020 |
| `RAPIDAPI_PROXY_SECRET` | RapidAPI proxy secret | - |

## 🔧 Configuration

### Development Settings
```bash
# .env file
PORT=30019
ENVIRONMENT=development
RMBG_API_KEY=dev-rmbg-key
PYTHON_SERVICE_HOST=localhost
PYTHON_SERVICE_PORT=30020
```

### Production Settings
```bash
# .env file
PORT=30019
ENVIRONMENT=production
RMBG_API_KEY=your-production-api-key
PYTHON_SERVICE_HOST=python-service
PYTHON_SERVICE_PORT=30020
RAPIDAPI_PROXY_SECRET=your-rapidapi-secret
```

## 📈 Features

- Background removal
- AI-powered processing
- Multiple input formats
- High-quality output
- Batch processing
- API integration
- Fast processing

## 🔍 Monitoring & Health

### Health Check Endpoint
```bash
curl http://localhost:8080/healthz
```

Response:
```json
{"status":"ok"}
```

### Metrics (Optional)
If enabled, metrics available at:
```bash
curl http://localhost:8080/metrics
```

## 🚨 Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Kill processes using port 30019
   lsof -ti:30019 | xargs kill -9
   
   # Restart services
   make docker-up
   ```

2. **Authentication Failures**
   ```bash
   # Check API key
   curl -H "Authorization: Bearer dev-rmbg-key" http://localhost:30019/healthz
   ```

3. **Service Communication Issues**
   ```bash
   # Check service status
   make status
   
   # View logs
   make docker-logs
   
   # Health check
   make health
   ```

4. **Docker Build Issues**
   ```bash
   # Clean rebuild
   make docker-rebuild
   
   # Clean all Docker resources
   make docker-clean
   ```

5. **Environment Issues**
   ```bash
   # Check environment variables
   docker compose logs go-server
   docker compose logs python-service
   ```

## 📚 API Endpoints

### Base URL
```
http://localhost:30019/v1/rmbg/
```

### Common Endpoints
- `GET /healthz` - Health check
- `POST /v1/rmbg/remove` - Remove background from uploaded file
- `POST /v1/rmbg/remove/base64` - Remove background from base64 image

### Example Usage
```bash
# Upload file
curl -X POST \
  -H "Authorization: Bearer dev-rmbg-key" \
  -F "file=@image.jpg" \
  http://localhost:30019/v1/rmbg/remove

# Base64 input
curl -X POST \
  -H "Authorization: Bearer dev-rmbg-key" \
  -H "Content-Type: application/json" \
  -d '{"file_base64":"iVBORw0KGgoAAAANSUhEUgAA..."}' \
  http://localhost:30019/v1/rmbg/remove/base64
```

## 🛠️ Development

### Development Setup
```bash
# Quick start (Docker Compose)
make docker-up

# View available commands
make help

# Monitor services
docker compose ps
make docker-logs

# Test the API
make test-api

# Development with hot reload (requires local setup)
# Note: Not recommended - use Docker Compose for consistency
```

### Code Structure
```
rm-bg-rembg/
├── cmd/
│   └── server/
│       └── main.go         # Go web server entry point
├── python_service/
│   ├── main.py             # Python ML service
│   └── requirements.txt    # Python dependencies
├── Dockerfile.go           # Go server Docker image
├── Dockerfile.python       # Python service Docker image
├── docker-compose.yml      # Multi-service deployment
├── coolify.yaml           # Production deployment
├── go.mod                 # Go dependencies
├── Makefile               # Build automation
├── .env.example           # Environment variables template
└── README.md              # This file
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.

## 🔗 Related Services

This API is part of a larger collection:
- [API Services Collection](https://github.com/your-username/api-services)
- [Other individual APIs](https://github.com/your-username?tab=repositories)

## 🆘 Support

For issues and support:
1. Check the [troubleshooting section](#-troubleshooting)
2. Review the [API documentation](http://localhost:8080/docs)
3. Open an issue on GitHub
4. Contact support team

---

**Built with Go for performance and reliability.** 🚀
