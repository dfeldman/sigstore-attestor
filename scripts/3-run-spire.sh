#!/usr/bin/env bash
set -xeo pipefail
SCRIPTDIR=$(dirname "$0")
BASEDIR="${SCRIPTDIR}/.."
source ${BASEDIR}/env

pushd ${BASEDIR}
if [[ -f tmp/spire-server.pid ]]; then
	(kill $(cat tmp/spire-server.pid) || echo "Couldn't kill spire-server")
	rm tmp/spire-server.pid
fi

if [[ -f tmp/spire-agent.pid ]]; then
	(kill $(cat tmp/spire-agent.pid) || echo "Couldn't kill spire-agent")
	rm tmp/spire-agent.pid
fi

# Build the sigstore attestor from source and copy into bin dir
pushd src/sigstoreattestor
go build ./...
popd
cp src/sigstoreattestor/sigstoreattestor bin/sigstoreattestor

rm -r ${BASEDIR}/data || echo "Unable to remove old data"

# Start up spire-server with conf from conf dir
bin/spire-server run -config conf/server.conf > log/server.log &

echo $! > tmp/spire-server.pid
sleep 1

# Create a join token
join_token_output=$(bin/spire-server token generate -spiffeID spiffe://example.org/localNode -socketPath sock/server.sock)

regex='Token: ([a-z0-9-]*)'
if [[ $join_token_output =~ $regex ]]
then
        join_token="${BASH_REMATCH[1]}"
        echo $join_token
else
        echo "Unexpected output from \"spire-server token generate\": ${join_token_output}"
	exit 1
fi

echo "Starting spire agent with new join token ${join_token}"
# Start up spire-agent with conf from conf dir
bin/spire-agent run -joinToken ${join_token} -config conf/agent.conf > log/agent.log &

echo $! > tmp/spire-agent.pid

# Create the needed registration entry
bin/spire-server entry create \
	-spiffeID spiffe://example.org/testWorkload \
	-parentID spiffe://example.org/localNode \
	-selector sigstoreattestor:subject:${SIGSTORE_SUBJECT} \
	-socketPath sock/server.sock

