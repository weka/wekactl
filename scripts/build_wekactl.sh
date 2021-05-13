#!/bin/bash

BUILD_VERSION=$1
COMMIT=$2
flags_path="wekactl/internal/env"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X $flags_path.BuildVersion=$BUILD_VERSION -X $flags_path.Commit=$COMMIT" -o tmp/upload/wekactl_linux_amd64 cmd/wekactl/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-X $flags_path.BuildVersion=$BUILD_VERSION -X $flags_path.Commit=$COMMIT" -o tmp/upload/wekactl_darwin_amd64 cmd/wekactl/*.go
