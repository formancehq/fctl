package wallets

import (
	"encoding/json"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func metadataFlag() pluginsdk.FlagSpec {
	return commandapi.StringFlag("metadata", "Metadata as JSON object or CSV key=value")
}

func ikFlag() pluginsdk.FlagSpec { return commandapi.StringFlag("ik", "Idempotency key") }

func validateMetadata(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	_, err := commandapi.Pairs(r.Flags["metadata"])
	return nil, err
}
