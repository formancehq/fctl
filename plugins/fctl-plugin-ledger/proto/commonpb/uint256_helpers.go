package commonpb

import (
	"math/big"

	"github.com/holiman/uint256"
)

// NewUint256 creates a new Uint256 proto message from a uint256.Int.
func NewUint256(v *uint256.Int) *Uint256 {
	return &Uint256{
		V0: v[0],
		V1: v[1],
		V2: v[2],
		V3: v[3],
	}
}

// IsZero returns true if all 4 limbs are zero.
func (u *Uint256) IsZero() bool {
	if u == nil {
		return true
	}
	return u.V0 == 0 && u.V1 == 0 && u.V2 == 0 && u.V3 == 0
}

// Dec returns the decimal string representation of the value.
func (u *Uint256) Dec() string {
	if u == nil {
		return "0"
	}
	var v uint256.Int
	v[0] = u.V0
	v[1] = u.V1
	v[2] = u.V2
	v[3] = u.V3
	return v.Dec()
}

// ToBigInt converts the Uint256 to a *big.Int.
func (u *Uint256) ToBigInt() *big.Int {
	if u == nil || u.IsZero() {
		return new(big.Int)
	}
	var v uint256.Int
	v[0] = u.V0
	v[1] = u.V1
	v[2] = u.V2
	v[3] = u.V3
	return v.ToBig()
}
