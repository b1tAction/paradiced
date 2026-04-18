# Paradiced - Makefile for building and running Nakama server

# Go module cache
GOMODCACHE ?= /app/.gomodcache

# Build output
PLUGIN_NAME ?= paradiced-server
PLUGIN_OUT ?= ./modules/$(PLUGIN_NAME).so

# Docker compose file
DOCKER_COMPOSE ?= docker-compose.yml

# Build the Nakama plugin as a shared object
build-plugin:
	GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_OUT) ./cmd/paradiced-server

# Build for development (verify compilation without plugin mode)
build-dev:
	GOMODCACHE=$(GOMODCACHE) go build ./...

# Run tests
test:
	GOMODCACHE=$(GOMODCACHE) go test ./...

# Run all tests with coverage
test-coverage:
	GOMODCACHE=$(GOMODCACHE) go test -cover ./...

# Create modules directory
prepare-modules:
	mkdir -p ./modules

# Start Nakama server with Docker (CockroachDB + Nakama)
docker-up:
	docker-compose up -d

# Stop Nakama server
docker-down:
	docker-compose down

# Stop and remove all data
docker-clean:
	docker-compose down -v

# View Nakama logs
docker-logs:
	docker-compose logs -f nakama

# View CockroachDB logs
docker-logs-db:
	docker-compose logs -f cockroachdb

# View all logs
docker-logs-all:
	docker-compose logs -f

# CockroachDB admin UI (http://localhost:8080)
cockroach-admin:
	@echo "CockroachDB Admin UI: http://localhost:8080"

# Initialize database (create nakama database)
db-init:
	@echo "Creating nakama database..."
	docker exec paradiced-cockroachdb cockroach sql --insecure -e "CREATE DATABASE IF NOT EXISTS nakama;"

# Full development cycle: prepare, build, and start
dev: prepare-modules build-plugin docker-up

# Wait for services to be healthy and initialize database
dev-init: dev
	@sleep 5
	@docker exec paradiced-cockroachdb cockroach sql --insecure -e "CREATE DATABASE IF NOT EXISTS nakama;" || true

# Development cycle with logs
dev-logs: dev docker-logs

# Rebuild plugin and restart Nakama (hot reload)
rebuild: build-plugin
	docker-compose restart nakama

# Check service status
status:
	docker-compose ps

# Default target
.PHONY: build-plugin build-dev test test-coverage prepare-modules docker-up docker-down docker-clean docker-logs docker-logs-db docker-logs-all cockroach-admin db-init dev dev-init dev-logs rebuild status
.DEFAULT_GOAL := build-dev