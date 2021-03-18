#!/bin/bash

set -e

. ./scripts/get_build_params.sh

AWS_DIST="internal/aws/dist/dist_generated.go"
docker build -t wekactl-deploy . \
--build-arg WEKACTL_AWS_LAMBDAS_BUCKETS="$WEKACTL_AWS_LAMBDAS_BUCKETS" \
--build-arg LAMBDAS_ID="$LAMBDAS_ID" \
--build-arg AWS_DIST="$AWS_DIST" \
--build-arg BUILD_VERSION="$BUILD_VERSION" \
--build-arg COMMIT="$COMMIT"

container_id=$(docker create wekactl-deploy:latest)
rm -rf tmp
docker cp "$container_id":/src/tmp tmp
docker rm -v "$container_id"

./scripts/distribute.sh "$LAMBDAS_ID" "$BUILD_VERSION"
