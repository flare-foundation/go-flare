## Unit Tests

Unit tests can be run using these commands from the `avalanchego/` folder:

```sh
go test $(go list ./... | grep -v /tests/) #avalanchego unit tests
```

```sh
cd ../coreth && go test ./... #coreth unit tests
```

## Run a Local Network with Flare Genesis

From the `go-flare/avalanchego` folder, run:
```sh
./scripts/localflare.sh
```

## Test P-chain Transactions

See the [test-scripts/README.md](../test-scripts/README.md) for instructions.

