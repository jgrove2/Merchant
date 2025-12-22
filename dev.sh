#!/bin/bash

# Load .env variables
export $(grep -v '^#' .env | xargs)

# Cleanup
cleanup() {
  kill $(jobs -p)
  exit
}
trap cleanup SIGINT SIGTERM

echo "Starting Dev Environment..."

# Frontend
cd merchant_ui && npm run dev &

# Backend Services
cd backend

# Run Air for each service pointing to its specific config
air -c .air/bff.toml &
air -c .air/manager.toml &
air -c .air/trader.toml &

wait
