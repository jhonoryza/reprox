# Use Go official image as a builder
FROM golang:1.22 AS builder

# Set the working directory
WORKDIR /app

# Copy the application source code
COPY . .

# Build the Go application for specified architecture
ARG GOOS
ARG GOARCH
RUN GOOS=$GOOS GOARCH=$GOARCH go build -ldflags '-s -w' -o client-cli client/*.go

# Set the working directory
FROM ubuntu:24.04

WORKDIR /app

# Copy the compiled binary from the builder
COPY --from=builder /app/client-cli .

# Run the compiled server
CMD ["./client-cli", "tcp", "-p", "5432", "-t", "5433" ,"-s", "pgsql"]