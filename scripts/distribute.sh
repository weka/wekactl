#!/bin/bash

set -e

if [[ "$WEKACTL_FORCE_DEV" == "1" ]]; then
  echo "Using random instead of hash for lambdas identifier"
  LAMBDAS_ID=dev/$(uuidgen)
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
    echo "pair:" "$awspair"
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
    region=$(echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | cut -d ',' -f 1 | cut -d '=' -f 1)
    bucket=$(echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | cut -d ',' -f 1 | cut -d '=' -f 2)
    if [[ "$DEPLOY" == "1" ]]; then
      distribute tmp/upload
      container_id=$(docker run -dit wekactl-deploy:latest go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" "$LAMBDAS_ID" "$AWS_DIST")
      docker wait "$container_id"
      docker cp "$container_id:/src/$AWS_DIST" "$AWS_DIST"
      cat $AWS_DIST
      wekactl_linux="https://$bucket.s3.$region.amazonaws.com/$LAMBDAS_ID/wekactl_linux_amd64"
      wekactl_darwin="https://$bucket.s3.$region.amazonaws.com/$LAMBDAS_ID/wekactl_darwin_amd64"
      echo "wekactl linux url: $wekactl_linux"
      echo "wekactl darwin url: $wekactl_darwin"
      if [[ "$GA" == "1" ]]; then
        ./scripts/create_release.sh "$wekactl_linux" "$wekactl_darwin"
      fi
    else
      distribute tmp/upload/wekactl-aws-lambdas.zip
      go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" "$LAMBDAS_ID" "$AWS_DIST"
    fi
    echo "lambdas url: https://$bucket.s3.$region.amazonaws.com/$LAMBDAS_ID/wekactl-aws-lambdas.zip"
  fi
fi
