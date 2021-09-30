#!/usr/bin/env bash
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="${SCRIPTDIR}/.."
source ${BASEDIR}/env

# Run the demo image
# The command spire-agent api fetch x509 -socketPath /sock/agent.sock is 
# baked into the image. No conf file is needed for this command.
docker run --mount "type=bind,src="$(pwd)"/sock,dst=/sock" ${DOCKER_USER}/demo-image:latest

