#!/bin/bash

. ./scripts/get_lambda_id.sh

AWS_DIST="internal/aws/dist/dist_generated.go"
docker build -t wekactl-deploy . \
--build-arg WEKACTL_AWS_LAMBDAS_BUCKETS="$WEKACTL_AWS_LAMBDAS_BUCKETS" \
--build-arg LAMBDAS_ID="$LAMBDAS_ID" \
--build-arg AWS_DIST="$AWS_DIST"

container_id=$(docker create wekactl-deploy:latest)
rm -rf tmp
docker cp "$container_id":/src/tmp tmp
docker rm -v "$container_id"

./scripts/distribute.sh "$LAMBDAS_ID"
