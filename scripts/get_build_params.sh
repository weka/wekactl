#!/bin/bash

set -e

export BUILD_VERSION
export COMMIT
export LAMBDAS_ID

if [[ $(git status --porcelain) != "" && "$WEKACTL_FORCE_DEV" != "1" ]]; then
  echo "Refusing to build lambdas on dirty repository, use WEKACTL_FORCE_DEV=1 to ignore"
  exit 1
fi

COMMIT="$(git rev-parse HEAD)"
BASE_VERSION=$(cat VERSION)

if [[ "$WEKACTL_FORCE_DEV" == "1" ]]; then
  LATEST_TAG=$(git describe --tags --exclude "*-*" --match "$BASE_VERSION.[0-9]*" --abbrev=0 2>/dev/null || true)
  LATEST_TAG="${LATEST_TAG:-$BASE_VERSION.0}"
  if [[ $(git status --porcelain) == "" ]]; then
    BUILD_VERSION="$LATEST_TAG-$COMMIT"
  else
    COMMIT="$COMMIT-dirty"
    BUILD_VERSION="$LATEST_TAG-$(uuidgen)"
  fi
    LAMBDAS_ID="dev/$BUILD_VERSION"
fi

if [[ "$DEPLOY" == "1" ]]; then
  git tag -l | xargs git tag -d  # delete all local tags
  git fetch --tags  # fetch all remote tags
fi

if [[ "$GA" == "1" ]]; then
  LATEST_TAG=$(git tag -l | grep "$BASE_VERSION" | grep -v "-" | tail -1 || true)
  if [[ -z $LATEST_TAG ]]; then
    patch=0
  else
    patch=$(echo "$LATEST_TAG" | cut -d '.' -f 3)
    patch=$((patch+1))
  fi
  BUILD_VERSION="$BASE_VERSION.$patch"
  if [ "$(git tag -l "$BUILD_VERSION")" ]; then
    echo "Error! tag $BUILD_VERSION already exists"
    exit 1
  fi
  LAMBDAS_ID="release/$BUILD_VERSION"
fi

if [[ "$RELEASE" == 1 ]]; then
  if [[ -z $VERSION ]]; then
    echo "You mush supply 'VERSION' environment variable in order to build a release"
    exit 1
  fi
  BUILD_VERSION="$VERSION"
  LAMBDAS_ID="release/$BUILD_VERSION"
  if aws s3 ls "s3://weka-wekactl-images-eu-west-1/$LAMBDAS_ID" > /dev/null 2>&1 ; then
	  echo "Release with VERSION=$VERSION already exists!"
	  exit 1
  fi
fi

echo "$BUILD_VERSION"
echo "$COMMIT"