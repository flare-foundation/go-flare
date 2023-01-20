# Flare & Coston2

Docker image for the Flare & Coston2 node implementation found on [github](https://github.com/flare-foundation/go-flare).

## Quickstart

```sh
docker run -d \
	-p 9650-9651:9650-9651 \
	flarefoundation/flare:latest
```

<b>Currently the default network is `costwo` but you can change that by providing a `NETWORK_ID` environment variable (i.e. `NETWORK_ID=flare`).</b>

## Mounting storage

The three points of interest for mounting are:

| Name | Default location |
|---|:--|
| Database | `/app/db` |
| Logging | `/app/logs` |
| Configuration | `/app/conf` |

<b>All of these may be changed using the environment variables.</b>

```sh
docker run -d \
	-v /tmp/db:/app/db \
	-v /tmp/conf:/app/conf \
	-p 9650:9650 \
	-e AUTOCONFIGURE_BOOTSTRAP=1 \
	flarefoundation/go-songbird:latest
```

## Container Configuration

These are the environment variables you can edit and their default values:

| Name | Default | Description |
|:--|:--|:--|
| `HTTP_HOST` | `0.0.0.0` | The bind address of the service |
| `HTTP_PORT` | `9650` | The port on which the API is served |
| `STAKING_PORT` | `9651` | The staking port for bootstrapping nodes |
| `PUBLIC_IP` | _(empty)_ | The public IP of the service |
| `DB_DIR` | `/app/db` | The database directory location |
| `DB_TYPE` | `leveldb` | The database type to be used |
| `BOOTSTRAP_IPS` | _(empty)_ | A list of bootstrap server ips; ref [--bootstrap-ips-string](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#--bootstrap-ips-string) |
| `BOOTSTRAP_IDS` | _(empty)_ | A list of bootstrap server ids; ref [--bootstrap-ids-string](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#--bootstrap-ids-string) |
| `CHAIN_CONFIG_DIR` | `/app/conf` | Configuration directory |
| `LOG_DIR` | `/app/logs` | Logging directory |
| `LOG_LEVEL` | `info` | Logging level |
| `NETWORK_ID` | `costwo` | The network id. The common ids are `flare \| costwo` |
| `AUTOCONFIGURE_PUBLIC_IP` | `0` | Set to `1` to autoconfigure `PUBLIC_IP`, skipped if PUBLIC_IP is set |
| `AUTOCONFIGURE_BOOTSTRAP` | `0` | Set to `1` to autoconfigure `BOOTSTRAP_IPS` and `BOOTSTRAP_IDS` |
| `AUTOCONFIGURE_BOOTSTRAP_ENDPOINT` | `https://coston2.flare.network/ext/info` <b>or</b> `https://flare.flare.network/ext/info` | Endpoint used for [bootstrapping](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#bootstrapping) when `AUTOCONFIGURE_BOOTSTRAP` is enabled. For the `flare` network id, this has to be set to `https://flare.flare.network/ext/info` |
| `EXTRA_ARGUMENTS` | | Extra arguments passed to flare binary |


## Node Configuration

The flare node can be configured by specifying your own configuration for the different chains but mainly the C (aka. Contract) chain. The specified configuration determines which capabilities the node has and it affects how the node has to be set up. We mainly distinguish between the three standard configurations described below.

### External API configuration

The external API configuration is set to only respond to API calls so it offloads the other internal nodes. What makes it external is the relatively small set of functions that it exposes which are meant for public use. The node with this configuration exposes the HTTP port (default: 9650) and does not need any publicly open ports to work. The images tagged with the suffix `api` have this configuration preloaded by default.

```sh
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": false,
  "coreth-admin-api-dir": "",
  "eth-apis": [
    "public-eth",
    "public-eth-filter",
    "net",
    "web3",
    "internal-public-eth",
    "internal-public-blockchain",
    "internal-public-transaction-pool"
  ],
}
```

### Internal API configuration

Similarly to the external API configuration, this one also responds to API calls but has additional calls exposed that help with longer running tasks, debugging, etc. It is therefore <b>NOT</b> meant for public use and it should <b>NOT</b> be publicly accessible. The node with this configuration exposes the HTTP port (default: 9650) and does not need any publicly open ports to work.

```sh
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": false,
  "coreth-admin-api-dir": "",
  "eth-apis": [
    "public-eth",
    "public-eth-filter",
    "private-admin",
    "public-debug",
    "private-debug",
    "net",
    "debug-tracer",
    "web3",
    "internal-public-eth",
    "internal-public-blockchain",
    "internal-public-transaction-pool",
    "internal-public-tx-pool",
    "internal-public-debug",
    "internal-private-debug",
    "internal-public-account",
    "internal-private-personal"
  ],
}
```

### Bootstrap configuration

The bootstrap configuration is meant for nodes that will accept and help provision new nodes that want to connect to the network. They need to be publicly accessible and need the staking port (default: 9651) port-forwarded while the http port may remain inaccessible from the public but is needed to initialise the bootstrapping process of a new node.

```sh
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": false,
  "coreth-admin-api-dir": "",
  "eth-apis": [
    "web3"
  ],
}
```

### Additional information

Here's a list of helpful links for additional information about configuration:

* [Chain types and configuration](https://docs.avax.network/nodes/maintain/chain-config-flags)
* [Staking](https://docs.avax.network/nodes/validate/staking)
* [Bootstrapping](https://docs.avax.network/nodes/maintain/avalanchego-config-flags#bootstrapping)
* [API calls](https://docs.avax.network/apis/avalanchego/apis)
