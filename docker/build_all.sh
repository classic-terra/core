#!/bin/bash

VERSION="${1:-a8bc017fcb10cf0cc55e4b0036e7a1bf7ef0ad1b}"

pushd .. 

git checkout $VERSION
docker build -t classic-terra/terraclassic.terrad-binary:$VERSION .
git checkout -

popd

docker build --build-arg version=$VERSION --build-arg chainid=columbus-5 -t classic-terra/terraclassic.terrad-node:$VERSION-columbus-5 .
docker build --build-arg version=$VERSION --build-arg chainid=rebel-1    -t classic-terra/terraclassic.terrad-node:$VERSION-rebel-1    .
docker build --build-arg version=$VERSION --build-arg chainid=rebel-2    -t classic-terra/terraclassic.terrad-node:$VERSION-rebel-2    .

