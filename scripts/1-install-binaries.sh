#!/usr/bin/env bash 
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="${SCRIPTDIR}/.."
SPIRE_URL="https://github.com/spiffe/spire/releases/download/v1.0.1/spire-1.0.1-linux-x86_64-glibc.tar.gz"
VERSION="spire-1.0.1"
COSIGN_URL="https://github.com/sigstore/cosign/releases/download/v1.2.0/cosign-linux-amd64"
REKOR_URL="https:////github.com/sigstore/rekor/releases/download/v0.3.0/rekor-cli-linux-amd64"

# Install SPIRE binaries
curl -L "${SPIRE_URL}" > tmp/spire.tgz
pushd "${BASEDIR}/tmp"
tar xvzf spire.tgz
popd
cp "${BASEDIR}/tmp/$VERSION/bin/spire-agent" "${BASEDIR}/bin/"
cp "${BASEDIR}/tmp/$VERSION/bin/spire-server" "${BASEDIR}/bin"

# Install Cosign
pushd "${BASEDIR}/bin"
curl -L ${COSIGN_URL} > cosign
chmod +x cosign
popd

# Install Rekor
pushd "${BASEDIR}/bin"
curl https://github.com/sigstore/rekor/releases/download/v0.3.0/rekor-cli-linux-amd64 > rekor-cli
chmod +x rekor-cli
popd
