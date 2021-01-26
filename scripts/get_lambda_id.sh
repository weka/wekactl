#!/bin/bash

set -e

export LAMBDAS_ID

if [[ "$WEKACTL_FORCE_DEV" == "1" ]]; then
  echo "Using random instead of hash for lambdas identifier"
  LAMBDAS_ID=dev/$(uuidgen)
else
  if [[ $(git status --porcelain) != "" ]]; then
    echo "Refusing to build lambdas on dirty repository, use WEKACTL_FORCE_DEV=1 to ignore"
    exit 1
  else
    LAMBDAS_ID=release/$(git rev-parse HEAD)
  fi
fi
