package main

import (
	"fmt"
	"os"

	"github.com/formancehq/fctl/v4/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
