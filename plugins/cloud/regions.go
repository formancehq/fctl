package cloud

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

func regionsManifest() pluginsdk.CommandSpec {
	return pluginsdk.CommandSpec{Use: "regions", Short: "Manage Cloud regions", Subcommands: []pluginsdk.CommandSpec{
		leaf("list", "List regions", 0, false), leaf("show REGION", "Show a region", 1, false), leaf("versions REGION", "List region versions", 1, false),
		withFlags(optionalArg(leafBody("create [NAME]", "Create a private region; save the returned secret", 1, false)), stringFlag("name", "Region name")),
		leaf("delete REGION", "Delete a region", 1, true),
	}}
}
func executeRegions(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	org, err := requireOrg(ctx, c, r)
	if err != nil {
		return nil, err
	}
	method, path := http.MethodGet, apiPath("organizations", org, "regions")
	var body json.RawMessage
	switch r.CommandPath[2] {
	case "show":
		path = apiPath("organizations", org, "regions", r.Args[0])
	case "versions":
		path = apiPath("organizations", org, "regions", r.Args[0], "versions")
	case "delete":
		method, path = http.MethodDelete, apiPath("organizations", org, "regions", r.Args[0])
	case "create":
		if len(r.Args) > 0 {
			r.Flags = cloneFlags(r.Flags)
			if r.Flags["name"] != "" {
				return nil, errConflictingName()
			}
			r.Flags["name"] = r.Args[0]
		}
		method = http.MethodPost
		body, err = bodyFields(r, map[string]string{"name": "name"}, "name")
		if err != nil {
			return nil, err
		}
	}
	return c.Do(ctx, method, path, nil, body, nil)
}
