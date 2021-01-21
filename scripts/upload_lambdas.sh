#!/bin/bash

set -e
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"
cd ../

if [[ ($(git status --porcelain) != "" || "$WEKACTL_FORCE_DEV" == "1") && "$WEKACTL_IGNORE_DIRTY" != "1" ]]; then
  echo "Refusing to build lambdas on dirty repository, use WEKACTL_IGNORE_DIRTY=1 to ignore"
  exit 1
fi

chmod +x ./scripts/build_lambdas.sh
./scripts/build_lambdas.sh

chmod +x ./scripts/distribute.sh
./scripts/distribute.sh
