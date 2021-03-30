#!/bin/bash

set -e
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"
cd ../

. ./scripts/get_build_params.sh

AWS_DIST="internal/aws/dist/dist_generated.go"
rm -f $AWS_DIST
go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" "$LAMBDAS_ID" "$AWS_DIST"

./scripts/build_lambdas.sh

./scripts/distribute.sh "$LAMBDAS_ID"
