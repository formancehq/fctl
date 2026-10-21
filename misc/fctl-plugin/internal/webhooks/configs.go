package webhooks

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func configOperations() []command.Operation {
	listOp := command.Leaf("list", "list", http.MethodGet, "configs", 0, 0, command.StringFlag("config-id", "Config ID filter"), command.StringFlag("endpoint", "Endpoint URL filter"))
	listOp.Query = map[string]string{"config-id": "id", "endpoint": "endpoint"}
	create := command.Confirmed(command.Leaf("create", "create <endpoint> <event-type>...", http.MethodPost, "configs", 2, 1000, command.StringFlag("secret", "Optional base64 signing secret (24 bytes)")))
	create.Body = webhookConfig
	update := command.Confirmed(command.Leaf("update", "update <config-id> <endpoint> <event-type>...", http.MethodPut, "configs/$0", 3, 1001, command.StringFlag("secret", "Optional signing secret")))
	update.Body = webhookConfig
	secret := command.Confirmed(command.Leaf("change-secret", "change-secret <config-id> [secret]", http.MethodPut, "configs/$0/secret/change", 1, 2))
	secret.Body = changeSecretBody
	return []command.Operation{listOp, create, update, secret,
		command.Confirmed(command.Leaf("activate", "activate <config-id>", http.MethodPut, "configs/$0/activate", 1, 1)),
		command.Confirmed(command.Leaf("deactivate", "deactivate <config-id>", http.MethodPut, "configs/$0/deactivate", 1, 1)),
		command.Confirmed(command.Leaf("delete", "delete <config-id>", http.MethodDelete, "configs/$0", 1, 1)),
		command.Confirmed(command.Leaf("test", "test <config-id>", http.MethodGet, "configs/$0/test", 1, 1)),
	}
}

func webhookConfig(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil {
		return nil, fmt.Errorf("use endpoint and event-type arguments")
	}
	offset := 0
	if r.CommandPath[len(r.CommandPath)-1] == "update" {
		offset = 1
	}
	endpoint := r.Args[offset]
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return nil, fmt.Errorf("webhook endpoint must be an absolute HTTP(S) URL")
	}
	for _, event := range r.Args[offset+1:] {
		if strings.TrimSpace(event) == "" {
			return nil, fmt.Errorf("event types cannot be empty")
		}
	}
	body := map[string]any{"endpoint": endpoint, "eventTypes": r.Args[offset+1:]}
	secret := r.Flags["secret"]
	if err := signingSecret(secret); err != nil {
		return nil, err
	}
	if offset == 0 || secret != "" || r.ChangedFlags["secret"] {
		body["secret"] = secret
	}
	return json.Marshal(body)
}

func signingSecret(value string) error {
	if value == "" {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != 24 {
		return fmt.Errorf("signing secret must be base64 encoding of 24 bytes")
	}
	return nil
}

func changeSecretBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	value := ""
	if len(r.Args) == 2 {
		value = r.Args[1]
	}
	if err := signingSecret(value); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{"secret": value})
}
