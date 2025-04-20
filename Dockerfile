FROM golang:1.21-alpine as builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o HLSpresso cmd/transcoder/main.go

# Final stage
FROM alpine:3.18

# Install FFmpeg and other dependencies
RUN apk add --no-cache ffmpeg ca-certificates

# Copy binary from builder stage
COPY --from=builder /app/HLSpresso /usr/local/bin/

# Create directories for data
RUN mkdir -p /data/input /data/output /data/downloads

# Set working directory
WORKDIR /data

# Set volume for persistent data
VOLUME ["/data/input", "/data/output", "/data/downloads"]

# Command to run
ENTRYPOINT ["HLSpresso"]

# Default arguments
CMD ["--help"] 