package commonpb

import (
	"math/big"

	"github.com/holiman/uint256"
)

// NewPosting creates a new Posting from the given parameters.
func NewPosting(source, destination, asset string, amount *big.Int) *Posting {
	var u uint256.Int
	if overflow := u.SetFromBig(amount); overflow {
		panic("commonpb.NewPosting: amount exceeds 256 bits")
	}
	return &Posting{
		Source:      source,
		Destination: destination,
		Amount:      NewUint256(&u),
		Asset:       asset,
	}
}
