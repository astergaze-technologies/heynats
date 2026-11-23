# Makefile for React + Gin project

.PHONY: build-client build-server build run dev dev-full cluster-up cluster-down dev-cluster clean install help

# Default target
help:
	@echo "Available commands:"
	@echo "  make install       - Install dependencies for both client and server"
	@echo "  make build-client  - Build React client for production"
	@echo "  make build-server  - Build Go server"
	@echo "  make build         - Build both client and server"
	@echo "  make run           - Build client and run Go server (production)"
	@echo "  make dev           - Run client in development mode only"
	@echo "  make dev-full      - Run both client and server in development mode"
	@echo "  make cluster-up    - Start 3-node NATS cluster (ports 4222-4224, monitor 8222-8224)"
	@echo "  make cluster-down  - Stop the 3-node NATS cluster"
	@echo "  make dev-cluster   - Start the cluster, then dev-full"
	@echo "  make clean         - Clean build artifacts"

# Install dependencies
install:
	@echo "Installing Go dependencies..."
	go mod tidy
	@echo "Installing client dependencies..."
	cd client && pnpm install

# Build React client
build-client:
	@echo "Building React client..."
	cd client && pnpm build
	@echo "Client built successfully!"

# Build Go server
build-server:
	@echo "Building Go server..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/server main.go
	@echo "Server built successfully!"

# Build both client and server
build: build-client build-server

# Build client and run server (main command)
run: build-client
	@echo "Starting Gin server..."
	go run main.go

# Run client in development mode
dev:
	@echo "Starting React development server..."
	cd client && pnpm dev

# Run both client and server in development mode
dev-full:
	@echo "Starting full development environment..."
	./dev.sh

cluster-up:
	docker compose -f docker-compose.cluster.yml up -d

cluster-down:
	docker compose -f docker-compose.cluster.yml down

dev-cluster: cluster-up dev-full

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf client/dist
	rm -rf bin
	@echo "Clean completed!"