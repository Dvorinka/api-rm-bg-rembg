#!/bin/bash

# Change to app directory
cd "$(dirname "$0")"

# Activate virtual environment (check if we're in production or local)
if [ -d "/opt/venv" ]; then
    source /opt/venv/bin/activate
else
    # For local testing
    if [ -d "test_venv" ]; then
        source test_venv/bin/activate
    fi
fi

# Start the application
python app/main.py
