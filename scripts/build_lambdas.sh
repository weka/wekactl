#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o tmp/lambdas-bin cmd/wekactl-aws-lambdas/*.go
cd tmp || exit 1
zip wekactl-aws-lambdas.zip lambdas-bin
rm lambdas-bin
