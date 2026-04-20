# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binaries
# Using CGO_ENABLED=0 for static binaries that work in alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/order ./cmd/order/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/payment ./cmd/payment/main.go

# Stage 2: Final runtime stage
FROM alpine:latest AS runner

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binaries from the builder stage
COPY --from=builder /app/bin/order /app/order
COPY --from=builder /app/bin/payment /app/payment

# Default entrypoint (will be overridden by docker-compose)
ENTRYPOINT ["/app/order"]
