package main

import (
	"errors"

	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/promise"
	"github.com/vlmoon99/near-sdk-go/types"
)

type Bid struct {
	Bidder string `json:"bidder"`
	Amount string `json:"amount"`
}

type AuctionInfo struct {
	HighestBid     Bid    `json:"highest_bid"`
	AuctionEndTime uint64 `json:"auction_end_time"`
	Auctioneer     string `json:"auctioneer"`
	Claimed        bool   `json:"claimed"`
	FtContract     string `json:"ft_contract"`
	NftContract    string `json:"nft_contract"`
	TokenId        string `json:"token_id"`
}

type InitInput struct {
	EndTime       uint64 `json:"end_time"`
	Auctioneer    string `json:"auctioneer"`
	FtContract    string `json:"ft_contract"`
	NftContract   string `json:"nft_contract"`
	TokenId       string `json:"token_id"`
	StartingPrice string `json:"starting_price"`
}

type FtOnTransferInput struct {
	SenderId string `json:"sender_id"`
	Amount   string `json:"amount"`
	Msg      string `json:"msg"`
}

// @contract:state
type FtAuctionContract struct {
	HighestBid     Bid    `json:"highest_bid"`
	AuctionEndTime uint64 `json:"auction_end_time"`
	Auctioneer     string `json:"auctioneer"`
	Claimed        bool   `json:"claimed"`
	FtContract     string `json:"ft_contract"`
	NftContract    string `json:"nft_contract"`
	TokenId        string `json:"token_id"`
}

// @contract:init
func (c *FtAuctionContract) Init(input InitInput) {
	currentAccount, _ := env.GetCurrentAccountId()
	c.HighestBid = Bid{
		Bidder: currentAccount,
		Amount: input.StartingPrice,
	}
	c.AuctionEndTime = input.EndTime
	c.Auctioneer = input.Auctioneer
	c.Claimed = false
	c.FtContract = input.FtContract
	c.NftContract = input.NftContract
	c.TokenId = input.TokenId
	env.LogString("FT Auction initialized")
}

// @contract:mutating
func (c *FtAuctionContract) FtOnTransfer(input FtOnTransferInput) (string, error) {
	blockTime := env.GetBlockTimeMs()
	if blockTime >= c.AuctionEndTime {
		return "", errors.New("auction has ended")
	}

	ft, err := env.GetPredecessorAccountID()
	if err != nil {
		return "", errors.New("failed to get caller account")
	}
	if ft != c.FtContract {
		return "", errors.New("the token is not supported")
	}

	newBid, err := types.U128FromString(input.Amount)
	if err != nil {
		return "", errors.New("invalid bid amount")
	}

	currentBid, err := types.U128FromString(c.HighestBid.Amount)
	if err != nil {
		return "", errors.New("invalid current bid amount in state")
	}

	if newBid.Cmp(currentBid) <= 0 {
		return "", errors.New("you must place a higher bid")
	}

	lastBidder := c.HighestBid.Bidder
	lastBid := currentBid

	c.HighestBid = Bid{
		Bidder: input.SenderId,
		Amount: input.Amount,
	}

	ftArgs := map[string]string{
		"receiver_id": lastBidder,
		"amount":      lastBid.String(),
	}

	oneYocto := types.U64ToUint128(1)
	gas30T := uint64(types.ONE_TERA_GAS * 30)

	promise.CreateBatch(c.FtContract).
		FunctionCall("ft_transfer", ftArgs, oneYocto, gas30T)

	return "0", nil
}

// @contract:mutating
func (c *FtAuctionContract) Claim() error {
	blockTime := env.GetBlockTimeMs()
	if blockTime <= c.AuctionEndTime {
		return errors.New("auction has not ended yet")
	}

	if c.Claimed {
		return errors.New("auction has been claimed")
	}

	c.Claimed = true

	ftArgs := map[string]string{
		"receiver_id": c.Auctioneer,
		"amount":      c.HighestBid.Amount,
	}

	nftArgs := map[string]string{
		"receiver_id": c.HighestBid.Bidder,
		"token_id":    c.TokenId,
	}

	oneYocto := types.U64ToUint128(1)
	gas30T := uint64(types.ONE_TERA_GAS * 30)

	promise.CreateBatch(c.FtContract).
		FunctionCall("ft_transfer", ftArgs, oneYocto, gas30T)

	promise.CreateBatch(c.NftContract).
		FunctionCall("nft_transfer", nftArgs, oneYocto, gas30T)

	return nil
}

// @contract:view
func (c *FtAuctionContract) GetHighestBid() Bid {
	return c.HighestBid
}

// @contract:view
func (c *FtAuctionContract) GetAuctionEndTime() uint64 {
	return c.AuctionEndTime
}

// @contract:view
func (c *FtAuctionContract) GetClaimed() bool {
	return c.Claimed
}

// @contract:view
func (c *FtAuctionContract) GetAuctionInfo() AuctionInfo {
	return AuctionInfo{
		HighestBid:     c.HighestBid,
		AuctionEndTime: c.AuctionEndTime,
		Auctioneer:     c.Auctioneer,
		Claimed:        c.Claimed,
		FtContract:     c.FtContract,
		NftContract:    c.NftContract,
		TokenId:        c.TokenId,
	}
}
