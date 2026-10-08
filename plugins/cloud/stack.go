package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

const stackDefaultVersion = "v4.0"

func executeStack(ctx context.Context, client *httpclient.Client, request pluginsdk.ExecuteRequest, path []string) (json.RawMessage, error) {
	path = stackRelativePath(path)
	request.CommandPath = append([]string{"stack"}, path...)
	request, err := pluginsdk.NormalizeRequest(pluginsdk.Manifest{ProtocolVersion: pluginsdk.ProtocolVersion, Root: stackManifest()}, request)
	if err != nil {
		return nil, err
	}
	organization, err := orgID(request)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("stack commands require a Membership client")
	}
	data, err := executeStackCommand(ctx, client, request, organization, strings.Join(path, "/"))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func stackRelativePath(path []string) []string {
	if len(path) > 0 && path[0] == "cloud" {
		path = path[1:]
	}
	if len(path) > 0 && path[0] == "stack" {
		path = path[1:]
	}
	return path
}

func executeStackCommand(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org, command string) (json.RawMessage, error) {
	if command != "history" && req.Flags["data"] != "" && req.Body == nil {
		return nil, fmt.Errorf("host must supply the already-read JSON request body")
	}
	switch command {
	case "create":
		return createStack(ctx, client, req, org)
	case "list":
		query := make(url.Values)
		for _, flag := range []string{"all", "deleted"} {
			if req.ChangedFlags[flag] {
				query.Set(flag, req.Flags[flag])
			}
		}
		return client.Do(ctx, http.MethodGet, stackCollectionPath(org), query, nil, nil)
	case "show":
		return showStack(ctx, client, req, org)
	case "update":
		return updateStack(ctx, client, req, org)
	case "upgrade":
		return upgradeStack(ctx, client, req, org)
	case "restore":
		return restoreStack(ctx, client, req, org)
	case "history":
		return stackHistory(ctx, client, req, org)
	case "users/link":
		return linkStackUser(ctx, client, req, org)
	default:
		return simpleStackCommand(ctx, client, req, org, command)
	}
}

func stackCollectionPath(org string) string { return httpclient.Path("organizations", org, "stacks") }

func selectedStack(req pluginsdk.ExecuteRequest, index int) (string, error) {
	id := req.Context["stack"]
	if len(req.Args) > index {
		id = req.Args[index]
	}
	if id == "" {
		var targets map[string][]string
		if raw := req.Context["stacks"]; raw != "" {
			if err := json.Unmarshal([]byte(raw), &targets); err != nil {
				return "", fmt.Errorf("invalid stacks context: %w", err)
			}
		}
		org, err := orgID(req)
		if err != nil {
			return "", err
		}
		ids := slices.Clone(targets[org])
		slices.Sort(ids)
		ids = slices.Compact(ids)
		if len(ids) != 1 {
			return "", fmt.Errorf("select a stack with its ID or --stack; available IDs: %s", strings.Join(ids, ", "))
		}
		id = ids[0]
	}
	return id, validID(id)
}

func stackResourcePath(org, id string, suffix ...string) string {
	return httpclient.Path(append([]string{"organizations", org, "stacks", id}, suffix...)...)
}

func simpleStackCommand(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org, command string) (json.RawMessage, error) {
	index := 0
	if slices.Contains([]string{"modules/enable", "modules/disable", "users/unlink"}, command) {
		index = 1
	}
	id, err := selectedStack(req, index)
	if err != nil {
		return nil, err
	}
	method, suffix, query := http.MethodGet, []string{}, make(url.Values)
	switch command {
	case "info", "version":
	case "delete":
		method = http.MethodDelete
		if req.ChangedFlags["force"] {
			query.Set("force", req.Flags["force"])
		}
	case "enable", "disable", "restore":
		method, suffix = http.MethodPut, []string{command}
	case "modules/list", "users/list":
		suffix = []string{strings.Split(command, "/")[0]}
	case "modules/enable", "modules/disable":
		method, suffix = http.MethodPost, []string{"modules"}
		if command == "modules/disable" {
			method = http.MethodDelete
		}
		if strings.TrimSpace(req.Args[0]) == "" {
			return nil, fmt.Errorf("module name must not be empty")
		}
		query.Set("name", req.Args[0])
	case "users/unlink":
		if strings.TrimSpace(req.Args[0]) == "" {
			return nil, fmt.Errorf("user ID must not be empty")
		}
		method, suffix = http.MethodDelete, []string{"users", req.Args[0]}
	default:
		return nil, fmt.Errorf("unknown stack command %q", command)
	}
	return client.Do(ctx, method, stackResourcePath(org, id, suffix...), query, nil, nil)
}

func stackObject(body json.RawMessage, allowed ...string) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	if body == nil {
		return fields, nil
	}
	if len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' {
		return nil, fmt.Errorf("stack request body must be a JSON object")
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	for key, value := range fields {
		if !slices.Contains(allowed, key) {
			return nil, fmt.Errorf("unknown stack request field %q", key)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("stack request field %s must not be null", key)
		}
	}
	return fields, nil
}

func stackSetString(fields map[string]json.RawMessage, key, value string) error {
	if existing, ok := fields[key]; ok {
		var current string
		if err := json.Unmarshal(existing, &current); err != nil {
			return fmt.Errorf("%s must be a string", key)
		}
		if current != value {
			return fmt.Errorf("conflicting %s values in body and arguments", key)
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fields[key] = encoded
	return nil
}

func stackString(fields map[string]json.RawMessage, key string) (string, error) {
	if fields[key] == nil {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(fields[key], &value); err != nil {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return value, nil
}

func stackNamedBody(req pluginsdk.ExecuteRequest, create bool) (map[string]json.RawMessage, error) {
	allowed := []string{"name", "metadata"}
	if create {
		allowed = append(allowed, "regionID", "version")
	}
	fields, err := stackObject(req.Body, allowed...)
	if err != nil {
		return nil, err
	}
	if create && len(req.Args) > 0 {
		if err := stackSetString(fields, "name", req.Args[0]); err != nil {
			return nil, err
		}
	}
	if req.ChangedFlags["name"] || req.Flags["name"] != "" {
		if err := stackSetString(fields, "name", req.Flags["name"]); err != nil {
			return nil, err
		}
	}
	if raw := fields["metadata"]; raw != nil {
		if err := stackValidateMetadata(raw); err != nil {
			return nil, fmt.Errorf("metadata must be an object of strings")
		}
	}
	return fields, nil
}

func stackValidateMetadata(raw json.RawMessage) error {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return err
	}
	for key, value := range metadata {
		if len(bytes.TrimSpace(value)) == 0 || bytes.TrimSpace(value)[0] != '"' {
			return fmt.Errorf("metadata value %s must be a string", key)
		}
	}
	return nil
}

func createStack(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	timeout, err := stackWaitTimeout(req)
	if err != nil {
		return nil, err
	}
	fields, err := stackNamedBody(req, true)
	if err != nil {
		return nil, err
	}
	name, err := stackString(fields, "name")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("stack name is required: use NAME, --name or --data")
	}
	region, err := stackCreateRegion(ctx, client, req, org, fields)
	if err != nil {
		return nil, err
	}
	version, err := stackRequestedVersion(req, fields, -1)
	if err != nil {
		return nil, err
	}
	if err := stackValidateVersion(ctx, client, org, region, version); err != nil {
		return nil, err
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	result, err := client.Do(ctx, http.MethodPost, stackCollectionPath(org), nil, body, nil)
	if err != nil {
		return nil, err
	}
	created, err := stackResponse(result)
	if err != nil {
		return nil, err
	}
	if created.ID == "" || (created.OrganizationID != "" && created.OrganizationID != org) {
		return nil, fmt.Errorf("created stack response has no stable ID or a different organization")
	}
	if req.Flags["no-wait"] == "true" {
		return result, nil
	}
	return waitStack(ctx, client, org, created.ID, version, timeout)
}

func stackCreateRegion(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string, fields map[string]json.RawMessage) (string, error) {
	if req.Flags["region"] != "" {
		if err := stackSetString(fields, "regionID", req.Flags["region"]); err != nil {
			return "", err
		}
	}
	region, err := stackString(fields, "regionID")
	if err != nil {
		return "", err
	}
	if region == "" {
		data, err := client.Do(ctx, http.MethodGet, httpclient.Path("organizations", org, "regions"), nil, nil, nil)
		if err != nil {
			return "", err
		}
		var catalog struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &catalog); err != nil {
			return "", err
		}
		if len(catalog.Data) != 1 || catalog.Data[0].ID == "" {
			return "", fmt.Errorf("select a region with --region or --data regionID")
		}
		region = catalog.Data[0].ID
	}
	if err := validID(region); err != nil {
		return "", fmt.Errorf("invalid region ID: %w", err)
	}
	if err := stackSetString(fields, "regionID", region); err != nil {
		return "", err
	}
	return region, nil
}

func stackRequestedVersion(req pluginsdk.ExecuteRequest, fields map[string]json.RawMessage, argIndex int) (string, error) {
	if argIndex >= 0 && len(req.Args) > argIndex {
		if err := stackSetString(fields, "version", req.Args[argIndex]); err != nil {
			return "", err
		}
	}
	if req.ChangedFlags["version"] {
		if err := stackSetString(fields, "version", req.Flags["version"]); err != nil {
			return "", err
		}
	}
	version, err := stackString(fields, "version")
	if err != nil {
		return "", err
	}
	if version == "" {
		version = stackDefaultVersion
	}
	if strings.TrimSpace(version) == "" {
		return "", fmt.Errorf("stack version must not be empty")
	}
	if err := stackSetString(fields, "version", version); err != nil {
		return "", err
	}
	return version, nil
}

func stackValidateVersion(ctx context.Context, client *httpclient.Client, org, region, version string) error {
	data, err := client.Do(ctx, http.MethodGet, httpclient.Path("organizations", org, "regions", region, "versions"), nil, nil, nil)
	if err != nil {
		return err
	}
	var catalog struct {
		Data []struct {
			Name       string `json:"name"`
			Deprecated bool   `json:"deprecated"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return err
	}
	for _, entry := range catalog.Data {
		if entry.Name == version && !entry.Deprecated {
			return nil
		}
	}
	return fmt.Errorf("region %s does not offer requested catalog version %s; no alternative version was selected", region, version)
}

type stackMetadata struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	RegionID       string `json:"regionID"`
	Status         string `json:"status"`
	State          string `json:"state"`
	Version        string `json:"version"`
}

func stackResponse(data json.RawMessage) (stackMetadata, error) {
	var envelope struct {
		Data *stackMetadata `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return stackMetadata{}, err
	}
	if envelope.Data == nil {
		return stackMetadata{}, fmt.Errorf("membership response is missing stack data")
	}
	return *envelope.Data, nil
}

func stackWaitTimeout(req pluginsdk.ExecuteRequest) (time.Duration, error) {
	value := req.Flags["wait-timeout"]
	if value == "" {
		value = "10m"
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return 0, fmt.Errorf("wait-timeout must be a positive duration")
	}
	return timeout, nil
}

func waitStack(ctx context.Context, client *httpclient.Client, org, id, version string, timeout time.Duration) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		result, err := client.Do(ctx, http.MethodGet, stackResourcePath(org, id), nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("waiting for stack %s (inspect with cloud stack show %s): %w", id, id, err)
		}
		ready, err := stackReadiness(result, org, id, version)
		if err != nil {
			return nil, err
		}
		if ready {
			return result, nil
		}
		if err := stackPollPause(ctx); err != nil {
			return nil, fmt.Errorf("waiting for stack %s (inspect with cloud stack show %s): %w", id, id, ctx.Err())
		}
	}
}

func stackReadiness(result json.RawMessage, org, id, version string) (bool, error) {
	state, err := stackResponse(result)
	if err != nil {
		return false, err
	}
	if state.ID != id || (state.OrganizationID != "" && state.OrganizationID != org) {
		return false, fmt.Errorf("membership readiness response targets a different stack")
	}
	if slices.Contains([]string{"DISABLED", "DELETED"}, state.Status) || slices.Contains([]string{"DISABLED", "DELETED"}, state.State) {
		return false, fmt.Errorf("stack %s entered %s/%s while waiting", id, state.State, state.Status)
	}
	return state.Status == "READY" && (version == "" || state.Version == version), nil
}

func stackPollPause(ctx context.Context) error {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func showStack(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	name := req.Flags["name"]
	if name == "" {
		id, err := selectedStack(req, 0)
		if err != nil {
			return nil, err
		}
		return client.Do(ctx, http.MethodGet, stackResourcePath(org, id), nil, nil, nil)
	}
	if len(req.Args) != 0 {
		return nil, fmt.Errorf("use either a stack ID or --name")
	}
	data, err := client.Do(ctx, http.MethodGet, stackCollectionPath(org), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	var match json.RawMessage
	for _, raw := range envelope.Data {
		var stack struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &stack); err != nil {
			return nil, err
		}
		if stack.Name == name {
			if match != nil {
				return nil, fmt.Errorf("stack name %q is ambiguous; use its ID", name)
			}
			match = raw
		}
	}
	if match == nil {
		return nil, fmt.Errorf("stack named %q not found", name)
	}
	return json.Marshal(struct {
		Data json.RawMessage `json:"data"`
	}{match})
}

func updateStack(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	id, err := selectedStack(req, 0)
	if err != nil {
		return nil, err
	}
	changes, err := stackNamedBody(req, false)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("provide --name or --data to update the stack")
	}
	current, err := client.Do(ctx, http.MethodGet, stackResourcePath(org, id), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(current, &envelope); err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "metadata"} {
		if raw := envelope.Data[key]; raw != nil && string(raw) != "null" {
			fields[key] = raw
		}
	}
	maps.Copy(fields, changes)
	name, err := stackString(fields, "name")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("stack name must not be empty")
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, http.MethodPut, stackResourcePath(org, id), nil, body, nil)
}

func upgradeStack(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	timeout, err := stackWaitTimeout(req)
	if err != nil {
		return nil, err
	}
	id, err := selectedStack(req, 0)
	if err != nil {
		return nil, err
	}
	fields, err := stackObject(req.Body, "version")
	if err != nil {
		return nil, err
	}
	version, err := stackRequestedVersion(req, fields, 1)
	if err != nil {
		return nil, err
	}
	current, err := client.Do(ctx, http.MethodGet, stackResourcePath(org, id), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	state, err := stackResponse(current)
	if err != nil {
		return nil, err
	}
	if state.ID != id || (state.OrganizationID != "" && state.OrganizationID != org) {
		return nil, fmt.Errorf("membership upgrade response targets a different stack")
	}
	if state.RegionID == "" {
		return nil, fmt.Errorf("membership stack response is missing regionID")
	}
	if state.Version == version {
		return waitExistingStackVersion(ctx, client, req, org, id, version, current, timeout)
	}
	if err := stackValidateVersion(ctx, client, org, state.RegionID, version); err != nil {
		return nil, err
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	result, err := client.Do(ctx, http.MethodPut, stackResourcePath(org, id, "upgrade"), nil, body, nil)
	if err != nil || req.Flags["no-wait"] == "true" {
		return result, err
	}
	return waitStack(ctx, client, org, id, version, timeout)
}

func waitExistingStackVersion(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org, id, version string, current json.RawMessage, timeout time.Duration) (json.RawMessage, error) {
	if req.Flags["no-wait"] == "true" {
		return current, nil
	}
	ready, err := stackReadiness(current, org, id, version)
	if err != nil {
		return nil, err
	}
	if ready {
		return current, nil
	}
	return waitStack(ctx, client, org, id, version, timeout)
}

func restoreStack(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	timeout, err := stackWaitTimeout(req)
	if err != nil {
		return nil, err
	}
	id, err := selectedStack(req, 0)
	if err != nil {
		return nil, err
	}
	result, err := client.Do(ctx, http.MethodPut, stackResourcePath(org, id, "restore"), nil, nil, nil)
	if err != nil {
		return nil, err
	}
	state, err := stackResponse(result)
	if err != nil {
		return nil, err
	}
	if state.ID != id || (state.OrganizationID != "" && state.OrganizationID != org) {
		return nil, fmt.Errorf("membership restore response targets a different stack")
	}
	if req.Flags["no-wait"] == "true" {
		return result, nil
	}
	return waitStack(ctx, client, org, id, state.Version, timeout)
}

func linkStackUser(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	id, err := selectedStack(req, 1)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Args[0]) == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}
	fields, err := stackObject(req.Body, "policyId")
	if err != nil {
		return nil, err
	}
	if flag := req.Flags["policy-id"]; flag != "" {
		value, err := strconv.ParseInt(flag, 10, 64)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("policy-id must be a positive 64-bit integer")
		}
		encoded := json.RawMessage(strconv.FormatInt(value, 10))
		if existing := fields["policyId"]; existing != nil && !bytes.Equal(bytes.TrimSpace(existing), encoded) {
			return nil, fmt.Errorf("conflicting policyId in body and --policy-id")
		}
		fields["policyId"] = encoded
	}
	var policy int64
	if err := json.Unmarshal(fields["policyId"], &policy); err != nil || policy <= 0 {
		return nil, fmt.Errorf("provide a positive policyId with --policy-id or --data")
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return client.Do(ctx, http.MethodPut, stackResourcePath(org, id, "users", req.Args[0]), nil, body, nil)
}

func stackHistory(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, org string) (json.RawMessage, error) {
	id, err := selectedStack(req, 0)
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseUint(req.Flags["page-size"], 10, 32)
	if err != nil || size < 1 || size > 1000 {
		return nil, fmt.Errorf("page-size must be between 1 and 1000")
	}
	query := url.Values{"stackId": {id}, "pageSize": {strconv.FormatUint(size, 10)}}
	if cursor := req.Flags["cursor"]; cursor != "" {
		if req.Flags["action"] != "" || req.Flags["user-id"] != "" || req.Flags["data"] != "" {
			return nil, fmt.Errorf("cursor cannot be combined with log filters")
		}
		query.Set("cursor", cursor)
	}
	if action := req.Flags["action"]; action != "" {
		if !strings.HasPrefix(action, "stacks.") {
			return nil, fmt.Errorf("stack history actions must use the stacks prefix")
		}
		query.Set("action", action)
	}
	if user := req.Flags["user-id"]; user != "" {
		query.Set("userId", user)
	}
	if filter := req.Flags["data"]; filter != "" {
		key, value, found := strings.Cut(filter, "=")
		if !found || key == "" {
			return nil, fmt.Errorf("data filter must be key=value")
		}
		query.Set("key", key)
		query.Set("value", value)
	}
	return client.Do(ctx, http.MethodGet, httpclient.Path("organizations", org, "logs"), query, nil, nil)
}
