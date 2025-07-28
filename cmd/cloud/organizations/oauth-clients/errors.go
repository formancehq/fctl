package oauth

import "fmt"

var (
	ErrMissingClientID = fmt.Errorf("client_id must be provided")
)
