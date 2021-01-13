#!/bin/bash

set -e
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"
cd ../

if [[ $(git status --porcelain) != "" || "$WEKACTL_FORCE_DEV" == "1" ]]; then
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


GOOS=linux GOARCH=amd64 go build -o tmp/lambdas-bin cmd/wekactl-aws-lambdas/*.go
cd tmp/
zip wekactl-aws-lambdas.zip lambdas-bin
cd -


echo "Building lambdas with ID: $LAMBDAS_ID"
AWS_DIST="internal/aws/dist/dist_generated.go"
rm -f $AWS_DIST

distribute () {
  ZIP_PATH=$1
  FILENAME=$(basename "$ZIP_PATH")
  first_target=""
  echo "Distributing lambdas to AWS regions"
  echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | while IFS=, read -r awspair; do
  IFS="=" read -r region bucket <<< "$awspair"
  if [[ "$first_target" == "" ]]; then
    first_target="s3://$bucket/$LAMBDAS_ID/$FILENAME"
    aws s3 cp --region "$region" "$ZIP_PATH" "$first_target" --acl public-read
    first_region=$region
  else
    aws s3 cp --region "$region" --source-region "$first_region" "$first_target" s3://"$bucket"/"$LAMBDAS_ID"/wekactl-aws-lambdas.zip --acl public-read
  fi
  done
}

if [[ -n $WEKACTL_AWS_LAMBDAS_BUCKETS ]]; then
  if [[ -z $WEKACTL_SKIP_GO_LAMBDA ]]; then
    distribute tmp/wekactl-aws-lambdas.zip
    go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" $LAMBDAS_ID $AWS_DIST
  fi
fi

cd -
