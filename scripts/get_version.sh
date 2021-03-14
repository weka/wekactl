#!/bin/bash

set -e

export BUILD_VERSION

BASE_VERSION=$(cat VERSION)
if [[ "$DEPLOY" == "1" ]]; then
  git tag -l | xargs git tag -d  # delete all local tags
  git fetch --tags  # fetch all remote tags
fi

if [[ "$GA" == "1" ]]; then
  if [[ -z $NEW_TAG ]]; then
    LATEST_TAG=$(git tag -l | grep "$BASE_VERSION" | grep -v "-" | tail -1 || true)
    if [[ -z $LATEST_TAG ]]; then
      patch=0
    else
      patch=$(echo "$LATEST_TAG" | cut -d '.' -f 3)
      patch=$((patch+1))
    fi
    BUILD_VERSION="$BASE_VERSION.$patch"
  else
    BUILD_VERSION="$NEW_TAG"
  fi
  if [ "$(git tag -l "$BUILD_VERSION")" ]; then
    echo "Error! tag $BUILD_VERSION already exists"
    exit 1
  fi
else
  LATEST_TAG=$(git describe --tags --exclude "*-*" --match "$BASE_VERSION.[0-9]*" --abbrev=0 2>/dev/null || true)
  LATEST_TAG="${LATEST_TAG:-$BASE_VERSION.0}"
  BUILD_VERSION="$LATEST_TAG-$(git rev-parse HEAD)"
fi
echo "$BUILD_VERSION"
