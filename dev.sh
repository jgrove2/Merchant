#!/bin/bash

# Default values
SLM_URL="http://localhost:8088/v1"

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

# Set REDIS_URL for local dev if not in .env
if [ -z "$REDIS_URL" ]; then
  export REDIS_URL="redis://localhost:6379"
fi

# Function to determine docker compose command
get_compose_cmd() {
    # Prefer 'docker compose' (v2)
    if docker compose version >/dev/null 2>&1; then
        echo "docker compose"
        return 0
    fi

    # Fallback to 'docker-compose' (v1)
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
        return 0
    fi

    echo "Error: Docker Compose not found." >&2
    exit 1
}

COMPOSE_CMD=$(get_compose_cmd)
echo "Using compose command: $COMPOSE_CMD"

# Start Redis and SLM via Docker (Dev Mode)
echo "Starting Redis and SLM containers..."

# Check if we need sudo privileges (user not in docker group)
SUDO=""
if ! docker info >/dev/null 2>&1; then
    echo "Warning: Current user does not have permission to run docker."
    echo "Requesting sudo privileges..."
    SUDO="sudo"
    
    # NOTE: Since we installed the 'docker compose' plugin for the USER,
    # 'sudo docker compose' will likely fail because root doesn't have the plugin.
    # In this specific case (V2 plugin installed for user + sudo required), we might hit issues.
    # We'll try to execute it, but if it fails, we'll warn the user.
fi

# Execute Docker Compose
# If sudo is required, we attempt to run it.
# If using 'docker compose', sudo might not find the plugin in root's home.
$SUDO $COMPOSE_CMD -f docker-compose.dev.yml up -d --remove-orphans || {
    echo "Error: Failed to start containers."
    if [ -n "$SUDO" ] && [ "$COMPOSE_CMD" = "docker compose" ]; then
        echo "Troubleshooting: You are running 'docker compose' (V2) with sudo."
        echo "The V2 plugin is installed for your user (~/.docker/cli-plugins), but not for root."
        echo "Solution 1 (Recommended): Add your user to the docker group: 'sudo usermod -aG docker \$USER' and re-login."
        echo "Solution 2: Install the plugin for root: 'sudo cp ~/.docker/cli-plugins/docker-compose /usr/local/lib/docker/cli-plugins/'"
    fi
    exit 1
}

# Wait for SLM to be ready
echo "Waiting for SLM service at $SLM_URL..."
MAX_RETRIES=60
COUNT=0
while [ $COUNT -lt $MAX_RETRIES ]; do
    if curl -s "$SLM_URL/models" >/dev/null; then
        echo "SLM service is ready!"
        break
    fi
    sleep 2
    COUNT=$((COUNT+1))
    echo -n "."
done

if [ $COUNT -eq $MAX_RETRIES ]; then
    echo "Warning: SLM service might not be ready. Proceeding anyway..."
fi

# Cleanup function
cleanup() {
  echo "Stopping services..."
  $SUDO $COMPOSE_CMD -f docker-compose.dev.yml stop
  kill $(jobs -p) 2>/dev/null
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
