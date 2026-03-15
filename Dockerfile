# Build stage
FROM python:3.11-slim AS builder

WORKDIR /app

# Copy requirements
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Final stage
FROM python:3.11-slim

RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the installed packages from builder stage
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy source code
COPY . .

# Expose port
EXPOSE 30019

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:30019/healthz || exit 1

# Run the application directly
CMD ["python", "app/main.py"]
