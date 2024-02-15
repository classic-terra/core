#!/bin/bash

source $(dirname $0)/common.sh

# Filtering by containers belonging to the "terra-testnet" network
LIST_CONTAINERS_CMD=$(docker ps -a --filter network=terra-testnet --format {{.ID}})
LIST_NETWORKS_CMD=$(docker network ls --filter name=terra-testnet --format {{.ID}})

if [[ "$LIST_CONTAINERS_CMD" != "" ]]; then
    echo "Removing stale e2e containers"
    docker container rm -f $LIST_CONTAINERS_CMD
else
    echo "No stale e2e containers found"
fi

if [[ "$LIST_NETWORKS_CMD" != "" ]]; then
    echo "Removing stale e2e networks"
    docker network rm $LIST_NETWORKS_CMD
else
    echo "No stale e2e networks found"
fi

local_git_sha=$(git rev-parse HEAD)
for cur_image_sha in $LIST_DOCKER_IMAGE_HASHES; do
    if [[ "$cur_image_sha" != "$local_git_sha" ]]; then
        echo "Removing stale e2e image with SHA $cur_image_sha"
        docker rmi -f $(docker images --filter=reference="terra:$cur_image_sha" -q)
    fi
done
