# Use Go official image as a builder
FROM golang:1.22 AS builder

# Set the working directory
WORKDIR /app

# Copy the application source code
COPY . .

# Build the Go application for specified architecture
ARG GOOS
ARG GOARCH
RUN GOOS=$GOOS GOARCH=$GOARCH go build -ldflags '-s -w' -o client client/*.go

# Set the working directory
FROM ubuntu:24.04

WORKDIR /app

# Copy the compiled binary from the builder
COPY --from=builder /app/client .

# Set environment variables
ENV DOMAIN=oracle.labkita.my.id
ENV DOMAIN_EVENT=oracle.labkita.my.id:4321

# Run the compiled server
CMD ["/client", "tcp", "-p", "5432", "-t", "5433" ,"-s", "pgsql"]