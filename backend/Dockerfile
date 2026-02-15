# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install goose for migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary, migrations, and goose
COPY --from=builder /build/server .
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /go/bin/goose /usr/local/bin/goose

EXPOSE 8080

# Run migrations then start server
CMD goose -dir ./migrations postgres \
    "host=${DB_HOST} port=${DB_PORT} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=disable" \
    up && ./server
