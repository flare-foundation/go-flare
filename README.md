# go-flare

go-flare is a modified version of [avalanchego@v1.7.18](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.18) + [coreth@v0.8.16](https://github.com/ava-labs/coreth/releases/tag/v0.8.16) that incorporates the Flare Time Series Oracle (FTSO) and State Connector. 

## System Requirements
- go version 1.18.5
- gcc, g++ and jq
- CPU: Equivalent of 8 AWS vCPU
- RAM: 16 GiB
- Storage: 1TB
- OS: Ubuntu 18.04/20.04 or macOS >= 10.15 (Catalina)

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

This guide explains how to deploy your own validator node so you can participate in the consensus and collect the rewards that the network provides to those who help secure it: https://docs.flare.network/infra/validation/deploying/

## Deploy an Observation Node

Observation nodes enable anyone to observe the network and submit transactions. Unlike validator nodes, which provide state consensus and add blocks, observation nodes remain outside the network and have no effect on consensus or blocks.

This guide explains how to deploy your own observation node: https://docs.flare.network/infra/observation/deploying/

## Tests

See `tests/README.md` for testing details

## Container image

Public container images are hosted on [Docker HUB](https://hub.docker.com/r/flarefoundation/go-flare) and [Github Packages](https://github.com/orgs/flare-foundation/packages?repo_name=go-flare);
```
docker.io/flarefoundation/go-flare
hgcr.io/flare-foundation/go-flare
```

### Container builds in CI

CI builds on each:
- push on `main` branch, pushes image tagged as "dev"
- creation of a tag, pushes images tagged as the tag itself

Builds: \
two images, `go-flare:<TAG>` one with `leveldb`
