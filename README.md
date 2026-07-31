# Clap Backend - Sports Fan Engagement Platform

A production-grade backend foundation for a large-scale realtime sports fan engagement platform built with Go, PostgreSQL, Redis, and Docker.

## Architecture

This project follows **Clean Architecture** and **Modular Monolith** principles, designed for scalability and future microservice extraction.

### Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL 16
- **Cache/Messaging**: Redis 7
- **ORM**: GORM
- **Authentication**: JWT (golang-jwt/jwt/v5)
- **Configuration**: Viper
- **Logging**: Zerolog
- **Containerization**: Docker & Docker Compose

### Project Structure

```
clap/
├── cmd/
│   └── api/                 # Application entry point
│       └── main.go
├── internal/
│   ├── modules/             # Business modules (modular monolith)
│   │   ├── auth/           # Authentication module
│   │   │   ├── models/
│   │   │   ├── repository/
│   │   │   ├── service/
│   │   │   ├── handler/
│   │   │   └── routes.go
│   │   ├── user/           # User profile module (example)
│   │   ├── club/           # Club management (placeholder)
│   │   ├── match/          # Match scheduling (placeholder)
│   │   ├── song/           # Song/anthem management (placeholder)
│   │   └── realtime/       # Realtime event interfaces
│   └── shared/             # Shared infrastructure
│       ├── config/         # Configuration management
│       ├── database/       # Database connection & base models
│       ├── redis/          # Redis connection layer
│       ├── logger/         # Structured logging
│       ├── middleware/     # HTTP middleware
│       ├── response/       # API response utilities
│       ├── errors/         # Error handling
│       ├── utils/          # JWT, password hashing
│       └── container/      # Dependency injection
├── pkg/
│   └── migrations/         # Database migrations
├── deployments/            # Deployment configurations
├── scripts/               # Utility scripts
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

## Features Implemented (Phase 1)

### Core Infrastructure
- ✅ Configuration management with environment variables
- ✅ Structured logging with Zerolog
- ✅ PostgreSQL connection with GORM
- ✅ Redis connection layer
- ✅ Base models with UUID, timestamps, soft delete
- ✅ Database migrations setup

### Authentication Module
- ✅ Passwordless register/login via email OTP
- ✅ OTP verify endpoint issuing JWT tokens
- ✅ Access token & refresh token mechanism
- ✅ Token refresh endpoint
- ✅ Role-based access control (RBAC)
- ✅ Protected routes with middleware

### API Foundation
- ✅ RESTful API with Gin framework
- ✅ API versioning (/api/v1)
- ✅ Standardized JSON response format
- ✅ Comprehensive error handling
- ✅ Request validation
- ✅ CORS middleware
- ✅ Request logging middleware
- ✅ Panic recovery middleware

### Realtime Preparation
- ✅ Realtime event interfaces
- ✅ Redis pub/sub implementation
- ✅ Extensible architecture for Centrifugo/NATS

### Example Module (User Profiles)
- ✅ Profile CRUD operations
- ✅ Repository pattern
- ✅ Service layer
- ✅ Handler with validation
- ✅ Route registration

### Docker & Deployment
- ✅ Multi-stage Dockerfile
- ✅ Docker Compose with PostgreSQL & Redis
- ✅ Health checks
- ✅ Environment configuration

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker & Docker Compose
- PostgreSQL client (optional)

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd clap
```

2. **Install Go dependencies**
```bash
go mod download
```

3. **Configure environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

### Running with Docker Compose (Recommended)

1. **Start all services**
```bash
docker compose up -d --build
```

This automatically:
- starts Postgres + Redis
- runs pending DB migrations (`migrate` service)
- seeds default roles
- starts the API (and nginx)

Swagger UI: `http://localhost:8081/swagger/index.html`

2. **Check service health**
```bash
docker compose ps
```

3. **View logs**
```bash
docker compose logs -f api
# migrate/seed logs:
docker compose logs migrate
```

4. **Stop services**
```bash
docker compose down
```

### Running Locally (API on host)

1. **Start PostgreSQL & Redis**
```bash
docker compose up -d postgres redis
```

2. **Migrate + seed** (only needed when the API runs on the host, not via Compose)
```bash
make migrate-seed
# or separately:
# make migrate
# make seed
```

3. **Run the application**
```bash
make run
# Or with hot reload:
make dev
```

The API will be available at `http://localhost:8080`

### Using Makefile

Available commands:
```bash
make help          # Show all available commands
make build         # Build the application
make run           # Run the application
make dev           # Run with hot reload (air)
make test          # Run tests
make clean         # Clean build artifacts
make docker-up     # Start Docker Compose services
make docker-down   # Stop Docker Compose services
make migrate       # Run database migrations
make seed          # Seed default roles
make migrate-seed  # Migrate + seed (automatic in Docker Compose)
```

The API will be available at `http://localhost:8080`

## API Endpoints

### Health Check
```
GET /health
```

### Authentication
```
POST   /api/v1/auth/register      # name + email → OTP
POST   /api/v1/auth/login         # email → OTP (call again to resend)
POST   /api/v1/auth/verify-otp    # email + code → tokens
POST   /api/v1/auth/refresh
```



### Users
```
GET    /api/v1/users/me           (protected)
PUT    /api/v1/users/me           (protected)
```

### Profiles
```
GET    /api/v1/profile/me                 (protected)
PATCH  /api/v1/profile/me                 (protected)
DELETE /api/v1/profile/me                 (protected)
POST   /api/v1/profile/me/avatar          (protected)
GET    /api/v1/profile/leaderboard        (protected)
```

## API Usage Examples

### Register (sends OTP)
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "user@example.com"
  }'
```

### Login (sends OTP — call again to resend)
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

### Verify OTP
```bash
curl -X POST http://localhost:8080/api/v1/auth/verify-otp \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "code": "1234"
  }'
```

### Access protected route
```bash
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Environment (development/production) | development |
| `SERVER_PORT` | Server port | 8080 |
| `SERVER_MODE` | Gin mode (debug/release) | debug |
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_USER` | PostgreSQL user | postgres |
| `DB_PASSWORD` | PostgreSQL password | - |
| `DB_NAME` | Database name | clap |
| `DB_SSL_MODE` | SSL mode | disable |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PORT` | Redis port | 6379 |
| `REDIS_PASSWORD` | Redis password | - |
| `JWT_SECRET` | JWT signing secret | - |
| `JWT_ACCESS_EXPIRY` | Access token expiry (seconds) | 900 |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry (seconds) | 604800 |
| `CORS_ALLOWED_ORIGINS` | CORS allowed origins | * |
| `SMTP_HOST` | SMTP host (MailHog in Docker: `mailhog`) | localhost |
| `SMTP_PORT` | SMTP port (MailHog: `1025`) | 1025 |
| `SMTP_USERNAME` | SMTP username (empty for MailHog) | - |
| `SMTP_PASSWORD` | SMTP password (empty for MailHog) | - |
| `SMTP_FROM` | From email address | noreply@clap.local |
| `SMTP_FROM_NAME` | From display name | Clap |
| `SMTP_USE_TLS` | STARTTLS (`false` for MailHog) | false |

### MailHog (local OTP inbox)

Compose starts MailHog automatically:
- SMTP: `localhost:1025` (host) / `mailhog:1025` (API container)
- Web UI: [http://localhost:8025](http://localhost:8025)

```bash
docker compose up -d --build
make test-smtp TO=test@clap.local
# then open http://localhost:8025
```

## Development

### Adding a New Module

1. Create module structure under `internal/modules/<module-name>/`:
```
internal/modules/<module-name>/
├── models/
├── repository/
├── service/
├── handler/
└── routes.go
```

2. Implement repository interface
3. Implement service layer
4. Implement handlers
5. Register routes in `routes.go`
6. Import and register in `cmd/api/main.go`

### Database Migrations

Migration files are located in `pkg/migrations/`:
- `001_init_schema.up.sql` - Apply schema
- `001_init_schema.down.sql` - Rollback schema

### Swagger / OpenAPI

- **Docker:** docs are generated automatically during image build (`Dockerfile` runs `swag init`)
- Local regenerate: `make swagger`
- UI (non-production), open via nginx so Try it out hits the same host:
  - Local: [http://localhost:8081/swagger/index.html](http://localhost:8081/swagger/index.html)
  - Server: `http://<SERVER_IP>:8081/swagger/index.html`
  - Do not use `localhost:8080` in the browser when the API runs on a remote Docker host
- Spec files: `cmd/api/docs/`
- Auth: use **Authorize** with `Bearer <access_token>`
- Rebuild after annotation changes: `docker compose build api --no-cache` (or `docker compose up -d --build`)

### Code Quality

- Follow Go best practices and idiomatic patterns
- Use interfaces for dependency injection
- Keep functions small and focused
- Add comprehensive error handling
- Write tests for business logic

## Security Considerations

- JWT secrets must be changed in production
- Use strong passwords for database
- Enable SSL in production environments
- Configure CORS appropriately for your domain
- Implement rate limiting for production
- Use environment variables for sensitive data

## Future Enhancements (Beyond Phase 1)

- WebSocket integration for realtime features
- Centrifugo or NATS for scalable pub/sub
- Match scheduling and management
- Anthem synchronization
- Lyrics synchronization
- Flash/vibration events
- Club management panels
- E-commerce/shop functionality
- Advanced rate limiting
- Metrics and monitoring (Prometheus)
- Distributed tracing
- Richer per-endpoint response schemas in Swagger

## License

[Your License Here]

## Support

For issues and questions, please open an issue on the repository.
