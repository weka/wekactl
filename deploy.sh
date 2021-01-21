#!/bin/bash

docker build -t wekactl-deploy .

container_id=$(docker create wekactl-deploy:latest)
rm -rf tmp
docker cp "$container_id":/src/tmp tmp
docker rm -v "$container_id"

chmod +x  scripts/distribute.sh
./scripts/distribute.sh
