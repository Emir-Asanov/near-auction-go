package main

import (
	_ "embed"
	"encoding/base64"
	"errors"

	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/promise"
	"github.com/vlmoon99/near-sdk-go/types"
)

//go:embed auction.wasm
var embeddedAuctionWasm []byte

const nearPerStorageByte = uint64(10_000_000_000_000_000_000)

type DeployInput struct {
	Name          string `json:"name"`
	EndTime       uint64 `json:"end_time"`
	Auctioneer    string `json:"auctioneer"`
	FtContract    string `json:"ft_contract"`
	NftContract   string `json:"nft_contract"`
	TokenId       string `json:"token_id"`
	StartingPrice string `json:"starting_price"`
}

type AuctionInitArgs struct {
	EndTime       uint64 `json:"end_time"`
	Auctioneer    string `json:"auctioneer"`
	FtContract    string `json:"ft_contract"`
	NftContract   string `json:"nft_contract"`
	TokenId       string `json:"token_id"`
	StartingPrice string `json:"starting_price"`
}

type DeployCallbackInput struct {
	Account  string `json:"account"`
	User     string `json:"user"`
	Attached string `json:"attached"`
}

type UpdateCodeInput struct {
	Code string `json:"code"`
}

// @contract:state
type FactoryContract struct {
	Code []byte `json:"code"`
}

// @contract:init
func (c *FactoryContract) Init() {
	c.Code = embeddedAuctionWasm
	env.LogString("Factory initialized")
}

// @contract:payable min_deposit=0
func (c *FactoryContract) DeployNewAuction(input DeployInput) error {
	currentAccount, err := env.GetCurrentAccountId()
	if err != nil {
		return errors.New("failed to get current account")
	}

	subaccount := input.Name + "." + currentAccount
	if len(subaccount) < 2 || len(subaccount) > 64 {
		return errors.New("invalid subaccount name")
	}

	attached, err := env.GetAttachedDeposit()
	if err != nil {
		return errors.New("failed to get attached deposit")
	}

	storageCost, err := types.U64ToUint128(nearPerStorageByte).SafeMul64(uint64(len(c.Code)))
	if err != nil {
		return errors.New("storage cost overflow")
	}
	extraDeposit, _ := types.U128FromString("100000000000000000000000")
	minimum, err := storageCost.Add(extraDeposit)
	if err != nil {
		return errors.New("minimum deposit overflow")
	}

	if attached.Cmp(minimum) < 0 {
		return errors.New("insufficient deposit to deploy auction")
	}

	caller, err := env.GetPredecessorAccountID()
	if err != nil {
		return errors.New("failed to get caller")
	}

	initArgs := AuctionInitArgs{
		EndTime:       input.EndTime,
		Auctioneer:    input.Auctioneer,
		FtContract:    input.FtContract,
		NftContract:   input.NftContract,
		TokenId:       input.TokenId,
		StartingPrice: input.StartingPrice,
	}

	callbackArgs := DeployCallbackInput{
		Account:  subaccount,
		User:     caller,
		Attached: attached.String(),
	}

	zero := types.Uint128{Hi: 0, Lo: 0}
	gas5T := uint64(types.ONE_TERA_GAS * 5)

	promise.CreateBatch(subaccount).
		CreateAccount().
		Transfer(attached).
		DeployContract(c.Code).
		FunctionCall("init", initArgs, zero, gas5T).
		Then(currentAccount).
		FunctionCall("deploy_new_auction_callback", callbackArgs, zero, gas5T).
		Value()

	return nil
}

// @contract:view
// @contract:promise_callback
func (c *FactoryContract) DeployNewAuctionCallback(input DeployCallbackInput, result promise.PromiseResult) bool {
	if result.Success {
		env.LogString("Correctly created and deployed to " + input.Account)
		return true
	}

	env.LogString("Error creating " + input.Account + ", returning " + input.Attached + " to " + input.User)

	attached, err := types.U128FromString(input.Attached)
	if err != nil {
		env.LogString("Failed to parse attached amount")
		return false
	}

	promise.CreateBatch(input.User).Transfer(attached)

	return false
}

// @contract:mutating
func (c *FactoryContract) UpdateAuctionContract(input UpdateCodeInput) error {
	caller, err := env.GetPredecessorAccountID()
	if err != nil {
		return errors.New("failed to get caller")
	}
	current, err := env.GetCurrentAccountId()
	if err != nil {
		return errors.New("failed to get current account")
	}
	if caller != current {
		return errors.New("only the contract itself can call this method")
	}

	code, err := base64.StdEncoding.DecodeString(input.Code)
	if err != nil {
		return errors.New("invalid base64 code")
	}

	c.Code = code
	return nil
}

// @contract:view
func (c *FactoryContract) GetCodeSize() int {
	return len(c.Code)
}
