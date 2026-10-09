// Package cloud implements Cloud commands using only the public plugin SDK.
package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type plugin struct{ client *http.Client }

// New creates a Cloud plugin with the host's authenticated Membership client.
func New(client *http.Client) pluginsdk.Plugin { return &plugin{client: client} }
func (*plugin) GetManifest(ctx context.Context) (pluginsdk.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.Manifest{}, err
	}
	return manifest(), nil
}
func (p *plugin) Execute(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	if err := ctx.Err(); err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	req, err := pluginsdk.NormalizeRequest(manifest(), req)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if p.client == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("cloud plugin requires an HTTP client")
	}
	client, err := httpclient.New(req.Endpoint, p.client)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	for _, arg := range req.Args {
		if err := validID(arg); err != nil {
			return pluginsdk.ExecuteResponse{}, err
		}
	}
	var data json.RawMessage
	switch req.CommandPath[1] {
	case "me":
		data, err = executeMe(ctx, client, req)
	case "organizations":
		data, err = executeOrganizations(ctx, client, req)
	case "regions":
		data, err = executeRegions(ctx, client, req)
	case "apps":
		data, err = executeApps(ctx, client, req, req.CommandPath)
	case "stack":
		var org string
		org, err = requireOrg(ctx, client, req)
		if err == nil {
			req.Context = maps.Clone(req.Context)
			if req.Context == nil {
				req.Context = make(map[string]string)
			}
			req.Context["organization"] = org
			data, err = executeStack(ctx, client, req, req.CommandPath)
		}
	default:
		err = fmt.Errorf("unsupported cloud command %q", req.CommandPath)
	}
	return pluginsdk.ExecuteResponse{Data: data}, err
}

func validID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." {
		return fmt.Errorf("identifier must be non-empty and cannot be a dot or double dot")
	}
	return nil
}

// orgID selects a target without changing the request or its host context.
func orgID(req pluginsdk.ExecuteRequest) (string, error) {
	id := req.Context["organization"]
	if id != "" {
		return id, validID(id)
	}
	var ids []string
	if raw := req.Context["organizations"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &ids); err != nil {
			return "", fmt.Errorf("invalid organizations context: %w", err)
		}
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) == 1 {
		return ids[0], validID(ids[0])
	}
	return "", fmt.Errorf("choose a Cloud organization with --organization ORGANIZATION_ID\nList organizations: fctl cloud organizations list")
}
func requireOrg(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest) (string, error) {
	if req.Context["organization"] != "" || req.Context["organizations"] != "" {
		return orgID(req)
	}
	data, err := client.Do(ctx, http.MethodGet, "/organizations", nil, nil, nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("list organizations: %w", err)
	}
	ids := make([]string, 0, len(result.Data))
	for _, org := range result.Data {
		ids = append(ids, org.ID)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) == 1 {
		return ids[0], validID(ids[0])
	}
	return "", fmt.Errorf("choose a Cloud organization with --organization ORGANIZATION_ID\nList organizations: fctl cloud organizations list")
}
func queryFlags(req pluginsdk.ExecuteRequest, names map[string]string) (url.Values, error) {
	q := url.Values{}
	for flag, param := range names {
		if v := req.Flags[flag]; v != "" {
			if flag == "page-size" && v == "0" {
				return nil, fmt.Errorf("--page-size must be positive")
			}
			q.Set(param, v)
		}
	}
	return q, nil
}

// bodyFields accepts host-decoded inline/@file JSON, with optional convenience flags.
// Explicit body fields and explicit flags cannot overwrite each other silently.
func bodyFields(req pluginsdk.ExecuteRequest, fields map[string]string, required ...string) (json.RawMessage, error) {
	values, err := cloudBodyObject(req)
	if err != nil {
		return nil, err
	}
	for flag, key := range fields {
		if err := applyBodyFlag(req, values, flag, key); err != nil {
			return nil, err
		}
	}
	for _, key := range required {
		v, ok := values[key]
		if !ok || string(v) == "null" || string(v) == `""` {
			return nil, fmt.Errorf("%s is required in --data or its convenience flag", key)
		}
	}
	for key, raw := range values {
		if err := validateBodyField(key, raw, slices.Contains(required, key)); err != nil {
			return nil, err
		}
	}
	return json.Marshal(values)
}
func cloudBodyObject(req pluginsdk.ExecuteRequest) (map[string]json.RawMessage, error) {
	if req.Flags["data"] != "" && req.Body == nil {
		return nil, fmt.Errorf("--data must be decoded into Body by the host")
	}
	values := make(map[string]json.RawMessage)
	if req.Body != nil {
		if err := json.Unmarshal(req.Body, &values); err != nil || values == nil {
			return nil, fmt.Errorf("--data must be a JSON object")
		}
	}
	return values, nil
}
func applyBodyFlag(req pluginsdk.ExecuteRequest, values map[string]json.RawMessage, flag, key string) error {
	value, ok := req.Flags[flag]
	if !ok || (!req.ChangedFlags[flag] && value == "") {
		return nil
	}
	if _, exists := values[key]; exists {
		return fmt.Errorf("provide %s using either --data or --%s", key, flag)
	}
	raw, err := encodeBodyFlag(flag, value)
	if err != nil {
		return err
	}
	values[key] = raw
	return nil
}
func encodeBodyFlag(flag, value string) (json.RawMessage, error) {
	if !strings.HasSuffix(flag, "policy-id") {
		return json.Marshal(value)
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("--%s must be a positive integer", flag)
	}
	return json.RawMessage(strconv.FormatInt(n, 10)), nil
}
func validateBodyField(key string, raw json.RawMessage, required bool) error {
	switch key {
	case "name", "domain", "description", "ownerID", "type", "clientID", "clientSecret":
		var value string
		if string(raw) == "null" || json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("%s must be a string", key)
		}
		if required && strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", key)
		}
	case "defaultPolicyID", "policyId":
		if key == "defaultPolicyID" && string(raw) == "null" {
			return nil
		}
		var value int64
		if json.Unmarshal(raw, &value) != nil || value <= 0 {
			return fmt.Errorf("%s must be a positive integer", key)
		}
	}
	return nil
}

func apiPath(segments ...string) string {
	return httpclient.Path(segments...)
}
