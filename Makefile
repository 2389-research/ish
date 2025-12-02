# ABOUTME: Makefile for ISH - fake Google API server.
# ABOUTME: Provides targets for building, testing, and running the server.

.PHONY: build test run seed reset clean help

# Default target
all: build

# Build the binary
build:
	go build -o ish ./cmd/ish

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run the server (builds first if needed)
run: build
	./ish serve

# Seed the database with test data
seed: build
	./ish seed

# Reset the database (wipe + reseed)
reset: build
	./ish reset

# Run server on custom port
run-port: build
	./ish serve --port $(PORT)

# Clean build artifacts
clean:
	rm -f ish ish.db coverage.out coverage.html
	rm -f test_*.db

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o ish-darwin-amd64 ./cmd/ish
	GOOS=darwin GOARCH=arm64 go build -o ish-darwin-arm64 ./cmd/ish
	GOOS=linux GOARCH=amd64 go build -o ish-linux-amd64 ./cmd/ish

# Help
help:
	@echo "ISH - Fake Google API Server"
	@echo ""
	@echo "Usage:"
	@echo "  make build      Build the binary"
	@echo "  make test       Run all tests"
	@echo "  make cover      Run tests with coverage report"
	@echo "  make run        Build and run the server on :9000"
	@echo "  make seed       Seed the database with test data"
	@echo "  make reset      Reset the database (wipe + reseed)"
	@echo "  make clean      Remove build artifacts"
	@echo "  make deps       Download and tidy dependencies"
	@echo "  make fmt        Format code"
	@echo "  make vet        Vet code"
	@echo "  make lint       Run golangci-lint"
	@echo "  make build-all  Build for darwin/linux amd64/arm64"
	@echo ""
	@echo "Custom port:"
	@echo "  make run-port PORT=8080"
