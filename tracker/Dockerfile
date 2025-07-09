# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files from tracker directory
COPY tracker/go.mod tracker/go.sum ./
RUN go mod download

# Copy the tracker source code
COPY tracker ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/tracker-server ./main.go

# Stage 2: Create a minimal final image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Security-related metadata
LABEL org.opencontainers.image.source=https://github.com/ShreyamKundu/peernet

# Add a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser

# Copy the binary from the builder stage
COPY --from=builder /app/tracker-server .

# Change ownership and switch to non-root user
RUN chown appuser:appgroup tracker-server
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./tracker-server"]