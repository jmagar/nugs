.PHONY: all clean catalog monitor detector apimon api build test test-unit test-integration test-coverage lint fmt vet sec quality dev install-air

# Build all binaries
all: catalog monitor detector apimon api gap_report

# Individual builds
catalog:
	go build -o bin/catalog_manager ./cmd/catalog

monitor:
	go build -o bin/monitor_artists ./cmd/monitor

detector:
	go build -o bin/missing_shows_detector ./cmd/detector

apimon:
	go build -o bin/api_monitor ./cmd/apimon

gap_report:
	go build -o bin/gap_report ./cmd/gap_report

# Build API server
api:
	go build -o bin/nugs-api ./cmd/api

# Install Air for hot reload development
install-air:
	@echo "Installing Air for hot reload..."
	@if ! command -v air >/dev/null 2>&1; then \
		go install github.com/air-verse/air@latest; \
	else \
		echo "Air is already installed"; \
	fi

# Run API server with hot reload (development mode)
dev: install-air
	@echo "Starting API server with hot reload..."
	@echo "Visit: http://localhost:8080"
	@echo "Swagger: http://localhost:8080/swagger/index.html"
	@echo "Press Ctrl+C to stop"
	air

# Build everything (same as all)
build: all

# Test the build process and run tests
test: deps test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -timeout 300s ./internal/...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -timeout 300s ./test/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -timeout 300s -coverprofile=coverage.out ./internal/... ./test/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./cmd/... ./internal/... || echo "Linter found issues (non-blocking)"; \
	else \
		echo "Note: golangci-lint not installed. Skipping lint check."; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./cmd/... ./internal/...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./cmd/... ./internal/...

# Run security scan
sec:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./cmd/... ./internal/...; \
	else \
		echo "Note: gosec not installed. Skipping security scan."; \
	fi

# Run quality checks
quality: fmt vet lint

# Legacy test command (for backward compatibility)
test-legacy:
	go mod tidy
	go vet ./...
	go build ./...

# Clean binaries
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf tmp/

# Install dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  all               - Build all binaries (CLI tools + API)"
	@echo "  catalog           - Build catalog_manager"
	@echo "  monitor           - Build monitor_artists"
	@echo "  detector          - Build missing_shows_detector"
	@echo "  apimon            - Build api_monitor"
	@echo "  api               - Build API server"
	@echo "  gap_report        - Build gap_report"
	@echo "  dev               - Start API server with hot reload (development)"
	@echo "  install-air       - Install Air hot reload tool"
	@echo "  test              - Run all tests (unit + integration)"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  lint              - Run linter (if available)"
	@echo "  fmt               - Format code"
	@echo "  vet               - Vet code"
	@echo "  sec               - Run security scan (if available)"
	@echo "  quality           - Run all quality checks"
	@echo "  clean             - Remove binaries and coverage files"
	@echo "  deps              - Install dependencies"