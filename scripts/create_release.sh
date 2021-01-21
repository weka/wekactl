#!/bin/bash

set -e

wekactl_linux=$1
wekactl_darwin=$2

git config --global user.email "botty@weka.io"
git config --global user.name "Wekabot"

if [[ -z $NEW_TAG ]]; then
  git tag -l | xargs git tag -d
  git fetch --tags
  latest_tag=$(git describe --tags "$(git rev-list --tags --max-count=1)" || echo "")
  latest_tag=${latest_tag:-'0.0.0'}
  version_base=$(echo "$latest_tag" | cut -d '.' -f 1,2)
  patch=$(echo "$latest_tag" | cut -d '.' -f 3)
  patch=$((patch+1))
  NEW_TAG="$version_base.$patch"
fi

git tag -a "$NEW_TAG" -m "GA release" -m "$wekactl_linux" -m "$wekactl_darwin"
git push --tags
