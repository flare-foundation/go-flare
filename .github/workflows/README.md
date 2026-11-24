# Go-Flare binary release workflow

## What is SLSA?

[SLSA](https://slsa.dev/) (Supply-chain Levels for Software Artifacts, pronounced "salsa") is an open security framework that defines a set of standards and controls to protect software integrity across the entire build pipeline. It's organized into build levels, each adding stronger guarantees around build integrity and provenance.

SLSA Build Level 3 (the level implemented here) requires that:
* Builds run in an isolated, ephemeral environment that cannot be influenced by the calling workflow.
* A signed, non-forgeable provenance attestation is generated, recording what was built, from which source, and by which builder.
* The build process itself is defined and verifiable, not dependent on mutable or untrusted inputs.

Provenance is generated with [GitHub artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations) (`actions/attest`). Attestations created on GitHub-hosted runners give SLSA Build Level 2; creating them inside a reusable workflow (`build-go.yaml`) makes that workflow the signer identity, which gives Level 3.

## Security Properties Introduced

**Reproducible Build Verification**

On every release, the amd64 binary is built twice, completely independently, and compared. If the hashes match, the build is confirmed reproducible. If they differ, the release fails and diffoscope runs automatically to show exactly what caused the difference.

The Go build flags that make this possible are:
* `trimpath` strips local filesystem paths from the binary
* `buildvcs=false` removes embedded VCS metadata that varies between builds

**Hash-Pinned GitHub Actions**

Where possible, GitHub Action and reusable workflow references in the pipeline are pinned to a full commit SHA rather than a mutable tag (e.g. `actions/checkout@f43a0e5ff2bd`... instead of @v3). This prevents a compromised or modified upstream action from silently altering the build, a common supply chain attack vector.

**SLSA3 Provenance Attestation**

Every build of the amd64 and arm64 binaries produces a signed SLSA v1 provenance attestation that records what was built, from which exact source commit, and by which workflow. Attestations are signed via Sigstore (the public-good instance for public repositories), stored in the GitHub attestations API and published with each release as a Sigstore bundle (`.sigstore.json`), so they can be verified online or offline.

The container images are attested the same way, and the attestations are also pushed to the registry next to the image.

**Keyless Release Signing**

The release zips are signed using Cosign. The resulting .sig and .pem files are published alongside the release artifacts and can be used to confirm there was no tampering with release files.

## Pipeline Architecture
The build pipeline is split across three workflow files, orchestrated by `build-binary.yaml`.

```
build-binary.yaml
  ├── build-go.yaml                (Go build + provenance attestation: amd64, arm64, and the verification build)
  ├── build-container.yml          (standard and -dless images, built in parallel from the binaries above)
  └── release job                  (verify + sign + publish)
```

### Job Flow

`build` and `build-arm64` (`build-go.yaml`) build the binaries natively on `ubuntu-26.04` and `ubuntu-26.04-arm` runners. The build job only has a read-only token; a separate `attest` job in the same reusable workflow downloads the binary and attests it with `actions/attest`, so build steps never have access to the OIDC token used for signing.

`build-verification` (`build-go.yaml`, tags only) runs in parallel. A plain second amd64 build using the same go-build composite action, producing an independent binary used only for the reproducibility check. It is not attested.

`images` (`build-container.yml`) does not compile anything: it copies the attested binaries into the images via `--build-context binaries=...`. Before that it verifies both binaries with `gh attestation verify` (signer `build-go.yaml`, same git ref and commit, GitHub-hosted runners only). It then smoke tests both architectures, pushes the images, attests them (`push-to-registry`), verifies the image attestation and signs the images with Cosign. The images therefore contain the same binaries as the release zips. Set the repository variable `USE_DOCKER_HUB=false` (e.g. on forks) to push to ghcr.io only. A local `docker build` without `--build-context` still builds from source.

`release` (tags only) pulls everything together:

* Compares the amd64 binary against the verification binary
* Verifies the provenance of both binaries with `gh attestation verify`
* Packages both architectures with the attestation bundle created by each build, and signs and verifies the zips with Cosign
* Publishes the release with the zips, `.sigstore.json` bundles, and Cosign .sig/.pem files

## Verification Guide

Each release publishes the following files, for both `amd64` and `arm64`:

| File | Description |
|------|-------------|
| `go-flare-<tag>-linux-<arch>.zip` | Release binary |
| `go-flare-<tag>-linux-<arch>.zip.sigstore.json` | SLSA provenance attestation (Sigstore bundle) for the binary in the zip |
| `go-flare-<tag>-linux-<arch>.zip.sig` | Cosign signature |
| `go-flare-<tag>-linux-<arch>.zip.pem` | Cosign certificate |

### Verifying SLSA Provenance

The attestation covers the raw `avalanchego` binary, not the zip archive. Extract it first, then verify with the [GitHub CLI](https://cli.github.com/) (requires `gh auth login`):

```
unzip go-flare-<tag>-linux-<arch>.zip avalanchego
gh attestation verify avalanchego \
  --repo flare-foundation/go-flare \
  --signer-workflow flare-foundation/go-flare/.github/workflows/build-go.yaml \
  --source-ref refs/tags/<tag>
```

To also pin the source commit (in case the tag is moved later), add `--source-digest <commit>`, taking the commit SHA from a source you trust, or from the repository with `git ls-remote https://github.com/flare-foundation/go-flare 'refs/tags/<tag>^{}' | cut -f1`. For an annotated tag, leaving out `^{}` returns the tag object SHA, which does not match.

To verify against the published bundle instead of the GitHub API, add `--bundle go-flare-<tag>-linux-<arch>.zip.sigstore.json`. It holds only the attestation of the release build. `gh attestation download` can return several attestations for the same binary, because rebuilding the same commit reproduces it byte for byte; cosign accepts only a single bundle, so use the published file with cosign.

Without the GitHub CLI, the bundle can be verified with [cosign](https://github.com/sigstore/cosign) (v2.4 or newer):

```
cosign verify-blob-attestation \
  --new-bundle-format \
  --bundle go-flare-<tag>-linux-<arch>.zip.sigstore.json \
  --type slsaprovenance1 \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github\.com/flare-foundation/go-flare/\.github/workflows/build-go\.yaml@refs/tags/<tag>$' \
  avalanchego
```

A successful verification confirms that the binary was built from the declared source repository and git ref, by the declared reusable workflow, on GitHub-hosted infrastructure.

Releases published before the switch to GitHub artifact attestations ship a `.build.slsa` file instead and are verified with [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) against `.github/workflows/build-go-slsa3.yaml`.

### Verifying the Cosign Signature

Install [cosign](https://github.com/sigstore/cosign), then run:

```
cosign verify-blob \
  --certificate go-flare-<tag>-linux-<arch>.zip.pem \
  --certificate-identity https://github.com/flare-foundation/go-flare/.github/workflows/build-binary.yaml@refs/tags/<tag> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature go-flare-<tag>-linux-<arch>.zip.sig \
  go-flare-<tag>-linux-<arch>.zip
```

This confirms the release zip was signed by the official release workflow and has not been tampered with since signing.

### Verifying Container Images

With the GitHub CLI (requires `gh auth login`), or without a login using the `cosign verify-attestation --new-bundle-format` command from the [README](../../README.md#container-image):

```
gh attestation verify oci://ghcr.io/flare-foundation/go-flare:<tag> \
  --repo flare-foundation/go-flare \
  --signer-workflow flare-foundation/go-flare/.github/workflows/build-container.yml
```

The same works for the `-dless` variant and for `docker.io/flarefoundation/go-flare`. Images are also signed with Cosign, see the [README](../../README.md#container-image).
