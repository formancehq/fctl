package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func meManifest() pluginsdk.CommandSpec {
	return pluginsdk.CommandSpec{Use: "me", Short: "Read the current user and manage invitations", Subcommands: []pluginsdk.CommandSpec{
		leaf("info", "Show the connected user", 0, false),
		{Use: "invitations", Short: "Manage personal invitations", Subcommands: []pluginsdk.CommandSpec{
			withFlags(leaf("list", "List invitations", 0, false), stringFlag("status", "Invitation status")),
			leaf("accept INVITATION", "Accept an invitation", 1, false), leaf("decline INVITATION", "Decline an invitation", 1, true),
		}},
	}}
}
func executeMe(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.CommandPath[2] == "info" {
		return c.Do(ctx, http.MethodGet, "/me", nil, nil, nil)
	}
	switch r.CommandPath[3] {
	case "list":
		q, err := queryFlags(r, map[string]string{"status": "status"})
		if err != nil {
			return nil, err
		}
		if organization := r.Context["organization"]; organization != "" {
			q.Set("organization", organization)
		}
		return c.Do(ctx, http.MethodGet, "/me/invitations", q, nil, nil)
	case "accept":
		return c.Do(ctx, http.MethodPost, apiPath("me", "invitations", r.Args[0], "accept"), nil, nil, nil)
	case "decline":
		return c.Do(ctx, http.MethodPost, apiPath("me", "invitations", r.Args[0], "reject"), nil, nil, nil)
	default:
		return nil, fmt.Errorf("unsupported invitation command")
	}
}
