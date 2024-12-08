# Use Go official image as a builder
FROM golang:1.22 as builder

# Set the working directory
WORKDIR /app

# Copy the application source code
COPY . .

# Build the Go application for specified architecture
ARG GOOS
ARG GOARCH
RUN GOOS=$GOOS GOARCH=$GOARCH go build -ldflags '-s -w' -o server server/*.go

# Create a minimal runtime image
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the compiled binary from the builder
COPY --from=builder /app/server .

# Set environment variables
ENV DOMAIN=labstack.myaddr.io
ENV DOMAIN_EVENT=labstack.myaddr.io:4321

# Expose default application port
EXPOSE 8080

# Run the compiled server
CMD ["./server"]
