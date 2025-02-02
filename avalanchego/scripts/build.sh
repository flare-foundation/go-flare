#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

print_usage() {
  printf "Usage: build [OPTIONS]

  Build avalanchego

  Options:

    -r  Build with race detector
"
}

race=''
while getopts 'r' flag; do
  case "${flag}" in
    r) race='-r' ;;
    *) print_usage
      exit 1 ;;
  esac
done

# Avalanchego root folder
AVALANCHE_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )
CORETH_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd ../../coreth && pwd )
# Load the versions
source "$AVALANCHE_PATH"/scripts/versions.sh
# Load the constants
source "$AVALANCHE_PATH"/scripts/constants.sh

# Download dependencies
echo "Downloading dependencies..."
go mod download -modcacherw

build_args="$race"

echo "Syncing with sources at GOPATH: $GOPATH"

rsync -ar --delete $AVALANCHE_PATH/* $GOPATH/pkg/mod/github.com/ava-labs/avalanchego@$avalanche_version
rsync -ar --delete $CORETH_PATH/* $GOPATH/pkg/mod/github.com/ava-labs/coreth@$coreth_version

# Build avalanchego
"$AVALANCHE_PATH"/scripts/build_avalanche.sh $build_args

# Build coreth
"$AVALANCHE_PATH"/scripts/build_coreth.sh

# Exit build successfully if the AvalancheGo binary is created successfully
if [[ -f "$avalanchego_path" ]]; then
        echo "Build Successful"
        exit 0
else
        echo "Build failure" >&2
        exit 1
fi
