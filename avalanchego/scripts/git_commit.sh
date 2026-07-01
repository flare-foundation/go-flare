#!/usr/bin/env bash

# Ignore warnings about variables appearing unused since this file is not the consumer of the variables it defines.
# shellcheck disable=SC2034

set -euo pipefail

AVALANCHE_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd ) # Directory above this script

# WARNING: this will use the most recent commit even if there are un-committed changes present
# Flare: avalanchego is nested inside the go-flare repo, so the .git lives at the repo root
# (there is no ${AVALANCHE_PATH}/.git). Use `git -C` discovery instead of upstream's hardcoded
# --git-dir so the commit resolves in both local and Docker (/app) builds.
git_commit="${AVALANCHEGO_COMMIT:-$(git -C "${AVALANCHE_PATH}" rev-parse HEAD)}"
commit_hash="${git_commit::8}"
