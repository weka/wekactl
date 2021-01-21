#!/bin/bash

if [[ $(git status --porcelain) != "" ]]; then
  echo "Refusing to build lambdas on dirty repository, use WEKACTL_IGNORE_DIRTY=1 to ignore"
  exit 1
fi

docker build -t wekactl-deploy .

container_id=$(docker create wekactl-deploy:latest)
rm -rf tmp
docker cp "$container_id":/src/tmp tmp
docker rm -v "$container_id"

./scripts/distribute.sh
