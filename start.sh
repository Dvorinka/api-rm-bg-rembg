#!/bin/bash

# Wait a moment for Python service to start
sleep 2

# Start Python service in background
cd /app
python python_service/main.py &

# Wait for Python service to be ready
echo "Waiting for Python service to start..."
for i in {1..30}; do
    if curl -s http://localhost:30020/healthz > /dev/null 2>&1; then
        echo "Python service is ready!"
        break
    fi
    echo "Waiting for Python service... ($i/30)"
    sleep 1
done

# Start Go server
echo "Starting Go server..."
exec ./go-server
