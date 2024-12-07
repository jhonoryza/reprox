#! /bin/bash

GOOS=linux GOARCH=386 go build -ldflags '-s' -o bin/client-linux client/*.go
GOOS=darwin GOARCH=arm64 go build -ldflags '-s' -o bin/client-mac client/*.go