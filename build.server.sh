#! /bin/bash

# linux
GOOS=linux      GOARCH=386      go build -ldflags '-s' -o bin/server-linux-386          server/*.go
GOOS=linux      GOARCH=amd64    go build -ldflags '-s' -o bin/server-linux-amd64        server/*.go
GOOS=linux      GOARCH=arm      go build -ldflags '-s' -o bin/server-linux-arm          server/*.go
GOOS=linux      GOARCH=arm64    go build -ldflags '-s' -o bin/server-linux-arm64        server/*.go

# mac
GOOS=darwin     GOARCH=arm64    go build -ldflags '-s' -o bin/server-darwin-arm64       server/*.go
GOOS=darwin     GOARCH=amd64    go build -ldflags '-s' -o bin/server-darwin-amd64       server/*.go

# windows
GOOS=windows    GOARCH=386      go build -ldflags '-s' -o bin/server-windows-386.exe    server/*.go
GOOS=windows    GOARCH=amd64    go build -ldflags '-s' -o bin/server-windows-amd64.exe  server/*.go