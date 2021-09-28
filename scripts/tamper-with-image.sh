#!/usr/bin/env bash 
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="$SCRIPTDIR/.."
source $BASEDIR/env

pushd "$BASEDIR/demo-image"

# Put the latest date into the demo image in order to force signature update
echo EEEEEVIL >  date

# Build the demo image
docker build -t ${DOCKER_USER}/demo-image .

echo "Sign into Docker Hub (may be cached)"
docker login
docker push ${DOCKER_USER}/demo-image:latest
popd

sleep 2

cd $BASEDIR
COSIGN_EXPERIMENTAL=1 bin/cosign verify docker.io/${DOCKER_USER}/demo-image:latest
