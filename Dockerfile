# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Install swag CLI (version aligned with go.mod)
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6

# Copy source code
COPY . .

# Generate Swagger/OpenAPI docs from handler annotations
RUN swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal

# Build the application (embeds generated docs via clap/cmd/api/docs)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata ffmpeg

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy config files
COPY --from=builder /app/internal/shared/config/config.yaml ./config/

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
