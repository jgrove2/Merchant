# backend/Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /bin/bff ./cmd/bff/main.go
RUN go build -o /bin/manager ./cmd/manager/main.go
RUN go build -o /bin/trader ./cmd/trader/main.go

# Final stage for all Go apps
FROM alpine:latest
RUN apk --no-cache add ca-certificates
