#!/bin/bash

set -e

WEKACTL_LINUX=$1
WEKACTL_DARWIN=$2
BUILD_VERSION=$3

git config --global user.email "botty@weka.io"
git config --global user.name "Wekabot"

git tag -a "$BUILD_VERSION" -m "GA release" -m "$WEKACTL_LINUX" -m "$WEKACTL_DARWIN"
git push --tags
