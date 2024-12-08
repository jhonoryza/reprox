#! /bin/bash

# linux
GOOS=linux      GOARCH=386      go build -ldflags '-s' -o bin/client-linux-386          client/*.go
GOOS=linux      GOARCH=amd64    go build -ldflags '-s' -o bin/client-linux-amd64        client/*.go
GOOS=linux      GOARCH=arm      go build -ldflags '-s' -o bin/client-linux-arm          client/*.go
GOOS=linux      GOARCH=arm64    go build -ldflags '-s' -o bin/client-linux-arm64        client/*.go

# mac
GOOS=darwin     GOARCH=arm64    go build -ldflags '-s' -o bin/client-darwin-arm64       client/*.go
GOOS=darwin     GOARCH=amd64    go build -ldflags '-s' -o bin/client-darwin-amd64       client/*.go

# windows
GOOS=windows    GOARCH=386      go build -ldflags '-s' -o bin/client-windows-386.exe    client/*.go
GOOS=windows    GOARCH=amd64    go build -ldflags '-s' -o bin/client-windows-amd64.exe  client/*.go