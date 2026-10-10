package payments

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func metadataFlag() pluginsdk.FlagSpec {
	return commandapi.StringFlag("metadata", "Metadata as JSON object or CSV key=value")
}

func paymentMetadata(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if len(r.Args) > 1 && r.Flags["metadata"] != "" {
		return nil, fmt.Errorf("provide metadata once")
	}
	var metadata map[string]string
	if len(r.Args) > 1 {
		var err error
		metadata, err = positionalMetadata(r.Args[1:])
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		metadata, err = commandapi.Pairs(r.Flags["metadata"])
		if err != nil {
			return nil, err
		}
	}
	if len(metadata) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}
	if r.CommandPath[1] == "bank-accounts" {
		return json.Marshal(map[string]any{"metadata": metadata})
	}
	return json.Marshal(metadata)
}

func paymentMetadataOperations() []commandapi.Operation {
	operations := []commandapi.Operation{}
	for _, group := range []string{"payments", "bank-accounts"} {
		command, path := "set-metadata", "payments/$0/metadata"
		if group == "bank-accounts" {
			command, path = "update-metadata", "bank-accounts/$0/metadata"
		}
		op := commandapi.Confirmed(commandapi.Leaf(group+" "+command, command+" <id> [key=value...]", http.MethodPatch, path, 1, 1000, metadataFlag()))
		op.Body = paymentMetadata
		operations = append(operations, op)
	}
	return operations
}

func positionalMetadata(args []string) (map[string]string, error) {
	metadata := map[string]string{}
	for _, v := range args {
		key, value, ok := strings.Cut(v, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("expected metadata key=value")
		}
		if _, duplicate := metadata[key]; duplicate {
			return nil, fmt.Errorf("duplicate metadata key %q", key)
		}
		metadata[key] = value
	}
	return metadata, nil
}
