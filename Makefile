# LedgerFlow Payment & Billing Service Makefile

GOPATH ?= $(shell go env GOPATH)
SWAG := $(GOPATH)/bin/swag
APP_NAME := ledgerflow-payment-service
MAIN_PATH := ./cmd/server/main.go

.PHONY: all build run test clean swagger swagger-all swagger-billing swagger-health swagger-payment install-tools help

all: build

## install-tools: Install swag CLI tool if not present
install-tools:
	@which swag > /dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed successfully."

## swagger: Generate Swagger documentation for all modules and overall API
swagger: install-tools swagger-all swagger-billing swagger-health swagger-payment
	@echo "All module Swagger specs updated successfully!"

## swagger-all: Generate consolidated API Swagger documentation
swagger-all:
	@echo "Generating consolidated Swagger documentation..."
	@mkdir -p docs/swagger/all
	@$(SWAG) init -g $(MAIN_PATH) -o docs/swagger/all --parseDependency --parseInternal

## swagger-billing: Generate Billing module Swagger documentation
swagger-billing:
	@echo "Generating Billing module Swagger documentation..."
	@mkdir -p docs/swagger/billing
	@$(SWAG) init -g handler.go --dir ./internal/modules/billing -o docs/swagger/billing --parseDependency --parseInternal

## swagger-health: Generate Health module Swagger documentation
swagger-health:
	@echo "Generating Health module Swagger documentation..."
	@mkdir -p docs/swagger/health
	@$(SWAG) init -g handler.go --dir ./internal/modules/health -o docs/swagger/health --parseDependency --parseInternal

## swagger-payment: Generate Payment module Swagger documentation
swagger-payment:
	@echo "Generating Payment module Swagger documentation..."
	@mkdir -p docs/swagger/payment
	@$(SWAG) init -g handler.go --dir ./internal/modules/payment -o docs/swagger/payment --parseDependency --parseInternal

## build: Build application binary
build:
	@echo "Building $(APP_NAME)..."
	@go build -o build/$(APP_NAME) $(MAIN_PATH)

## run: Run the service locally
run: swagger
	@echo "Starting $(APP_NAME)..."
	@go run $(MAIN_PATH)

## test: Run unit tests
test:
	@echo "Running tests..."
	@go test -v ./...

## clean: Clean build directory and generated docs
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/ docs/swagger/

## help: Show Makefile targets
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
