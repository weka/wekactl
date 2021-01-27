#!/bin/bash

BUILD_VERSION=$1
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X wekactl/internal/cli/version.BuildVersion=$BUILD_VERSION" -o tmp/upload/wekactl_linux_amd64 cmd/wekactl/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-X wekactl/internal/cli/version.BuildVersion=$BUILD_VERSION" -o tmp/upload/wekactl_darwin_amd64 cmd/wekactl/*.go
