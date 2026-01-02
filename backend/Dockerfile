# backend/Dockerfile
FROM golang:1.23-alpine AS builder
# Install build-base for CGO (required by sqlite-vec and go-sqlite3)
RUN apk add --no-cache build-base

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Enable CGO
ENV CGO_ENABLED=1
# Point to local include directory for SQLite headers if needed, 
# but usually build-base handles standard headers. 
# However, for sqlite-vec, we might need specific flags.
# We'll rely on the go.mod dependencies to pull in the C code, 
# but we need gcc (via build-base) to compile it.

RUN go build -o /bin/bff ./cmd/bff/main.go
RUN go build -o /bin/manager ./cmd/manager/main.go
RUN go build -o /bin/trader ./cmd/trader/main.go

# Final stage for all Go apps
FROM alpine:latest
RUN apk --no-cache add ca-certificates
# We need to copy the binaries to a location that matches our docker-compose command expectation
# docker-compose expects /app/manager, /app/bff, etc.
COPY --from=builder /bin/bff /app/bff
COPY --from=builder /bin/manager /app/manager
COPY --from=builder /bin/trader /app/trader

WORKDIR /app
