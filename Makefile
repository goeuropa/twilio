.PHONY: build run test test-short test-coverage clean lint fmt vet deps tidy help

# Binary name
BINARY_NAME=oba-twilio

# Build the application
build:
	go build -o $(BINARY_NAME) .

# Build and run the application
run: build
	./$(BINARY_NAME)

# Run without building (development mode)
dev:
	go run main.go

# Run all tests
test:
	go test ./...

# Run tests without integration tests (faster for development)
test-short:
	go test -short ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Lint the code (requires golangci-lint)
lint:
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Vet the code
vet:
	go vet ./...

# Install/update dependencies
deps:
	go mod download

# Tidy up go.mod and go.sum
tidy:
	go mod tidy

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Display help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run in development mode (go run)"
	@echo "  test          - Run all tests"
	@echo "  test-short    - Run tests without integration tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  lint          - Lint the code (requires golangci-lint)"
	@echo "  fmt           - Format the code"
	@echo "  vet           - Vet the code"
	@echo "  deps          - Install/update dependencies"
	@echo "  tidy          - Tidy up go.mod and go.sum"
	@echo "  clean         - Clean build artifacts"
	@echo "  help          - Display this help message"