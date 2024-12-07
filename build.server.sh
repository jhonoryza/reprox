#! /bin/bash

GOOS=linux GOARCH=386 go build -ldflags '-s' -o bin/server-linux server/*.go
GOOS=darwin GOARCH=arm64 go build -ldflags '-s' -o bin/server-mac server/*.go