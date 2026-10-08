package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

func organizationsManifest() pluginsdk.CommandSpec {
	name := stringFlag("name", "Organization or resource name")
	history := withFlags(optionalArg(leaf("history [ORGANIZATION]", "List organization audit logs", 1, false)), paginationFlags()...)
	history = withFlags(history, stringFlag("stack-id", "Filter by stack ID"), stringFlag("user-id", "Filter by user ID"), stringFlag("action", "Filter by action"), stringFlag("key", "Filter metadata key"), stringFlag("value", "Filter metadata value"))
	return pluginsdk.CommandSpec{Use: "organizations", Short: "Manage organizations and access", Subcommands: []pluginsdk.CommandSpec{
		withFlags(leaf("list", "List organizations", 0, false), boolFlag("expand", "Include expanded organization information")),
		withFlags(optionalArg(leaf("describe [ORGANIZATION]", "Show an organization", 1, false)), boolFlag("expand", "Include expanded organization information")),
		withFlags(optionalArg(leafBody("create [NAME]", "Create an organization", 1, false)), name, stringFlag("domain", "Organization domain"), stringFlag("default-policy-id", "Default policy ID"), stringFlag("owner-id", "Owner user ID")),
		withFlags(optionalArg(leafBody("update [ORGANIZATION]", "Update an organization, preserving omitted fields", 1, true)), name, stringFlag("domain", "Organization domain"), stringFlag("default-policy-id", "Default policy ID")),
		optionalArg(leaf("delete [ORGANIZATION]", "Delete an organization", 1, true)), history,
		{Use: "users", Short: "Manage organization users", Subcommands: []pluginsdk.CommandSpec{
			leaf("list", "List organization users", 0, false), leaf("show USER", "Show an organization user", 1, false), withFlags(leafBody("link USER", "Link a user with a policy", 1, true), stringFlag("policy-id", "Policy ID")), leaf("unlink USER", "Unlink a user", 1, true),
		}},
		{Use: "invitations", Short: "Manage organization invitations", Subcommands: []pluginsdk.CommandSpec{
			withFlags(leaf("list", "List organization invitations", 0, false), stringFlag("status", "Invitation status")), withFlags(optionalArg(leaf("send [EMAIL]", "Send an organization invitation", 1, false)), stringFlag("email", "Invitee email")), leaf("delete INVITATION", "Delete an invitation", 1, true),
		}},
		{Use: "policies", Short: "Manage organization policies", Subcommands: []pluginsdk.CommandSpec{
			leaf("list", "List policies", 0, false), leaf("show POLICY", "Show a policy", 1, false),
			withFlags(leafBody("create", "Create a policy", 0, false), name, stringFlag("description", "Policy description")),
			withFlags(leafBody("update POLICY", "Update a policy, preserving omitted fields", 1, true), name, stringFlag("description", "Policy description")),
			leaf("delete POLICY", "Delete a policy", 1, true), leaf("add-scope POLICY SCOPE", "Add a scope to a policy", 2, true), leaf("remove-scope POLICY SCOPE", "Remove a scope from a policy", 2, true),
		}},
		{Use: "oauth-clients", Short: "Manage organization OAuth clients", Subcommands: []pluginsdk.CommandSpec{
			leaf("list", "List OAuth clients", 0, false), leaf("show CLIENT", "Show an OAuth client", 1, false),
			withFlags(leafBody("create", "Create an OAuth client; save the returned secret", 0, false), name, stringFlag("description", "Client description")),
			withFlags(leafBody("update CLIENT", "Update an OAuth client, preserving omitted fields", 1, true), name, stringFlag("description", "Client description")),
			leaf("delete CLIENT", "Delete an OAuth client", 1, true),
		}},
		{Use: "authentication-provider", Short: "Configure organization authentication", Subcommands: []pluginsdk.CommandSpec{
			leaf("show", "Show the authentication provider", 0, false), leaf("delete", "Delete the authentication provider", 0, true),
			providerConfigureManifest(),
		}},
		{Use: "applications", Short: "Manage applications enabled for an organization", Subcommands: []pluginsdk.CommandSpec{
			withFlags(leaf("list", "List organization applications", 0, false), boolFlag("expand", "Include application details")), leaf("show APPLICATION", "Show an organization application", 1, false), leaf("enable APPLICATION", "Enable an application", 1, false), leaf("disable APPLICATION", "Disable an application", 1, true),
		}},
	}}
}
func providerConfigureManifest() pluginsdk.CommandSpec {
	s := leafBody("configure [TYPE NAME CLIENT_ID CLIENT_SECRET]", "Configure an authentication provider", 4, true)
	s.Args.Min = 0
	return withFlags(s, stringFlag("type", "github, google, microsoft or oidc"), stringFlag("name", "Provider name"), stringFlag("provider-client-id", "OAuth client ID"), stringFlag("provider-client-secret", "OAuth client secret"), stringFlag("oidc-issuer", "OIDC issuer URL"), stringFlag("microsoft-tenant", "Microsoft tenant ID"))
}
func executeOrganizations(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	action := r.CommandPath[2]
	if action == "list" {
		q, err := queryFlags(r, map[string]string{"expand": "expand"})
		if err != nil {
			return nil, err
		}
		return c.Do(ctx, http.MethodGet, "/organizations", q, nil, nil)
	}
	if action == "create" {
		return createOrganization(ctx, c, r)
	}
	if len(r.CommandPath) == 3 && len(r.Args) > 0 {
		r.Context = maps.Clone(r.Context)
		if r.Context == nil {
			r.Context = make(map[string]string)
		}
		r.Context["organization"] = r.Args[0]
	}
	org, err := requireOrg(ctx, c, r)
	if err != nil {
		return nil, err
	}
	path := apiPath("organizations", org)
	switch action {
	case "describe":
		q, err := queryFlags(r, map[string]string{"expand": "expand"})
		if err != nil {
			return nil, err
		}
		return c.Do(ctx, http.MethodGet, path, q, nil, nil)
	case "delete":
		return c.Do(ctx, http.MethodDelete, path, nil, nil, nil)
	case "update":
		return updateOrganization(ctx, c, r, path)
	case "history":
		q, err := queryFlags(r, map[string]string{"cursor": "cursor", "page-size": "pageSize", "stack-id": "stackId", "user-id": "userId", "action": "action", "key": "key", "value": "value"})
		if err != nil {
			return nil, err
		}
		return c.Do(ctx, http.MethodGet, path+"/logs", q, nil, nil)
	case "users":
		return executeOrgUsers(ctx, c, r, org)
	case "invitations":
		return executeOrgInvitations(ctx, c, r, org)
	case "policies":
		return executeOrgPolicies(ctx, c, r, org)
	case "oauth-clients":
		return executeOrgClients(ctx, c, r, org)
	case "authentication-provider":
		return executeProvider(ctx, c, r, org)
	case "applications":
		return executeOrgApplications(ctx, c, r, org)
	}
	return nil, fmt.Errorf("unsupported organization command")
}

const organizationPolicyGuidance = "organization update requires a positive default policy ID: provide --default-policy-id or defaultPolicyID in --data; Membership cannot preserve or clear an unset policy"

func validateOrganizationDefaultPolicy(raw json.RawMessage) error {
	var id int64
	if json.Unmarshal(raw, &id) != nil || id <= 0 {
		return fmt.Errorf("%s", organizationPolicyGuidance)
	}
	return nil
}

func organizationUpdatePatch(r pluginsdk.ExecuteRequest) (map[string]json.RawMessage, error) {
	patch, err := bodyFields(r, map[string]string{"name": "name", "domain": "domain", "default-policy-id": "defaultPolicyID"})
	if err != nil {
		if strings.Contains(err.Error(), "defaultPolicyID") || strings.Contains(err.Error(), "--default-policy-id") {
			return nil, fmt.Errorf("%w; %s", err, organizationPolicyGuidance)
		}
		return nil, err
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("provide fields to update using --data or flags")
	}
	if policy, explicit := changes["defaultPolicyID"]; explicit {
		if err := validateOrganizationDefaultPolicy(policy); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

func updateOrganization(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, path string) (json.RawMessage, error) {
	changes, err := organizationUpdatePatch(r)
	if err != nil {
		return nil, err
	}
	current, err := c.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(current, &envelope) != nil || envelope.Data == nil {
		return nil, fmt.Errorf("response is missing resource data")
	}
	values := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "domain", "defaultPolicyID"} {
		if value, ok := envelope.Data[key]; ok {
			values[key] = value
		}
	}
	maps.Copy(values, changes)
	if err := validateOrganizationDefaultPolicy(values["defaultPolicyID"]); err != nil {
		return nil, err
	}
	if value := values["name"]; len(value) == 0 || string(value) == `""` || string(value) == "null" {
		return nil, fmt.Errorf("name is required")
	}
	body, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, http.MethodPut, path, nil, body, nil)
}
func executeOrgUsers(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	method, path := http.MethodGet, apiPath("organizations", org, "users")
	var body json.RawMessage
	if len(r.Args) > 0 {
		path = apiPath("organizations", org, "users", r.Args[0])
	}
	switch r.CommandPath[3] {
	case "unlink":
		method = http.MethodDelete
	case "link":
		var err error
		body, err = bodyFields(r, map[string]string{"policy-id": "policyId"}, "policyId")
		if err != nil {
			return nil, err
		}
		method = http.MethodPut
	}
	return c.Do(ctx, method, path, nil, body, nil)
}
func executeOrgInvitations(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	path := apiPath("organizations", org, "invitations")
	switch r.CommandPath[3] {
	case "list":
		q, err := queryFlags(r, map[string]string{"status": "status"})
		if err != nil {
			return nil, err
		}
		return c.Do(ctx, http.MethodGet, path, q, nil, nil)
	case "delete":
		return c.Do(ctx, http.MethodDelete, apiPath("organizations", org, "invitations", r.Args[0]), nil, nil, nil)
	case "send":
		email := r.Flags["email"]
		if len(r.Args) > 0 {
			if email != "" && email != r.Args[0] {
				return nil, fmt.Errorf("email argument conflicts with --email")
			}
			email = r.Args[0]
		}
		if strings.TrimSpace(email) == "" {
			return nil, fmt.Errorf("email is required")
		}
		return c.Do(ctx, http.MethodPost, path, url.Values{"email": {email}}, nil, nil)
	}
	return nil, fmt.Errorf("unsupported invitation command")
}
func executeOrgPolicies(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	for _, id := range r.Args {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("policy and scope IDs must be positive integers")
		}
	}
	path := apiPath("organizations", org, "policies")
	method := http.MethodGet
	var body json.RawMessage
	if len(r.Args) > 0 {
		path = apiPath("organizations", org, "policies", r.Args[0])
	}
	switch r.CommandPath[3] {
	case "create":
		var err error
		body, err = bodyFields(r, map[string]string{"name": "name", "description": "description"}, "name")
		if err != nil {
			return nil, err
		}
		method = http.MethodPost
	case "update":
		return updateResource(ctx, c, r, path, map[string]string{"name": "name", "description": "description"}, []string{"name", "description"}, "name")
	case "delete":
		method = http.MethodDelete
	case "add-scope":
		method = http.MethodPut
		path = apiPath("organizations", org, "policies", r.Args[0], "scopes", r.Args[1])
	case "remove-scope":
		method = http.MethodDelete
		path = apiPath("organizations", org, "policies", r.Args[0], "scopes", r.Args[1])
	}
	return c.Do(ctx, method, path, nil, body, nil)
}
func executeOrgClients(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	path := apiPath("organizations", org, "clients")
	method := http.MethodGet
	var body json.RawMessage
	if len(r.Args) > 0 {
		path = apiPath("organizations", org, "clients", r.Args[0])
	}
	switch r.CommandPath[3] {
	case "create":
		var err error
		body, err = bodyFields(r, map[string]string{"name": "name", "description": "description"})
		if err != nil {
			return nil, err
		}
		method = http.MethodPost
	case "update":
		return updateResource(ctx, c, r, path, map[string]string{"name": "name", "description": "description"}, []string{"name", "description"}, "name")
	case "delete":
		method = http.MethodDelete
	}
	return c.Do(ctx, method, path, nil, body, nil)
}
func executeProvider(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	path := apiPath("organizations", org, "authentication-provider")
	switch r.CommandPath[3] {
	case "show":
		return c.Do(ctx, http.MethodGet, path, nil, nil, nil)
	case "delete":
		return c.Do(ctx, http.MethodDelete, path, nil, nil, nil)
	case "configure":
		body, err := providerBody(r)
		if err != nil {
			return nil, err
		}
		return c.Do(ctx, http.MethodPut, path, nil, body, nil)
	}
	return nil, fmt.Errorf("unsupported authentication provider command")
}
func providerBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	r, err := providerArguments(r)
	if err != nil {
		return nil, err
	}
	body, err := bodyFields(r, map[string]string{"type": "type", "name": "name", "provider-client-id": "clientID", "provider-client-secret": "clientSecret"}, "type", "name", "clientID", "clientSecret")
	if err != nil {
		return nil, err
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(body, &values); err != nil {
		return nil, err
	}
	var kind string
	if err := json.Unmarshal(values["type"], &kind); err != nil {
		return nil, fmt.Errorf("provider type must be a string")
	}
	config, err := providerConfigFlags(r, kind)
	if err != nil {
		return nil, err
	}
	if _, exists := values["config"]; exists {
		if len(config) > 0 {
			return nil, fmt.Errorf("provide config using either --data or provider flags")
		}
	} else {
		raw, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		values["config"] = raw
	}
	if kind == "oidc" {
		if err := validateOIDCConfig(values["config"]); err != nil {
			return nil, err
		}
	}
	return json.Marshal(values)
}
func providerArguments(r pluginsdk.ExecuteRequest) (pluginsdk.ExecuteRequest, error) {
	if len(r.Args) == 0 {
		return r, nil
	}
	if len(r.Args) > 4 {
		return r, fmt.Errorf("configure accepts at most TYPE NAME CLIENT_ID CLIENT_SECRET; supply remaining values with flags or JSON")
	}
	r.Flags = cloneFlags(r.Flags)
	names := []string{"type", "name", "provider-client-id", "provider-client-secret"}
	for i, argument := range r.Args {
		name := names[i]
		if r.ChangedFlags[name] || r.Flags[name] != "" {
			return r, fmt.Errorf("argument conflicts with --%s", name)
		}
		r.Flags[name] = argument
	}
	return r, nil
}
func providerConfigFlags(r pluginsdk.ExecuteRequest, kind string) (map[string]string, error) {
	config := map[string]string{}
	switch kind {
	case "google", "github":
	case "microsoft":
		if tenant := r.Flags["microsoft-tenant"]; tenant != "" {
			config["tenant"] = tenant
		}
	case "oidc":
		if issuer := r.Flags["oidc-issuer"]; issuer != "" {
			if err := httpclient.ValidateURL(issuer); err != nil {
				return nil, fmt.Errorf("invalid OIDC issuer: %w", err)
			}
			config["issuer"] = issuer
		}
	default:
		return nil, fmt.Errorf("unsupported provider type %q", kind)
	}
	return config, nil
}
func validateOIDCConfig(raw json.RawMessage) error {
	var config struct {
		Issuer string `json:"issuer"`
	}
	if json.Unmarshal(raw, &config) != nil || config.Issuer == "" {
		return fmt.Errorf("OIDC config requires issuer")
	}
	return httpclient.ValidateURL(config.Issuer)
}

// PUT replaces these resources. Read and merge only writable API fields.
func updateResource(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, path string, flags map[string]string, writable []string, required ...string) (json.RawMessage, error) {
	patch, err := bodyFields(r, flags)
	if err != nil {
		return nil, err
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("provide fields to update using --data or flags")
	}
	current, err := c.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(current, &envelope) != nil || envelope.Data == nil {
		return nil, fmt.Errorf("response is missing resource data")
	}
	values := make(map[string]json.RawMessage)
	for _, key := range writable {
		if value, ok := envelope.Data[key]; ok {
			values[key] = value
		}
	}
	maps.Copy(values, changes)
	for _, key := range required {
		if value, ok := values[key]; !ok || string(value) == `""` || (string(value) == "null" && key != "defaultPolicyID") {
			return nil, fmt.Errorf("%s is required", key)
		}
	}
	body, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, http.MethodPut, path, nil, body, nil)
}
func cloneFlags(flags map[string]string) map[string]string {
	cloned := maps.Clone(flags)
	if cloned == nil {
		cloned = make(map[string]string)
	}
	return cloned
}
func errConflictingName() error { return fmt.Errorf("name argument conflicts with --name") }

func createOrganization(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if len(r.Args) > 0 {
		r.Flags = cloneFlags(r.Flags)
		if r.Flags["name"] != "" {
			return nil, errConflictingName()
		}
		r.Flags["name"] = r.Args[0]
	}
	body, err := bodyFields(r, map[string]string{"name": "name", "domain": "domain", "default-policy-id": "defaultPolicyID", "owner-id": "ownerID"}, "name")
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, http.MethodPost, "/organizations", nil, body, nil)
}
func executeOrgApplications(ctx context.Context, c *httpclient.Client, r pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	path := apiPath("organizations", org, "applications")
	method := http.MethodGet
	switch r.CommandPath[3] {
	case "show":
		path = apiPath("organizations", org, "applications", r.Args[0])
	case "enable":
		method = http.MethodPut
		path = apiPath("organizations", org, "applications", r.Args[0])
	case "disable":
		method = http.MethodDelete
		path = apiPath("organizations", org, "applications", r.Args[0])
	}
	q := url.Values{}
	if r.CommandPath[3] == "list" {
		var err error
		q, err = queryFlags(r, map[string]string{"expand": "expand"})
		if err != nil {
			return nil, err
		}
	}
	return c.Do(ctx, method, path, q, nil, nil)
}
