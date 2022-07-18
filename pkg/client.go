package fctl

import (
	ledgerclient "github.com/numary/numary-sdk-go"
)

type Client struct {
	Ledger *ledgerclient.APIClient
}

func NewClient(getProfile GetCurrentProfile) *Client {

	profile, _, _ := getProfile()

	config := ledgerclient.NewConfiguration()
	config.Servers = ledgerclient.ServerConfigurations{{
		URL: profile.URI,
	}}

	lc := ledgerclient.NewAPIClient(config)

	client := &Client{
		Ledger: lc,
	}

	return client
}
