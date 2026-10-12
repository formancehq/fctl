package main

import (
	"github.com/formancehq/fctl/pkg/pluginsdk/transport"

	legacy "github.com/formancehq/fctl/misc/fctl-plugin"
)

func main() { transport.Serve(legacy.New) }
