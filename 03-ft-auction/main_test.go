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

func setupTest(t *testing.T) *FtAuctionContract {
	t.Helper()
	m := mockSys(t)
	m.Storage = make(map[string][]byte)
	m.CurrentAccountIdSys = "auction.testnet"
	m.PredecessorAccountIdSys = "auction.testnet"
	m.BlockTimestampSys = beforeEndNs
	m.AttachedDepositSys = types.Uint128{Hi: 0, Lo: 0}
	m.Promises = nil

	c := &FtAuctionContract{}
	c.Init(InitInput{
		EndTime:       auctionEndTimeMs,
		Auctioneer:    "auctioneer.testnet",
		FtContract:    "ft.testnet",
		NftContract:   "nft.testnet",
		TokenId:       "token-1",
		StartingPrice: "10000",
	})
	return c
}

func setFtBidder(t *testing.T, sender string, amount string) {
	t.Helper()
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"
	_ = sender
	_ = amount
}

func setBlockTime(t *testing.T, ns uint64) {
	t.Helper()
	mockSys(t).BlockTimestampSys = ns
}

func TestFtAuction_Init(t *testing.T) {
	c := setupTest(t)
	info := c.GetAuctionInfo()

	if info.HighestBid.Bidder != "auction.testnet" {
		t.Errorf("initial bidder: want auction.testnet, got %s", info.HighestBid.Bidder)
	}
	if info.HighestBid.Amount != "10000" {
		t.Errorf("starting price: want 10000, got %s", info.HighestBid.Amount)
	}
	if info.AuctionEndTime != auctionEndTimeMs {
		t.Errorf("end time: want %d, got %d", auctionEndTimeMs, info.AuctionEndTime)
	}
	if info.FtContract != "ft.testnet" {
		t.Errorf("ft_contract: want ft.testnet, got %s", info.FtContract)
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

func TestFtAuction_FtOnTransfer_Success(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	refund, err := c.FtOnTransfer(FtOnTransferInput{
		SenderId: "alice.testnet",
		Amount:   "50000",
		Msg:      "",
	})
	if err != nil {
		t.Fatalf("ft_on_transfer failed: %v", err)
	}
	if refund != "0" {
		t.Errorf("expected refund=0, got %s", refund)
	}

	bid := c.GetHighestBid()
	if bid.Bidder != "alice.testnet" || bid.Amount != "50000" {
		t.Errorf("expected alice/50000, got %s/%s", bid.Bidder, bid.Amount)
	}
}

func TestFtAuction_FtOnTransfer_TooLow(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, _ = c.FtOnTransfer(FtOnTransferInput{SenderId: "alice.testnet", Amount: "50000", Msg: ""})

	_, err := c.FtOnTransfer(FtOnTransferInput{
		SenderId: "bob.testnet",
		Amount:   "5000",
		Msg:      "",
	})
	if err == nil {
		t.Fatal("expected error for bid too low, got nil")
	}
	if err.Error() != "you must place a higher bid" {
		t.Errorf("unexpected error: %v", err)
	}

	bid := c.GetHighestBid()
	if bid.Bidder != "alice.testnet" {
		t.Errorf("bidder should still be alice, got %s", bid.Bidder)
	}
}

func TestFtAuction_FtOnTransfer_BelowStartingPrice(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, err := c.FtOnTransfer(FtOnTransferInput{
		SenderId: "alice.testnet",
		Amount:   "5000",
		Msg:      "",
	})
	if err == nil {
		t.Fatal("expected error for bid below starting price, got nil")
	}
}

func TestFtAuction_FtOnTransfer_WrongToken(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "other-ft.testnet"

	_, err := c.FtOnTransfer(FtOnTransferInput{
		SenderId: "alice.testnet",
		Amount:   "50000",
		Msg:      "",
	})
	if err == nil {
		t.Fatal("expected error for unsupported token, got nil")
	}
	if err.Error() != "the token is not supported" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFtAuction_FtOnTransfer_AfterEnd(t *testing.T) {
	c := setupTest(t)
	setBlockTime(t, afterEndNs)

	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, err := c.FtOnTransfer(FtOnTransferInput{
		SenderId: "alice.testnet",
		Amount:   "50000",
		Msg:      "",
	})
	if err == nil {
		t.Fatal("expected error for bid after auction end, got nil")
	}
	if err.Error() != "auction has ended" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFtAuction_Claim_BeforeEnd(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, _ = c.FtOnTransfer(FtOnTransferInput{SenderId: "alice.testnet", Amount: "50000", Msg: ""})

	setBlockTime(t, beforeEndNs)
	err := c.Claim()
	if err == nil {
		t.Fatal("expected error for claim before end, got nil")
	}
	if err.Error() != "auction has not ended yet" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFtAuction_Claim_Success(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, _ = c.FtOnTransfer(FtOnTransferInput{SenderId: "alice.testnet", Amount: "50000", Msg: ""})

	setBlockTime(t, afterEndNs)
	if err := c.Claim(); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if !c.GetClaimed() {
		t.Error("expected claimed=true")
	}
}

func TestFtAuction_Claim_AlreadyClaimed(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, _ = c.FtOnTransfer(FtOnTransferInput{SenderId: "alice.testnet", Amount: "50000", Msg: ""})

	setBlockTime(t, afterEndNs)
	_ = c.Claim()

	err := c.Claim()
	if err == nil {
		t.Fatal("expected error for double claim, got nil")
	}
	if err.Error() != "auction has been claimed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFtAuction_FullLifecycle(t *testing.T) {
	c := setupTest(t)
	m := mockSys(t)
	m.PredecessorAccountIdSys = "ft.testnet"

	_, _ = c.FtOnTransfer(FtOnTransferInput{SenderId: "alice.testnet", Amount: "50000", Msg: ""})
	_, _ = c.FtOnTransfer(FtOnTransferInput{SenderId: "bob.testnet", Amount: "60000", Msg: ""})

	_, err := c.FtOnTransfer(FtOnTransferInput{SenderId: "alice.testnet", Amount: "50000", Msg: ""})
	if err == nil {
		t.Fatal("alice's low re-bid should have failed")
	}

	bid := c.GetHighestBid()
	if bid.Bidder != "bob.testnet" || bid.Amount != "60000" {
		t.Errorf("expected bob/60000, got %s/%s", bid.Bidder, bid.Amount)
	}

	setBlockTime(t, afterEndNs)

	m.PredecessorAccountIdSys = "ft.testnet"
	_, err = c.FtOnTransfer(FtOnTransferInput{SenderId: "charlie.testnet", Amount: "999999", Msg: ""})
	if err == nil {
		t.Fatal("bid after end should fail")
	}

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
