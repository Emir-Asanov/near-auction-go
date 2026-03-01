# near-auction-go

NEAR Protocol auction smart contracts written in Go using [near-sdk-go](https://github.com/vlmoon99/near-sdk-go) v0.1.1.

## Contracts

| Directory | Description |
|-----------|-------------|
| `01-basic-auction` | Basic NEAR auction (bids in NEAR tokens) |
| `02-nft-auction` | NFT auction with `nft_transfer` on claim |
| `03-ft-auction` | Fungible token auction via `ft_on_transfer` |
| `04-factory` | Factory contract that deploys auction subaccounts |

Each contract has:
- `main.go` — contract source
- `main_test.go` — unit tests (`near-go test package`)
- `integration_tests/` — Rust sandbox integration tests (`cargo run`)
- `main.wasm` — pre-built WASM binary

The `core/` module contains shared types used across contracts.

## Prerequisites

### 1. near-go CLI (from near-cli-go)

```bash
git clone https://github.com/vlmoon99/near-cli-go
cd near-cli-go
go build -o near-go ./cmd/near-go
# Move binary to a directory in your PATH, e.g.:
mv near-go ~/bin/near-go
```

Verify: `near-go version`

### 2. Rust + Cargo

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

Verify: `cargo --version`

The Rust integration tests use [near-workspaces](https://github.com/near/near-workspaces-rs) which automatically downloads the NEAR sandbox binary on first run.

## Running All Tests

```bash
./build_test.sh
```

This script sequentially for each contract:
1. Builds the WASM (`near-go build`)
2. Runs unit tests (`near-go test package`)
3. Runs integration tests (`cargo run` inside `integration_tests/`)

For `04-factory` it also copies `03-ft-auction/main.wasm` → `04-factory/auction.wasm` before building (the factory embeds the auction WASM at compile time).

## Running Individually

```bash
# Build
cd 01-basic-auction && near-go build

# Unit tests
cd 01-basic-auction && near-go test package

# Integration tests
cd 01-basic-auction/integration_tests && cargo run
```

## Project Structure

```
near-auction-go/
├── core/                    # Shared Go module (Bid type)
│   ├── go.mod
│   └── types.go
├── 01-basic-auction/
│   ├── go.mod               # requires near-sdk-go v0.1.1, core
│   ├── main.go
│   ├── main_test.go
│   ├── main.wasm
│   └── integration_tests/
│       ├── Cargo.toml
│       └── src/main.rs
├── 02-nft-auction/          # same structure
├── 03-ft-auction/           # same structure
├── 04-factory/
│   ├── go.mod               # requires near-sdk-go v0.1.1
│   ├── main.go
│   ├── main_test.go
│   ├── auction.wasm         # embedded auction contract (from 03-ft-auction)
│   ├── main.wasm
│   └── integration_tests/
│       ├── Cargo.toml
│       └── src/main.rs
├── build_test.sh            # full build + test script
└── Makefile
```
