.PHONY: all build-01 build-02 build-03 build-04 test-01 test-02 test-03 test-04 test-all

all: build-01 build-02 build-03 build-04

build-01:
	cd 01-basic-auction && near-sdk-go build

build-02:
	cd 02-nft-auction && near-sdk-go build

build-03:
	cd 03-ft-auction && near-sdk-go build

build-04:
	cd 04-factory && near-sdk-go build

test-01:
	cd 01-basic-auction && go test ./...

test-02:
	cd 02-nft-auction && go test ./...

test-03:
	cd 03-ft-auction && go test ./...

test-04:
	cd 04-factory && go test ./...

test-all: test-01 test-02 test-03 test-04
