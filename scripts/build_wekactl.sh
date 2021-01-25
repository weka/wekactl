#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tmp/upload/wekactl_linux_amd64 cmd/wekactl/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o tmp/upload/wekactl_darwin_amd64 cmd/wekactl/*.go
