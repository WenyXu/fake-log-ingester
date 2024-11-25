# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o fake-log-ingester

# Final stage
FROM alpine:3.19
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/fake-log-ingester .

ENTRYPOINT ["./fake-log-ingester"]