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
	GOMODCACHE=$(GOMODCACHE) go build ./cmd/paradiced-server

# Run tests
test:
	GOMODCACHE=$(GOMODCACHE) go test ./...

# Run all tests with coverage
test-coverage:
	GOMODCACHE=$(GOMODCACHE) go test -cover ./...

# Create modules directory
prepare-modules:
	mkdir -p ./modules

# Start Nakama server with Docker
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

# Full development cycle: prepare, build, and start
dev: prepare-modules build-plugin docker-up

# Development cycle with logs
dev-logs: dev docker-logs

# Default target
.PHONY: build-plugin build-dev test test-coverage prepare-modules docker-up docker-down docker-clean docker-logs dev dev-logs
.DEFAULT_GOAL := build-dev