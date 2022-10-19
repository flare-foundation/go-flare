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

### Container builds in CI

CI builds on each:
- push on `main` branch, pushes image tagged as "latest"
- creation of a tag, pushes images tagged as the tag itself

Builds: \
two images, `go-flare:<TAG>` one with `leveldb` and `go-falre:<TAG>-rocksdb` with RocksDB builtin

### Build arguments

| Argument name | Default value | description |
|---|---|---|
| `DB_TYPE` | `leveldb` | if `rocksdb` the image will be built with rocksdb support; ref [docs.avax.network](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#database) |


### Runtime environment variables

| Varible name | Default value | description |
|---|---|---|
| `HTTP_HOST` | `0.0.0.0` | Should always be `0.0.0.0` as it's a container |
| `HTTP_PORT` | `9650` | |
| `STAKING_PORT` | `9651` | |
| `PUBLIC_IP` | ` ` | can be autoconfigured by having `AUTOCONFIGURE_PUBLIC_IP` enabled |
| `DB_TYPE` | `leveldb` | One of `leveldb \| rocksdb \| memdb \| memdb`. Rocksdb can only be used with images whose tags end with `-rocksdb`. |
| `DB_DIR` | `/app/db` | |
| `BOOTSTRAP_IPS` | ` ` | [--bootstrap-ids-string](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#--bootstrap-ids-string), can be autoconfigured by enabling `AUTOCONFIGURE_BOOTSTRAP` |
| `BOOTSTRAP_IDS` | ` ` | [--bootstrap-ips-string](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#--bootstrap-ips-string), can be autoconfigured by enabling `AUTOCONFIGURE_BOOTSTRAP` |
| `CHAIN_CONFIG_DIR` | `/app/conf` | |
| `LOG_DIR` | `/app/logs` | |
| `LOG_LEVEL` | `info` | |
| `NETWORK_ID` | `coston2` | One of `flare \| costwo`. Define the [target network](https://docs.flare.network/dev/reference/network-configs/) to work with |
| `AUTOCONFIGURE_PUBLIC_IP` | `1` | Autoconfigure PUBLIC_IP, skipped if PUBLIC_IP is set |
| `AUTOCONFIGURE_BOOTSTRAP` | `1` | Enables auto-fetch of [--bootstrap-ids-string](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#--bootstrap-ids-string) and [--bootstrap-ips-string](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#--bootstrap-ips-string) values from `AUTOCONFIGURE_BOOTSTRAP_ENDPOINT` |
| `AUTOCONFIGURE_BOOTSTRAP_ENDPOINT` | `https://coston2.flare.network/ext/info` | Endpoint used for [bootstrapping](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#bootstrapping) info fetch |
| `EXTRA_ARGUMENTS` | ` ` | Extra arguments passed to flare binary when running it from `entrypoint.sh` |
