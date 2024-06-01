# AVIDA: Cheqd Node Interchaintests

This repo is created in order to test interchain tx for [`AVIDA`] contracts with the Cheqd network.
The submodule `contracts` contains the [`AVIDA`] code and `contracts_wasm` is where the built and optimised contracts are stored.

## Building contracts

```sh
make build-contracts
```

## Running tests

```sh
go test -v ./...

# Run only the e2e IBC test in avida_ibc_test.go
go test --short
```


[`AVIDA`]: https://github.com/nymlab/avida
