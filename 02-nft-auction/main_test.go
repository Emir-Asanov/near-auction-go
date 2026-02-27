package main

import (
	"testing"

	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/system"
	"github.com/vlmoon99/near-sdk-go/types"
)

const (
	auctionEndTimeMs = uint64(1000)
	beforeEndNs      = uint64(500) * 1_000_000
	afterEndNs       = uint64(2000) * 1_000_000
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

func setupTest(t *testing.T) *NftAuctionContract {
	t.Helper()
	m := mockSys(t)
	m.Storage = make(map[string][]byte)
	m.CurrentAccountIdSys = "auction.testnet"
	m.PredecessorAccountIdSys = "auction.testnet"
	m.BlockTimestampSys = beforeEndNs
	m.AttachedDepositSys = types.Uint128{Hi: 0, Lo: 0}
	m.Promises = nil

	c := &NftAuctionContract{}
	c.Init(InitInput{
		EndTime:     auctionEndTimeMs,
		Auctioneer:  "auctioneer.testnet",
		NftContract: "nft.testnet",
		TokenId:     "token-1",
	})
	return c
}

func setBidder(t *testing.T, account string, depositLo uint64) {
	t.Helper()
	m := mockSys(t)
	m.PredecessorAccountIdSys = account
	m.AttachedDepositSys = types.Uint128{Hi: 0, Lo: depositLo}
}

func setBlockTime(t *testing.T, ns uint64) {
	t.Helper()
	mockSys(t).BlockTimestampSys = ns
}

func TestNftAuction_Init(t *testing.T) {
	c := setupTest(t)
	info := c.GetAuctionInfo()

	if info.HighestBid.Bidder != "auction.testnet" {
		t.Errorf("initial bidder: want auction.testnet, got %s", info.HighestBid.Bidder)
	}
	if info.HighestBid.Amount != "1" {
		t.Errorf("initial amount: want 1, got %s", info.HighestBid.Amount)
	}
	if info.AuctionEndTime != auctionEndTimeMs {
		t.Errorf("end time: want %d, got %d", auctionEndTimeMs, info.AuctionEndTime)
	}
	if info.Auctioneer != "auctioneer.testnet" {
		t.Errorf("auctioneer: want auctioneer.testnet, got %s", info.Auctioneer)
	}
	if info.NftContract != "nft.testnet" {
		t.Errorf("nft_contract: want nft.testnet, got %s", info.NftContract)
	}
	if info.TokenId != "token-1" {
		t.Errorf("token_id: want token-1, got %s", info.TokenId)
	}
	if info.Claimed {
		t.Error("claimed should be false after init")
	}
}

func TestNftAuction_Bid_Success(t *testing.T) {
	c := setupTest(t)

	setBidder(t, "alice.testnet", 100)
	if err := c.Bid(); err != nil {
		t.Fatalf("bid failed: %v", err)
	}

	bid := c.GetHighestBid()
	if bid.Bidder != "alice.testnet" || bid.Amount != "100" {
		t.Errorf("expected alice/100, got %s/%s", bid.Bidder, bid.Amount)
	}
}

func TestNftAuction_Bid_TooLow(t *testing.T) {
	c := setupTest(t)

	setBidder(t, "alice.testnet", 100)
	_ = c.Bid()

	setBidder(t, "bob.testnet", 50)
	if err := c.Bid(); err == nil {
		t.Fatal("expected error for bid too low, got nil")
	}
}

func TestNftAuction_Bid_AfterEnd(t *testing.T) {
	c := setupTest(t)

	setBlockTime(t, afterEndNs)
	setBidder(t, "alice.testnet", 100)

	err := c.Bid()
	if err == nil {
		t.Fatal("expected error for bid after auction end, got nil")
	}
	if err.Error() != "auction has ended" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNftAuction_Claim_BeforeEnd(t *testing.T) {
	c := setupTest(t)

	setBidder(t, "alice.testnet", 100)
	_ = c.Bid()

	setBlockTime(t, beforeEndNs)
	err := c.Claim()
	if err == nil {
		t.Fatal("expected error for claim before end, got nil")
	}
	if err.Error() != "auction has not ended yet" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNftAuction_Claim_Success(t *testing.T) {
	c := setupTest(t)

	setBidder(t, "alice.testnet", 100)
	_ = c.Bid()

	setBlockTime(t, afterEndNs)
	if err := c.Claim(); err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	if !c.GetClaimed() {
		t.Error("expected claimed=true")
	}
}

func TestNftAuction_Claim_AlreadyClaimed(t *testing.T) {
	c := setupTest(t)

	setBidder(t, "alice.testnet", 100)
	_ = c.Bid()

	setBlockTime(t, afterEndNs)
	_ = c.Claim()

	err := c.Claim()
	if err == nil {
		t.Fatal("expected error for double claim, got nil")
	}
	if err.Error() != "auction has already been claimed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNftAuction_FullLifecycle(t *testing.T) {
	c := setupTest(t)

	setBidder(t, "alice.testnet", 100)
	if err := c.Bid(); err != nil {
		t.Fatalf("alice bid failed: %v", err)
	}

	setBidder(t, "bob.testnet", 300)
	if err := c.Bid(); err != nil {
		t.Fatalf("bob bid failed: %v", err)
	}

	setBidder(t, "alice.testnet", 200)
	if err := c.Bid(); err == nil {
		t.Fatal("alice's low bid should have failed")
	}

	bid := c.GetHighestBid()
	if bid.Bidder != "bob.testnet" || bid.Amount != "300" {
		t.Errorf("expected bob/300, got %s/%s", bid.Bidder, bid.Amount)
	}

	setBlockTime(t, afterEndNs)

	if err := c.Claim(); err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	info := c.GetAuctionInfo()
	if !info.Claimed {
		t.Error("auction should be claimed")
	}
	if info.HighestBid.Bidder != "bob.testnet" {
		t.Errorf("winner should be bob, got %s", info.HighestBid.Bidder)
	}
}
