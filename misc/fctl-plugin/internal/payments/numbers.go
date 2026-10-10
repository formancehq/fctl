package payments

import (
	"encoding/json"
	"fmt"
	"math/big"
)

func integer(value string, nonnegative bool) (json.Number, error) {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || (nonnegative && n.Sign() < 0) {
		return "", fmt.Errorf("expected %sinteger, got %q", map[bool]string{true: "nonnegative ", false: ""}[nonnegative], value)
	}
	return json.Number(n.String()), nil
}
