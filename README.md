<p align="left">
  <a href="https://flare.network/" target="blank"><img src="https://content.flare.network/Flare-2.svg" width="300" alt="Flare Logo" /></a>
</p>

# go-flare

go-flare is a modified version of [avalanchego@v1.15.1](https://github.com/ava-labs/avalanchego/releases/tag/v1.15.1) incorporating specific features for Flare and Songbird networks. These features include prioritized contract handling and the invocation of the daemon contract.

See [release notes](./RELEASES-flare.md) for more info.

## System Requirements

- go version 1.26.10
- gcc, g++ and jq
- CPU: Equivalent of 8 AWS vCPU
- RAM: 16 GiB
- Storage: 1TB Flare / 3.5TB Songbird
- OS: Ubuntu 24.04

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

Each tag is published in a standard variant and a distroless variant (`<TAG>-dless`), both for `linux/amd64` and `linux/arm64`.

### Verify the cosign signature

Images are signed using [Cosign](https://github.com/sigstore/cosign) with the GitHub OIDC provider. Images are signed by the reusable `build-image.yml` workflow; images published before it was introduced were signed by `build-container.yml`, so the regexp below accepts both. To verify the image, run this command:

```bash
cosign verify \
  --certificate-identity-regexp="^https://github\.com/flare-foundation/go-flare/\.github/workflows/(build-image|build-container)\.yml@" \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  ghcr.io/flare-foundation/go-flare:<TAG>

cosign verify \
  --certificate-identity-regexp="^https://github\.com/flare-foundation/go-flare/\.github/workflows/(build-image|build-container)\.yml@" \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  docker.io/flarefoundation/go-flare:<TAG>
```

### Inspect provenance and SBOM

Images are built with BuildKit [SLSA provenance](https://slsa.dev/) (`mode=max`) and an [SPDX](https://spdx.dev/) SBOM attached to the image index. Both can be inspected with `docker buildx imagetools`:

```bash
# provenance: build origin, source repository, commit, workflow
docker buildx imagetools inspect ghcr.io/flare-foundation/go-flare:<TAG> \
  --format '{{ json (index .Provenance "linux/amd64") }}' | jq

# SBOM: list of packages
docker buildx imagetools inspect ghcr.io/flare-foundation/go-flare:<TAG> \
  --format '{{ json (index .SBOM "linux/amd64").SPDX.packages }}' | jq
```

### Verify GitHub artifact attestations

Provenance and SBOM are also published as signed [GitHub artifact attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations) and pushed to `ghcr.io` alongside the image. Browse all attestations for this repository: <https://github.com/flare-foundation/go-flare/attestations>

Verification requires [GitHub CLI](https://cli.github.com/) 2.49 or later:

```bash
# --signer-workflow only accepts attestations signed by the reusable build workflow
# (.github/workflows/build-image.yml), which is what makes the provenance SLSA Build L3.

# build provenance
gh attestation verify oci://ghcr.io/flare-foundation/go-flare:<TAG> \
  --owner flare-foundation \
  --signer-workflow flare-foundation/go-flare/.github/workflows/build-image.yml

# SBOM
gh attestation verify oci://ghcr.io/flare-foundation/go-flare:<TAG> \
  --owner flare-foundation \
  --signer-workflow flare-foundation/go-flare/.github/workflows/build-image.yml \
  --predicate-type https://spdx.dev/Document/v2.3
```

Docker Hub images share the same digest, so the same commands work with `oci://docker.io/flarefoundation/go-flare:<TAG>`.

### Container builds in CI

CI builds on each:

- push on `main` (or `feature/**`) branch, pushes images tagged as `dev` and `dev-dless`
- creation of a `v*` tag, pushes images tagged as the tag itself and `<TAG>-dless`
