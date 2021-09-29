#!/usr/bin/env bash
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="$SCRIPTDIR/.."
source $BASEDIR/env

pushd ${BASEDIR}
if [[ -f tmp/spire-server.pid ]]; then
        (kill $(cat tmp/spire-server.pid) || echo "Couldn't kill spire-server")
        rm tmp/spire-server.pid
fi

if [[ -f tmp/spire-agent.pid ]]; then
        (kill $(cat tmp/spire-agent.pid) || echo "Couldn't kill spire-agent")
        rm tmp/spire-agent.pid
fi

rm -rv ./data/* || echo "Unable to remove data files"
rm -rv ./tmp/* || echo "Unable to remove temp files"
rm -rv ./sock/* || echo "Unable to remove sock files"
docker rmi --force ${DOCKER_USER}/demo-image || echo "Unable to delete demo image"
