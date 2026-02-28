package core

// Bid represents a single bid placed in an auction.
type Bid struct {
	Bidder string `json:"bidder"`
	Amount string `json:"amount"`
}
