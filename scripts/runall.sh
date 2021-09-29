#!/usr/bin/env bash
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="$SCRIPTDIR/.."
source $BASEDIR/env

pushd ${BASEDIR}
bash scripts/cleanup.sh
sleep 1
bash scripts/0-check-requirements.sh
bash scripts/1-install-binaries.sh
bash scripts/2-build-and-sign-image.sh
bash scripts/3-run-spire.sh
sleep 5
bash scripts/4-attest.sh

