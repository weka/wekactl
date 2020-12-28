#!/bin/bash

set -e
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"
cd ../

if [[ $(git status --porcelain) != "" ]]; then
  if [[ "$WEKACTL_IGNORE_DIRTY" == "1" ]]; then
    echo "Using random instead of hash for lambdas identifier"
    LAMBDAS_ID=dev/$(uuidgen)
  else
    echo "Refusing to build lambdas on dirty repository, use WEKACTL_IGNORE_DIRTY=1 to ignore"
    exit 1
  fi
fi

if [[ "$LAMBDAS_ID" == "" ]]; then
  LAMBDAS_ID=release/$(git rev-parse HEAD)
fi


GOOS=linux GOARCH=amd64 go build -o tmp/lambdas-bin cmd/wekactl/*.go
cd tmp/
zip wekactl.zip lambdas-bin
cd -

echo "Building lambdas with ID: $LAMBDAS_ID"
AWS_DIST="internal/aws/dist/dist_generated.go"
rm -f $AWS_DIST

first_target=""
if [[ -n $WEKACTL_AWS_LAMBDAS_BUCKETS ]]; then
  echo "Distributing lambdas to AWS regions"
  echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | while IFS=, read -r awspair; do
  IFS="=" read -r region bucket <<< "$awspair"
  if [[ "$first_target" == "" ]]; then
    first_target="s3://$bucket/$LAMBDAS_ID/wekactl.zip"
    aws s3 cp --region $region tmp/wekactl.zip $first_target --acl public-read
    first_region=$region
  else
    aws s3 cp --region $region --source-region $first_region $first_target s3://$bucket/$LAMBDAS_ID/wekactl.zip --acl public-read
  fi
  done

  go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" lambda/id $AWS_DIST
fi

cd -
