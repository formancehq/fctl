package wallets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

type subject struct{ kind, id, name, balance string }

func parseSubject(value string) (subject, error) {
	if id, found := strings.CutPrefix(value, "account="); found {
		if err := commandapi.Identifier(id); err != nil {
			return subject{}, err
		}
		return subject{kind: "ACCOUNT", id: id}, nil
	}
	if !strings.HasPrefix(value, "wallet=") {
		return subject{}, fmt.Errorf("invalid subject %q", value)
	}
	definition, balance, hasBalance := strings.Cut(strings.TrimPrefix(value, "wallet="), "/")
	if !hasBalance {
		balance = "main"
	}
	if err := commandapi.Identifier(balance); err != nil {
		return subject{}, err
	}
	kind, id, found := strings.Cut(definition, ":")
	if !found || (kind != "id" && kind != "name") || commandapi.Identifier(id) != nil {
		return subject{}, fmt.Errorf("wallet subject requires id:ID or name:NAME")
	}
	out := subject{kind: "WALLET", balance: balance}
	if kind == "name" {
		out.name = id
	} else {
		out.id = id
	}
	return out, nil
}

func resolveSubject(ctx context.Context, client *httpclient.Client, value string) (map[string]string, error) {
	s, err := parseSubject(value)
	if err != nil {
		return nil, err
	}
	if s.name != "" {
		s.id, err = walletByName(ctx, client, s.name)
		if err != nil {
			return nil, err
		}
	}
	out := map[string]string{"type": s.kind, "identifier": s.id}
	if s.balance != "" {
		out["balance"] = s.balance
	}
	return out, nil
}

func validateSubjects(values []string) error {
	for _, value := range values {
		if _, err := parseSubject(value); err != nil {
			return err
		}
	}
	return nil
}

func walletSubjectBody(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, body json.RawMessage) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	if r.CommandPath[1] == "credit" {
		sources, err := walletSources(ctx, client, r.Flags["source"])
		if err != nil {
			return nil, err
		}
		object["sources"] = sources
	} else if r.Flags["destination"] != "" {
		destination, err := resolveSubject(ctx, client, r.Flags["destination"])
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(destination)
		if err != nil {
			return nil, err
		}
		object["destination"] = encoded
	}
	return json.Marshal(object)
}

func walletSources(ctx context.Context, client *httpclient.Client, value string) (json.RawMessage, error) {
	values, err := commandapi.List(value)
	if err != nil {
		return nil, err
	}
	sources := []map[string]string{}
	for _, value := range values {
		s, err := resolveSubject(ctx, client, value)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return json.Marshal(sources)
}
