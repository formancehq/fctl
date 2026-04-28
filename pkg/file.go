package fctl

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func ReadFile(cmd *cobra.Command, where string) (string, error) {
	var ret string
	if where == "-" {
		if NeedConfirm(cmd) {
			return "", errors.New("You need to use --confirm flag to use stdin")
		}
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil && err != io.EOF {
			return "", errors.Wrapf(err, "reading stdin")
		}

		ret = string(data)
	} else {
		data, err := os.ReadFile(filepath.Clean(where))
		if err != nil {
			return "", errors.Wrapf(err, "reading file %s", where)
		}
		ret = string(data)
	}
	return ret, nil
}
