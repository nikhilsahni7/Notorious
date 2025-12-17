# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# Use CGO_ENABLED=0 for a static binary that works in Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS requests and tzdata for timezones
RUN apk --no-cache add ca-certificates tzdata


WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy migrations and templates (if any)
# Note: migrations are handled by the app on startup
COPY --from=builder /app/migrations ./migrations
# COPY --from=builder /app/templates ./templates

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./main"]
