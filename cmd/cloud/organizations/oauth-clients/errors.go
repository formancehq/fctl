package oauth_clients

import "fmt"

var (
	ErrMissingClientID = fmt.Errorf("client_id must be provided")
)
