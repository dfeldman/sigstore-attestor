#!/usr/bin/env bash 
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="$SCRIPTDIR/.."
source $BASEDIR/env

# Remove any old demo image
docker rmi demo-image

pushd "$BASEDIR/demo-image"

# Put the latest date into the demo image in order to force signature update
echo $(date) >  date

# Build the demo image
docker build -t ${DOCKER_USER}/demo-image

echo "Sign into Docker Hub (may be cached)"
docker login
docker push ${DOCKER_USER}/demo-image:latest
popd

# Sign the demo image. This is based off a tutorial from here: 
# https://dev.to/martinheinz/signing-software-the-easy-way-with-sigstore-and-cosign-kde
COSIGN_EXPERIMENTAL=1 bin/cosign sign \
-oidc-issuer "https://oauth2.sigstore.dev/auth" \
-fulcio-url "https://fulcio.sigstore.dev" \
-rekor-url "https://rekor.sigstore.dev"  \
docker.io/${DOCKER_USER}/demo-image

COSIGN_EXPERIMENTAL=1 bin/cosign verify docker.io/${DOCKER_USER}/demo-image:latest
