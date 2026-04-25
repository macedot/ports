# syntax=docker/dockerfile:1

# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod files first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO disabled for static binary
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o /app/server .

# Runtime stage
FROM gcr.io/distroless/static-debian12

# The container needs access to /proc/net/* for socket parsing.
# Run with --pid=host to access host PID namespace.
#
# Example:
#   docker run --pid=host listen-ports:latest

ENV PORT=8080

EXPOSE 8080

COPY --from=builder /app/server /app/server

ENTRYPOINT ["/app/server"]
