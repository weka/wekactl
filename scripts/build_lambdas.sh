#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o tmp/upload/bootstrap cmd/wekactl-aws-lambdas/*.go
cd tmp/upload || exit 1
zip wekactl-aws-lambdas.zip bootstrap
rm bootstrap
