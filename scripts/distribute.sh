#!/bin/bash

set -e

if [[ $(git status --porcelain) != "" || "$WEKACTL_FORCE_DEV" == "1" ]]; then
  if [[ "$WEKACTL_IGNORE_DIRTY" == "1" ]]; then
    echo "Using random instead of hash for lambdas identifier"
    LAMBDAS_ID=dev/$(uuidgen)
  else
    echo "Refusing to build lambdas on dirty repository, use WEKACTL_IGNORE_DIRTY=1 to ignore"
    exit 1
  fi
else
  LAMBDAS_ID=release/$(git rev-parse HEAD)
fi

echo "Building lambdas with ID: $LAMBDAS_ID"
AWS_DIST="internal/aws/dist/dist_generated.go"
rm -f $AWS_DIST

distribute () {
  ZIP_PATH=$1
  if [[ -d "$ZIP_PATH" ]]; then
      recursive="--recursive"
  fi
  first_target=""
  echo "Distributing to AWS regions"
  echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | tr ',' '\n' | while read -r awspair; do
    echo "pair:" $awspair
  IFS="=" read -r region bucket <<< "$awspair"
  if [[ "$first_target" == "" ]]; then
    first_target="s3://$bucket/$LAMBDAS_ID/"
    aws s3 cp --region "$region" "$ZIP_PATH" "$first_target" --acl public-read $recursive
    first_region=$region
  else
    aws s3 cp --region "$region" --source-region "$first_region" "$first_target" s3://"$bucket"/"$LAMBDAS_ID"/ --acl public-read $recursive
  fi
  done
}

if [[ -n $WEKACTL_AWS_LAMBDAS_BUCKETS ]]; then
  if [[ -z $WEKACTL_SKIP_GO_LAMBDA ]]; then
    if [[ "$DEPLOY" == "1" ]]; then
      distribute tmp/
      docker run -it wekactl-deploy:latest go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" $LAMBDAS_ID $AWS_DIST
    else
      distribute tmp/wekactl-aws-lambdas.zip
      go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" $LAMBDAS_ID $AWS_DIST
    fi
  fi
fi
