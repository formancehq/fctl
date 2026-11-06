package main

import (
	"github.com/formancehq/fctl/v3/pkg/pluginsdk"
)

func main() {
	pluginsdk.Serve(&LedgerPlugin{})
}
