#!/usr/bin/env bash
# Node image for Titan Docker topology.
#
# Default: locally built from this repo (pnpm run node:docker:build).
# Deployed / CI: override with ghcr.io/pakeku/go-titan:<tag>
#
#   TITAN_NODE_IMAGE=ghcr.io/pakeku/go-titan:dev-explorer pnpm run node:docker:start:all

export TITAN_NODE_IMAGE="${TITAN_NODE_IMAGE:-go-titan-local}"

ensure_titan_node_image() {
	local root="${1:?repo root required}"

	if docker image inspect "$TITAN_NODE_IMAGE" >/dev/null 2>&1; then
		return 0
	fi

	if [[ "$TITAN_NODE_IMAGE" != "go-titan-local" ]]; then
		echo "Image $TITAN_NODE_IMAGE not found locally. Pull it or set TITAN_NODE_IMAGE=go-titan-local." >&2
		exit 1
	fi

	echo "Building $TITAN_NODE_IMAGE from $root/Dockerfile ..."
	docker build -t "$TITAN_NODE_IMAGE" "$root"
}