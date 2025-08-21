.PHONY: all build run dev clean test test-unit test-integration migrate-up migrate-down sqlc docker-build docker-run docker-dev docker-stop

# Go variables
BINARY_NAME=app
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

# Default target
all: build

# Build the Go application
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/server/main.go

# Run the compiled application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Run the application with air for hot-reloading (development)
dev:
	@echo "Starting dev server with air..."
	@air

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f $(BINARY_NAME)

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -v ./... -tags=unit -coverprofile=coverage-unit.out

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v ./... -tags=integration -coverprofile=coverage-integration.out

migrate-up:
	@echo "Applying database migrations (via Docker)..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) exec app migrate -path /app/db/migration -database "$DB_URL" -verbose up

migrate-down:
	@echo "Rolling back last database migration (via Docker)..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) exec app migrate -path /app/db/migration -database "$DB_URL" -verbose down 1

sqlc:
	@echo "Generating SQLC code (via Docker)..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) exec app sqlc generate --path /app/sqlc.yaml

# Docker commands
DOCKER_COMPOSE_FILE=compose.yml:compose.dev.yml
DOCKER_IMAGE_NAME=yourprojectname # Change this to your desired image name

docker-build:
	@echo "Building Docker image..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) build

docker-run:
	@echo "Running Docker containers..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-dev:
	@echo "Running Docker containers for development (with hot-reloading)..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) up --build app

docker-stop:
	@echo "Stopping Docker containers..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) down

help:
	@echo "Available targets:"
	@echo "  all                - Build the application (default)"
	@echo "  build              - Build the Go application"
	@echo "  run                - Run the compiled application"
	@echo "  dev                - Run the application with air for hot-reloading"
	@echo "  clean              - Clean build artifacts"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests"
	@echo "  test-integration   - Run integration tests"
	@echo "  migrate-up         - Apply all database migrations"
	@echo "  migrate-down       - Rollback all database migrations"
	@echo "  sqlc               - Generate SQLC code"
	@echo "  docker-build       - Build Docker image using docker-compose"
	@echo "  docker-run         - Run Docker containers in detached mode"
	@echo "  docker-dev         - Run Docker containers for development with hot-reloading"
	@echo "  docker-stop        - Stop Docker containers"
