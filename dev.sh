#!/bin/bash

# Load .env variables
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo ".env file not found. Please create one from example.env"
  exit 1
fi

# Ensure data directory exists
mkdir -p data

# Ensure DATABASE_URL is set for local dev if not in .env
if [ -z "$DATABASE_URL" ]; then
  export DATABASE_URL="$(pwd)/data/merchant.db"
fi

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

# Export CGO_CFLAGS to include the local sqlite headers
export CGO_CFLAGS="-I$(pwd)/include"

# Run Air for each service pointing to its specific config
air -c .air/bff.toml &
air -c .air/manager.toml &
air -c .air/trader.toml &

wait
