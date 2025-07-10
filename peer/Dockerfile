# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the proto files first (less likely to change)
COPY proto ./proto

# Copy go.mod and go.sum files
COPY peer/go.mod peer/go.sum ./
RUN go mod download

# Copy the peer source code (all subdirectories: cli, config, file, p2p)
COPY peer ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/peer-client ./main.go

# Stage 2: Create a minimal final image
FROM alpine:latest

# Install ca-certificates for HTTPS/TLS requests
RUN apk --no-cache add ca-certificates

# Security-related metadata
LABEL org.opencontainers.image.source="https://github.com/ShreyamKundu/peernet"

# Add a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser

# Copy the binary from the builder stage
COPY --from=builder /app/peer-client .

# Create a directory for downloads and shared files
RUN mkdir data && chown appuser:appgroup data

# Change ownership of the binary and switch to non-root user
RUN chown appuser:appgroup peer-client
USER appuser

# # Set the entrypoint
# ENTRYPOINT ["./peer-client"]