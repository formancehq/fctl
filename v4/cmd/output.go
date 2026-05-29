package cmd

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/render"
)

func writeStructuredOutput(cmd *cobra.Command, value any) (bool, error) {
	format, err := outputFormat(cmd)
	if err != nil {
		return false, err
	}
	switch format {
	case "json":
		return true, render.JSON(cmd.OutOrStdout(), value)
	case "yaml":
		return true, render.YAML(cmd.OutOrStdout(), value)
	default:
		return false, nil
	}
}
