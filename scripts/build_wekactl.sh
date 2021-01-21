#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o tmp/wekactl cmd/wekactl/*.go
cd tmp || exit 1
zip wekactl.zip wekactl
rm wekactl
