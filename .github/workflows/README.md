# Go-Flare binary release workflow

> Note: The SLSA3 isolated build job always runs on `ubuntu-latest`

## What is SLSA?

[SLSA](https://slsa.dev/) (Supply-chain Levels for Software Artifacts, pronounced "salsa") is an open security framework that defines a set of standards and controls to protect software integrity across the entire build pipeline. It's organized into four build levels (1–4), each adding stronger guarantees around build integrity and provenance. 

SLSA Build Level 3 (the level implemented here) requires that:
* Builds run in an isolated, ephemeral environment that cannot be influenced by the calling workflow.
* A signed, non-forgeable provenance attestation is generated, recording what was built, from which source, and by which builder.
* The build process itself is defined and verifiable, not dependent on mutable or untrusted inputs.

## Security Properties Introduced

**Reproducible Build Verification**

On every release, the binary is built twice, completely independently, and compared. If the hashes match, the build is confirmed reproducible. If they differ, the release fails and diffoscope runs automatically to show exactly what caused the difference.

The Go build flags that make this possible are:
* `trimpath` strips local filesystem paths from the binary
* `buildvcs=false` removes embedded VCS metadata that varies between builds

**Hash-Pinned GitHub Actions**

Where possible, GitHub Action and reusable workflow references in the pipeline are pinned to a full commit SHA rather than a mutable tag (e.g. `actions/checkout@f43a0e5ff2bd`... instead of @v3). This prevents a compromised or modified upstream action from silently altering the build, a common supply chain attack vector.

**SLSA3 Provenance Attestation**

Every release build produces a signed provenance file (`.build.slsa`) that cryptographically attests to what was built, from which exact source commit, and by which workflow. The attestation is signed via Sigstore and recorded in the Rekor public transparency log, making it independently verifiable and tamper-evident. This makes it impossible to publish a binary that wasn't produced by the declared build process.

**Keyless Release Signing**

The final release artifact is signed using Cosign. The resulting .sig and .pem files are published alongside the release artifact and can be used to confirm there was no tampering with release files.

## Pipeline Architecture
The build pipeline is split across three workflow files, orchestrated by `build-binary.yaml`.

```
build-binary.yaml
  ├── build-go-slsa3.yaml          (SLSA3 isolated build + provenance)
  ├── build-go-verification.yaml   (independent verification/reproducible build)
  └── release job                  (verify + sign + publish)
```

### SLSA BYOB Framework

SLSA3 build uses the [Build Your Own Builder (BYOB) framework](https://github.com/slsa-framework/slsa-github-generator/blob/main/BYOB.md). BYOB allows wrapping an existing build process into a SLSA3-compliant builder without implementing provenance generation from scratch.

The framework introduces three components:

* Tool Reusable Workflow (TRW) — The SLSA3 builder that callers invoke. It initializes the BYOB framework and delegates execution to the SLSA runner.
* Tool Callback Action (TCA) — The action that BYOB invokes in an isolated, ephemeral VM — separate from the calling workflow. It runs the actual build, hashes the output artifacts, and uploads them securely.
* BYOB Delegator — Receives the SLSA token from the TRW, runs the TCA in isolation, and generates the signed provenance attestation.

### Job Flow

`slsa-setup` initializes the BYOB framework via `setup-generic@v2.1.0`, passing the TCA path and workflow inputs. It receives back a short-lived `slsa-token` that encodes the build parameters.

`slsa-run` passes the token to the BYOB delegator, which takes over. It runs the TCA in an isolated VM, collects artifact digests from the SLSA layout file produced by `generate-layout.sh`, and generates a signed `.build.slsa` provenance file.

`build-go-verification` runs in parallel. A plain second build using the same go-build composite action, producing an independent binary used only for the reproducibility check.

`release` pulls everything together:

* Compares the SLSA build binary against the verification binary
* Verifies the SLSA provenance with slsa-verifier, asserting source URI and builder identity
* Packages, signs, and verifies the release zip with Cosign
* Publishes the release with the zip, .build.slsa, and Cosign .sig/.pem files

## Verification Guide

Each release publishes the following files:

| File | Description |
|------|-------------|
| `go-flare-<tag>-linux-amd64.zip` | Release binary |
| `go-flare-<tag>-linux-amd64.zip.build.slsa` | SLSA provenance attestation |
| `go-flare-<tag>-linux-amd64.zip.sig` | Cosign signature |
| `go-flare-<tag>-linux-amd64.zip.pem` | Cosign certificate |

### Verifying SLSA Provenance

The SLSA attestation covers the raw `avalanchego` binary, not the zip archive. Extract it first, then verify:

```
unzip go-flare-<tag>-linux-amd64.zip avalanchego
slsa-verifier verify-artifact \
  avalanchego \
  --provenance-path go-flare-<tag>-linux-amd64.zip.build.slsa \
  --source-uri github.com/flare-foundation/go-flare \
  --builder-id https://github.com/flare-foundation/go-flare/.github/workflows/build-go-slsa3.yaml@refs/tags/<tag>
```

A successful verification confirms that the binary was built from the declared source repository, by the declared workflow, in an isolated SLSA3-compliant environment.

### Verifying the Cosign Signature

Install [cosign](https://github.com/sigstore/cosign), then run:

```
cosign verify-blob \
  --certificate go-flare-<tag>-linux-amd64.zip.pem \
  --certificate-identity https://github.com/flare-foundation/go-flare/.github/workflows/build-binary.yaml@refs/tags/<tag> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature go-flare-<tag>-linux-amd64.zip.sig \
  go-flare-<tag>-linux-amd64.zip
```

This confirms the release zip was signed by the official release workflow and has not been tampered with since signing.