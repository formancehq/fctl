package connections

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const initialCursorFlag = "initial-cursor"

func validateSelectors(cmd *cobra.Command) error {
	if cmd.Flags().Changed(versionFlag) && cmd.Flags().Changed(channelFlag) {
		return fmt.Errorf("--version and --channel are mutually exclusive; select exactly one")
	}
	for _, flag := range []string{versionFlag, channelFlag} {
		if cmd.Flags().Changed(flag) && strings.TrimSpace(fctl.GetString(cmd, flag)) == "" {
			return fmt.Errorf("--%s cannot be empty", flag)
		}
	}
	if cmd.Flags().Changed(channelFlag) {
		if _, ok := channelPrereleases[fctl.GetString(cmd, channelFlag)]; !ok {
			return fmt.Errorf("unsupported --channel; use stable, rc, beta, or alpha")
		}
	}
	return nil
}

func initialCursor(cmd *cobra.Command) ([]byte, error) {
	if !cmd.Flags().Changed(initialCursorFlag) {
		return nil, nil
	}
	raw := fctl.GetString(cmd, initialCursorFlag)
	decoded, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || raw == "" || base64.StdEncoding.EncodeToString(decoded) != raw {
		return nil, fmt.Errorf("--initial-cursor must be non-empty canonical base64")
	}
	return decoded, nil
}
