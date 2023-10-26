#!/bin/bash

set -e

LAMBDAS_ID=$1
BUILD_VERSION=$2 # only for GA

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
    first_target="s3://$bucket/$LAMBDAS_ID"
    aws s3 cp --region "$region" "$ZIP_PATH" "$first_target/" --acl public-read $recursive
    first_region=$region
  else
    aws s3 cp --region "$region" --source-region "$first_region" "$first_target/wekactl-aws-lambdas.zip" s3://"$bucket"/"$LAMBDAS_ID"/wekactl-aws-lambdas.zip --acl public-read
  fi
  done
}

# use filenames created in build_wekactl.sh
filenames_arr=("wekactl_linux_amd64" "wekactl_darwin_amd64" "wekactl_linux_arm64" "wekactl_darwin_arm64")

if [[ -n $WEKACTL_AWS_LAMBDAS_BUCKETS ]]; then
  if [[ -z $WEKACTL_SKIP_GO_LAMBDA ]]; then
    region=$(echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | cut -d ',' -f 1 | cut -d '=' -f 1)
    bucket=$(echo "$WEKACTL_AWS_LAMBDAS_BUCKETS" | cut -d ',' -f 1 | cut -d '=' -f 2)
    if [[ "$DEPLOY" == "1" ]]; then
      distribute tmp/upload

      distributions=()
      for filename in "${filenames_arr[@]}"; do
        distribution="https://$bucket.s3.$region.amazonaws.com/$LAMBDAS_ID/$filename"
        distributions+=("$distribution")
        echo "wekactl $filename url: $distribution"
      done

      if [[ "$GA" == "1" ]]; then
        ./scripts/create_release.sh "$BUILD_VERSION" "${distributions[@]}"
      fi
    else
      distribute tmp/upload/wekactl-aws-lambdas.zip
    fi
    echo "lambdas url: https://$bucket.s3.$region.amazonaws.com/$LAMBDAS_ID/wekactl-aws-lambdas.zip"
  fi
fi
