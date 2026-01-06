#!/bin/bash

# Ensure we are in the project root (where the script is located)
cd "$(dirname "$0")"

# Load .env variables
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    # Use set -a to automatically export variables defined in .env
    set -a
    source .env
    set +a
else
    echo "Warning: .env file not found."
fi

# Change to backend directory to run the go command within the module context
cd backend

# Run the benchmark script
# Passing "$@" allows the user to provide flags like -model or -cases if needed
echo "Starting benchmark..."
go run cmd/benchmark/main.go "$@"
