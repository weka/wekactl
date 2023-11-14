#!/bin/bash

set -e

BUILD_VERSION="$1"
DISTRIBUTIONS=("$@")
DISTRIBUTIONS=("${DISTRIBUTIONS[@]:1}") # removed the 1st element (BUILD_VERSION)

git checkout master
git pull
git tag "$BUILD_VERSION"
git push --set-upstream origin master
git push --tags

github_repo="weka/wekactl"

filepaths_arr=()
for distribution in "${DISTRIBUTIONS[@]}"; do
  filename=$(basename "$distribution")
  filepaths_arr+=("tmp/upload/$filename")
done

filepaths=$(printf "%s " "${filepaths_arr[@]}")

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
echo "Fetched github token"
AUTH="Authorization: token $GITHUB_TOKEN"
dist_str=$(printf '\\n%s' "${DISTRIBUTIONS[@]}")
release_body="GA release$dist_str"
echo "Creating draft release $BUILD_VERSION with body: $release_body"
result=$(curl \
  -X POST \
  -H "$AUTH" \
  https://api.github.com/repos/$github_repo/releases \
  -d "{\"tag_name\":\"$BUILD_VERSION\", \"name\":\"$BUILD_VERSION\", \"body\":\"$release_body\", \"draft\": true}")

id=$(echo "$result" | jq -c ".id")

for filepath in $filepaths; do
  curl \
    -H "$AUTH" \
    -H "Content-Type: $(file -b --mime-type "$filepath")" \
    --data-binary @"$filepath" \
    "https://uploads.github.com/repos/$github_repo/releases/$id/assets?name=$(basename "$filepath")"
done
