#!/usr/bin/env bash 
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="$SCRIPTDIR/.."
source $BASEDIR/env

# Remove any old demo image
docker rmi demo-image || echo "Couldn't remove old image"

pushd "$BASEDIR/demo-image"

# Put the latest date into the demo image in order to force signature update
echo $(date) >  date

# Build the demo image
docker build -t ${DOCKER_USER}/demo-image .

echo "Sign into Docker Hub (may be cached)"
docker login
docker push ${DOCKER_USER}/demo-image:latest
popd

sleep 2

rm cosign.key cosign.pub
# Sign the demo image. This is based on https://github.com/sigstore/cosign 
bin/cosign generate-key-pair

bin/cosign sign \
  -key cosign.key \
  docker.io/${DOCKER_USER}/demo-image

bin/cosign verify -key cosign.pub docker.io/${DOCKER_USER}/demo-image:latest
