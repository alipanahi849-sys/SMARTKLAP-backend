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
- ✅ User registration with password hashing
- ✅ User login with JWT tokens
- ✅ Access token & refresh token mechanism
- ✅ Token refresh endpoint
- ✅ Logout (single device & all devices)
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
docker-compose up -d
```

2. **Check service health**
```bash
docker-compose ps
```

3. **View logs**
```bash
docker-compose logs -f api
```

4. **Stop services**
```bash
docker-compose down
```

### Running Locally

1. **Start PostgreSQL & Redis**
```bash
docker-compose up -d postgres redis
```

2. **Run migrations**
```bash
make migrate
```

3. **Seed default roles**
```bash
make seed
```

4. **Run the application**
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
```

The API will be available at `http://localhost:8080`

## API Endpoints

### Health Check
```
GET /health
```

### Authentication
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout        (protected)
POST   /api/v1/auth/logout-all    (protected)
```

### Users
```
GET    /api/v1/users/me           (protected)
PUT    /api/v1/users/me           (protected)
```

### Profiles
```
GET    /api/v1/profiles/me        (protected)
POST   /api/v1/profiles/me        (protected)
PUT    /api/v1/profiles/me        (protected)
DELETE /api/v1/profiles/me        (protected)
```

## API Usage Examples

### Register a new user
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123"
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
- News feed
- E-commerce/shop functionality
- Advanced rate limiting
- Metrics and monitoring (Prometheus)
- Distributed tracing
- API documentation (Swagger/OpenAPI)

## License

[Your License Here]

## Support

For issues and questions, please open an issue on the repository.
