package main

import (
	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func main() {
	pluginsdk.Serve(&LedgerPlugin{})
}
