package main

import (
	"errors"

	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/promise"
	"github.com/vlmoon99/near-sdk-go/types"
)

// Bid stores the current highest bidder and their bid amount (in yoctoNEAR).
type Bid struct {
	Bidder string `json:"bidder"`
	Amount string `json:"amount"` // yoctoNEAR as decimal string
}

// AuctionInfo is a flat view of the contract state, returned by GetAuctionInfo.
type AuctionInfo struct {
	HighestBid     Bid    `json:"highest_bid"`
	AuctionEndTime uint64 `json:"auction_end_time"` // milliseconds
	Auctioneer     string `json:"auctioneer"`
	Claimed        bool   `json:"claimed"`
}

// InitInput is the parameter for the Init constructor.
type InitInput struct {
	EndTime    uint64 `json:"end_time"`   // auction end time in milliseconds
	Auctioneer string `json:"auctioneer"` // account that receives winning bid
}

// @contract:state
type AuctionContract struct {
	HighestBid     Bid    `json:"highest_bid"`
	AuctionEndTime uint64 `json:"auction_end_time"` // milliseconds
	Auctioneer     string `json:"auctioneer"`
	Claimed        bool   `json:"claimed"`
}

// @contract:init
func (c *AuctionContract) Init(input InitInput) {
	currentAccount, _ := env.GetCurrentAccountId()
	c.HighestBid = Bid{
		Bidder: currentAccount,
		Amount: "1", // initial bid: 1 yoctoNEAR
	}
	c.AuctionEndTime = input.EndTime
	c.Auctioneer = input.Auctioneer
	c.Claimed = false
	env.LogString("Auction initialized")
}

// Bid places a new bid. The caller must attach more NEAR than the current
// highest bid. The previous highest bidder is automatically refunded.
//
// @contract:mutating
func (c *AuctionContract) Bid() error {
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

	c.HighestBid = Bid{
		Bidder: caller,
		Amount: deposit.String(),
	}

	// Refund the previous highest bidder
	promise.CreateBatch(lastBidder).Transfer(lastBid)

	return nil
}

// Claim transfers the highest bid to the auctioneer. Can only be called after
// the auction has ended and only once.
//
// @contract:mutating
func (c *AuctionContract) Claim() error {
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

	// Transfer winning bid to auctioneer
	promise.CreateBatch(c.Auctioneer).Transfer(winningBid)

	return nil
}

// GetHighestBid returns the current highest bid (bidder + amount).
//
// @contract:view
func (c *AuctionContract) GetHighestBid() Bid {
	return c.HighestBid
}

// GetAuctionEndTime returns the auction end time in milliseconds.
//
// @contract:view
func (c *AuctionContract) GetAuctionEndTime() uint64 {
	return c.AuctionEndTime
}

// GetAuctioneer returns the account that receives the winning bid.
//
// @contract:view
func (c *AuctionContract) GetAuctioneer() string {
	return c.Auctioneer
}

// GetClaimed returns whether the auction has been claimed.
//
// @contract:view
func (c *AuctionContract) GetClaimed() bool {
	return c.Claimed
}

// GetAuctionInfo returns the full auction state in one call.
//
// @contract:view
func (c *AuctionContract) GetAuctionInfo() AuctionInfo {
	return AuctionInfo{
		HighestBid:     c.HighestBid,
		AuctionEndTime: c.AuctionEndTime,
		Auctioneer:     c.Auctioneer,
		Claimed:        c.Claimed,
	}
}
