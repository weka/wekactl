#!/bin/bash

set -e

WEKACTL_LINUX=$1
WEKACTL_DARWIN=$2
BUILD_VERSION=$3

github_repo="weka/wekactl"
filenames="tmp/upload/wekactl_linux_amd64 tmp/upload/wekactl_darwin_amd64"

if [ -z "$DEPLOY_APP_ID" ]; then
  echo "You must supply DEPLOY_APP_ID environment variable !"
  exit 1
fi

if [ -z "$DEPLOY_APP_PRIVATE_KEY" ]; then
  echo "You must supply DEPLOY_APP_PRIVATE_KEY environment variable !"
  exit 1
fi

docker build -t github-token . -f scripts/python.Dockerfile
eval "$(docker run -e "DEPLOY_APP_ID=$DEPLOY_APP_ID" -e "DEPLOY_APP_PRIVATE_KEY=$DEPLOY_APP_PRIVATE_KEY" github-token)"

AUTH="Authorization: token $GITHUB_TOKEN"
release_body="GA release\n$WEKACTL_LINUX\n$WEKACTL_DARWIN"
result=$(curl \
  -X POST \
  -H "$AUTH" \
  https://api.github.com/repos/$github_repo/releases \
  -d "{\"tag_name\":\"$BUILD_VERSION\", \"name\":\"$BUILD_VERSION\", \"body\":\"$release_body\", \"draft\": true}")

id=$(echo "$result" | jq -c ".id")

for filename in $filenames; do
  curl \
    -H "$AUTH" \
    -H "Content-Type: $(file -b --mime-type "$filename")" \
    --data-binary @"$filename" \
    "https://uploads.github.com/repos/$github_repo/releases/$id/assets?name=$(basename "$filename")"
done
