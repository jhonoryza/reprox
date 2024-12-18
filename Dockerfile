# Use Go official image as a builder
FROM golang:1.22 AS builder

# Set the working directory
WORKDIR /app

# Copy the application source code
COPY . .

# Build the Go application for specified architecture
ARG GOOS
ARG GOARCH
RUN GOOS=$GOOS GOARCH=$GOARCH go build -ldflags '-s -w' -o reprox_server server/*.go

# Create a minimal runtime image
FROM ubuntu:24.04

# Set the working directory
WORKDIR /app/

# Copy the compiled binary from the builder
COPY --from=builder /app/reprox_server .

# Set environment variables
ENV DOMAIN=labstack.myaddr.io
ENV DOMAIN_EVENT=labstack.myaddr.io:4321

# Expose default application port
EXPOSE 80
EXPOSE 443
EXPOSE 4321

# Run the compiled server
CMD ["./reprox_server"]
