.PHONY: all build-01 build-02 build-03 build-04 test-01 test-02 test-03 test-04 test-all

all: build-01 build-02 build-03 build-04

build-01:
	cd 01-basic-auction && near-go build

build-02:
	cd 02-nft-auction && near-go build

build-03:
	cd 03-ft-auction && near-go build

build-04:
	cd 04-factory && near-go build

test-01:
	cd 01-basic-auction && near-go test package

test-02:
	cd 02-nft-auction && near-go test package

test-03:
	cd 03-ft-auction && near-go test package

test-04:
	cd 04-factory && near-go test package

test-all: test-01 test-02 test-03 test-04

integration-test-01:
	cd 01-basic-auction/integration_tests && cargo run

integration-test-02:
	cd 02-nft-auction/integration_tests && cargo run

integration-test-03:
	cd 03-ft-auction/integration_tests && cargo run

integration-test-04:
	cd 04-factory/integration_tests && cargo run
