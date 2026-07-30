.PHONY: help build run test clean docker-up docker-down migrate seed migrate-seed dev swagger

# Default target
help:
	@echo "Available commands:"
	@echo "  make build      - Build the application"
	@echo "  make run        - Run the application"
	@echo "  make dev        - Run with hot reload (air)"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make docker-up  - Start Docker Compose services"
	@echo "  make docker-down- Stop Docker Compose services"
	@echo "  make migrate      - Run database migrations"
	@echo "  make seed         - Seed default roles"
	@echo "  make migrate-seed - Migrate + seed (Docker does this automatically)"
	@echo "  make swagger      - Generate Swagger docs (swag)"

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/api ./cmd/api
	@echo "Build complete: bin/api"

# Run the application
run:
	@echo "Running application..."
	@go run ./cmd/api/main.go

# Run with hot reload
dev:
	@echo "Running with hot reload..."
	@air

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ tmp/
	@go clean
	@echo "Clean complete"

# Start Docker Compose services
docker-up:
	@echo "Starting Docker Compose services..."
	@docker-compose up -d
	@echo "Services started"

# Stop Docker Compose services
docker-down:
	@echo "Stopping Docker Compose services..."
	@docker-compose down
	@echo "Services stopped"

# Run database migrations (idempotent; tracks schema_migrations)
migrate:
	@echo "Running database migrations..."
	@if [ "$(OS)" = "Windows_NT" ]; then powershell -ExecutionPolicy Bypass -File scripts/run_migrations.ps1; else SKIP_SEED=1 sh scripts/migrate_and_seed.sh; fi

# Seed default roles (idempotent)
seed:
	@echo "Seeding default roles..."
	@if [ "$(OS)" = "Windows_NT" ]; then powershell -ExecutionPolicy Bypass -File scripts/seed_roles.ps1; else sh scripts/seed_roles.sh; fi

# Migrate + seed (same as Docker migrate service)
migrate-seed:
	@sh scripts/migrate_and_seed.sh

# Generate Swagger/OpenAPI docs
swagger:
	@echo "Generating Swagger docs..."
	@command -v swag >/dev/null 2>&1 || GOBIN="$$(go env GOPATH)/bin" go install github.com/swaggo/swag/cmd/swag@latest
	@PATH="$$(go env GOPATH)/bin:$$PATH" swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal
	@echo "Swagger docs generated in cmd/api/docs"
	@echo "UI available at http://localhost:8080/swagger/index.html after make run"
