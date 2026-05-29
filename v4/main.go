package main

import "github.com/formancehq/fctl/v4/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
