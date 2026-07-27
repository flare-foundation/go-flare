<p align="left">
  <a href="https://flare.network/" target="blank"><img src="https://content.flare.network/Flare-2.svg" width="300" alt="Flare Logo" /></a>
</p>

# go-flare

go-flare is a modified version of [avalanchego@v1.14.2](https://github.com/ava-labs/avalanchego/releases/tag/v1.14.2) incorporating specific features for Flare and Songbird networks. These features include prioritized contract handling and the invocation of the daemon contract.

See [release notes](./RELEASES-flare.md) for more info.

## System Requirements

- go version 1.25.11
- gcc, g++ and jq
- CPU: Equivalent of 8 AWS vCPU
- RAM: 16 GiB
- Storage: 1TB Flare / 3.5TB Songbird
- OS: Ubuntu 22.04/24.04

## Compilation

After cloning this repository, run:

```sh
cd go-flare/avalanchego && ./scripts/build.sh
```

## Deploy a Validation Node

These servers fulfill a critical role in securing the network:

- They check that all received transactions are valid.
- They run a consensus algorithm so that all validators in the network agree on the transactions to add to the blockchain.
- Finally, they add the agreed-upon transactions to their copy of the ledger.

This guide explains how to deploy your own validator node so you can participate in the consensus and collect the rewards that the network provides to those who help secure it: <https://docs.flare.network/infra/validation/deploying/>

## Deploy an Observation Node

Observation nodes enable anyone to observe the network and submit transactions. Unlike validator nodes, which provide state consensus and add blocks, observation nodes remain outside the network and have no effect on consensus or blocks.

This guide explains how to deploy your own observation node: <https://docs.flare.network/infra/observation/deploying/>

## Tests

See `tests/README.md` for testing details

## Container image

Public container images are hosted on [Docker Hub](https://hub.docker.com/r/flarefoundation/go-flare) and [GitHub Packages](https://github.com/orgs/flare-foundation/packages?repo_name=go-flare):

```
docker.io/flarefoundation/go-flare
ghcr.io/flare-foundation/go-flare
```

### Verify the cosign signature

Images are signed using [Cosign](https://github.com/sigstore/cosign) with the GitHub OIDC provider. To verify the image, run this command:

```bash
cosign verify \
  --certificate-identity-regexp="^https://github\.com/flare-foundation/go-flare/\.github/workflows/build-container\.yml@" \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  ghcr.io/flare-foundation/go-flare:<TAG>

cosign verify \
  --certificate-identity-regexp="^https://github\.com/flare-foundation/go-flare/\.github/workflows/build-container\.yml@" \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  docker.io/flarefoundation/go-flare:<TAG>
```

### Inspect provenance + SBOM

Both are attached to the image and inspectable with `docker buildx imagetools`:

```bash
# provenance — build origin, source repo, commit, workflow
docker buildx imagetools inspect ghcr.io/flare-foundation/go-flare:<TAG> \
  --format '{{ json (index .Provenance "linux/amd64") }}' | jq

# SBOM — list of packages
docker buildx imagetools inspect ghcr.io/flare-foundation/go-flare:<TAG> \
  --format '{{ json (index .SBOM "linux/amd64").SPDX.packages }}' | jq
```

### Verify GitHub artifact attestations

Browse all attestations for this repository: <https://github.com/flare-foundation/go-flare/attestations>

Requires [GitHub CLI](https://cli.github.com/) 2.49 or later:

```bash
gh attestation verify oci://ghcr.io/flare-foundation/go-flare:<TAG> --owner flare-foundation
```

### Container builds in CI

Builds run on:

- Push to `main` branch → pushes image tagged `dev` + `dev-dless`
- Push of a pre-release tag (`v*-rc.*`, `v*-alpha.*`, etc.) → pushes tagged image + `-dless` variant
- Push of a stable tag (e.g. `v1.13.0`) → pushes tagged image + `-dless` variant AND updates `latest` + `latest-dless`

The `latest` tag **only advances on stable releases**. Pre-releases do not update it.
