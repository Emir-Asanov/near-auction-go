package main

import (
	"encoding/base64"
	"testing"

	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/system"
	"github.com/vlmoon99/near-sdk-go/types"
)

func init() {
	env.SetEnv(system.NewMockSystem())
}

func mockSys(t *testing.T) *system.MockSystem {
	t.Helper()
	m, ok := env.NearBlockchainImports.(*system.MockSystem)
	if !ok {
		t.Fatal("environment is not MockSystem")
	}
	return m
}

func setupTest(t *testing.T) *FactoryContract {
	t.Helper()
	m := mockSys(t)
	m.Storage = make(map[string][]byte)
	m.CurrentAccountIdSys = "factory.testnet"
	m.PredecessorAccountIdSys = "factory.testnet"
	m.AttachedDepositSys = types.Uint128{Hi: 0, Lo: 0}
	m.Promises = nil

	c := &FactoryContract{}
	c.Init()
	return c
}

func TestFactory_Init(t *testing.T) {
	c := setupTest(t)

	if len(c.Code) == 0 {
		t.Error("expected embedded WASM code after init")
	}
	if c.GetCodeSize() != len(embeddedAuctionWasm) {
		t.Errorf("code size: want %d, got %d", len(embeddedAuctionWasm), c.GetCodeSize())
	}
}

func TestFactory_UpdateAuctionContract_Success(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "factory.testnet"
	m.CurrentAccountIdSys = "factory.testnet"

	newWasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0xFF}
	encoded := base64.StdEncoding.EncodeToString(newWasm)

	if err := c.UpdateAuctionContract(UpdateCodeInput{Code: encoded}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if c.GetCodeSize() != len(newWasm) {
		t.Errorf("code size: want %d, got %d", len(newWasm), c.GetCodeSize())
	}
}

func TestFactory_UpdateAuctionContract_Unauthorized(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "attacker.testnet"
	m.CurrentAccountIdSys = "factory.testnet"

	encoded := base64.StdEncoding.EncodeToString([]byte("fake"))
	err := c.UpdateAuctionContract(UpdateCodeInput{Code: encoded})
	if err == nil {
		t.Fatal("expected error for unauthorized update, got nil")
	}
	if err.Error() != "only the contract itself can call this method" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFactory_UpdateAuctionContract_InvalidBase64(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "factory.testnet"
	m.CurrentAccountIdSys = "factory.testnet"

	err := c.UpdateAuctionContract(UpdateCodeInput{Code: "not-valid-base64!!!"})
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestFactory_DeployNewAuction_InsufficientDeposit(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "user.testnet"
	m.AttachedDepositSys = types.Uint128{Hi: 0, Lo: 1}

	err := c.DeployNewAuction(DeployInput{
		Name:          "my-auction",
		EndTime:       9999999,
		Auctioneer:    "user.testnet",
		FtContract:    "ft.testnet",
		NftContract:   "nft.testnet",
		TokenId:       "token-1",
		StartingPrice: "10000",
	})
	if err == nil {
		t.Fatal("expected error for insufficient deposit, got nil")
	}
	if err.Error() != "insufficient deposit to deploy auction" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFactory_DeployNewAuction_InvalidName(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "user.testnet"
	m.AttachedDepositSys = types.Uint128{Hi: 10, Lo: 0}

	// "x" + "." + "factory.testnet" = "x.factory.testnet" — valid length
	// Empty name → ".factory.testnet" — length is fine but semantically bad
	// Let's use a name that makes total length < 2 — not really possible with factory.testnet
	// Instead test a name that makes total > 64 chars
	longName := "this-is-a-very-long-name-that-will-exceed-the-maximum-account-id-length-limit"
	err := c.DeployNewAuction(DeployInput{
		Name:          longName,
		EndTime:       9999999,
		Auctioneer:    "user.testnet",
		FtContract:    "ft.testnet",
		NftContract:   "nft.testnet",
		TokenId:       "token-1",
		StartingPrice: "10000",
	})
	if err == nil {
		t.Fatal("expected error for invalid name, got nil")
	}
	if err.Error() != "invalid subaccount name" {
		t.Errorf("unexpected error: %v", err)
	}
}
