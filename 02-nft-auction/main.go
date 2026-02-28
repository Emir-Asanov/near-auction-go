package main

import (
	"errors"

	"github.com/emirsuyunasanov/near-auction-go/core"
	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/promise"
	"github.com/vlmoon99/near-sdk-go/types"
)

type AuctionInfo struct {
	HighestBid     core.Bid `json:"highest_bid"`
	AuctionEndTime uint64   `json:"auction_end_time"`
	Auctioneer     string   `json:"auctioneer"`
	Claimed        bool     `json:"claimed"`
	NftContract    string   `json:"nft_contract"`
	TokenId        string   `json:"token_id"`
}

type InitInput struct {
	EndTime     uint64 `json:"end_time"`
	Auctioneer  string `json:"auctioneer"`
	NftContract string `json:"nft_contract"`
	TokenId     string `json:"token_id"`
}

// @contract:state
type NftAuctionContract struct {
	HighestBid     core.Bid `json:"highest_bid"`
	AuctionEndTime uint64   `json:"auction_end_time"`
	Auctioneer     string   `json:"auctioneer"`
	Claimed        bool     `json:"claimed"`
	NftContract    string   `json:"nft_contract"`
	TokenId        string   `json:"token_id"`
}

// @contract:init
func (c *NftAuctionContract) Init(input InitInput) {
	currentAccount, _ := env.GetCurrentAccountId()
	c.HighestBid = core.Bid{
		Bidder: currentAccount,
		Amount: "1",
	}
	c.AuctionEndTime = input.EndTime
	c.Auctioneer = input.Auctioneer
	c.Claimed = false
	c.NftContract = input.NftContract
	c.TokenId = input.TokenId
	env.LogString("NFT Auction initialized")
}

// @contract:mutating
func (c *NftAuctionContract) Bid() error {
	blockTime := env.GetBlockTimeMs()
	if blockTime >= c.AuctionEndTime {
		return errors.New("auction has ended")
	}

	deposit, err := env.GetAttachedDeposit()
	if err != nil {
		return errors.New("failed to get attached deposit")
	}

	currentBid, err := types.U128FromString(c.HighestBid.Amount)
	if err != nil {
		return errors.New("invalid current bid amount in state")
	}

	if deposit.Cmp(currentBid) <= 0 {
		return errors.New("you must place a higher bid")
	}

	caller, err := env.GetPredecessorAccountID()
	if err != nil {
		return errors.New("failed to get caller account")
	}

	lastBidder := c.HighestBid.Bidder
	lastBid := currentBid

	c.HighestBid = core.Bid{
		Bidder: caller,
		Amount: deposit.String(),
	}

	promise.CreateBatch(lastBidder).Transfer(lastBid)

	return nil
}

// @contract:mutating
func (c *NftAuctionContract) Claim() error {
	blockTime := env.GetBlockTimeMs()
	if blockTime <= c.AuctionEndTime {
		return errors.New("auction has not ended yet")
	}

	if c.Claimed {
		return errors.New("auction has already been claimed")
	}

	c.Claimed = true

	winningBid, err := types.U128FromString(c.HighestBid.Amount)
	if err != nil {
		return errors.New("invalid winning bid amount in state")
	}

	nftArgs := map[string]string{
		"receiver_id": c.HighestBid.Bidder,
		"token_id":    c.TokenId,
	}

	oneYocto := types.U64ToUint128(1)
	gas30T := uint64(types.ONE_TERA_GAS * 30)

	promise.CreateBatch(c.Auctioneer).
		Transfer(winningBid).
		Then(c.NftContract).
		FunctionCall("nft_transfer", nftArgs, oneYocto, gas30T).
		Value()

	return nil
}

// @contract:view
func (c *NftAuctionContract) GetHighestBid() core.Bid {
	return c.HighestBid
}

// @contract:view
func (c *NftAuctionContract) GetAuctionEndTime() uint64 {
	return c.AuctionEndTime
}

// @contract:view
func (c *NftAuctionContract) GetAuctioneer() string {
	return c.Auctioneer
}

// @contract:view
func (c *NftAuctionContract) GetClaimed() bool {
	return c.Claimed
}

// @contract:view
func (c *NftAuctionContract) GetAuctionInfo() AuctionInfo {
	return AuctionInfo{
		HighestBid:     c.HighestBid,
		AuctionEndTime: c.AuctionEndTime,
		Auctioneer:     c.Auctioneer,
		Claimed:        c.Claimed,
		NftContract:    c.NftContract,
		TokenId:        c.TokenId,
	}
}
