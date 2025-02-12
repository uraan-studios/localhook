# Use official Golang image as a build stage
FROM golang:latest AS builder

# Set working directory
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Go application
RUN go build -o bin/sshtunnel

# Use a minimal base image for running the application
FROM debian:bookworm-slim

# Set working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/sshtunnel ./sshtunnel

# Expose necessary ports
EXPOSE 5000 2222

# Expose the additional ports from 49152 to 65535
EXPOSE 49152-65535

# Set the default command to run the application
CMD ["./sshtunnel", "-domain", "live.localhook.online", "-httpPort", "5000", "-sshPort", "2222"]