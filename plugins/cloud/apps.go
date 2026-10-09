package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

const appsFileLimit = 4 << 20

func executeApps(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, path []string) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// This guard precedes normalization, file access and every service request.
	if req.Flags["experimental"] != "true" {
		return nil, fmt.Errorf("cloud apps requires --experimental")
	}
	if len(path) > 0 && path[0] == "cloud" {
		path = path[1:]
	}
	if len(path) > 0 && path[0] == "apps" {
		path = path[1:]
	}
	req.CommandPath = append([]string{"apps"}, path...)
	req, err := pluginsdk.NormalizeRequest(pluginsdk.Manifest{ProtocolVersion: pluginsdk.ProtocolVersion, Root: appsManifest()}, req)
	if err != nil {
		return nil, err
	}
	if client == nil || client.HTTPClient() == nil {
		return nil, fmt.Errorf("apps requires an injected Deploy client")
	}
	if err := appsValidateRequest(req); err != nil {
		return nil, err
	}
	cmd := strings.Join(path, "/")
	timeout, err := appsTimeout(req)
	if err != nil {
		return nil, err
	}
	if cmd == "manifests/create" || cmd == "manifests/versions/push" {
		return appsUpload(ctx, client, req, cmd)
	}
	method, segments, q, body, err := appsRoute(req, cmd)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(cmd, "download") {
		return appsDownload(ctx, client, req, httpclient.Path(segments...), q)
	}
	result, err := client.Do(ctx, method, httpclient.Path(segments...), q, body, nil)
	return appsResult(ctx, client, req, cmd, body, result, err, timeout)
}

func appsValidateRequest(req pluginsdk.ExecuteRequest) error {
	for _, key := range []string{"id", "app-id", "manifest-id"} {
		if v := req.Flags[key]; v != "" && (strings.TrimSpace(v) == "" || v == "." || v == "..") {
			return fmt.Errorf("invalid --%s", key)
		}
	}
	if req.Flags["data"] != "" && req.Body == nil {
		return fmt.Errorf("host must decode --data into Body")
	}
	command, err := pluginsdk.FindCommand(pluginsdk.Manifest{Root: appsManifest()}, req.CommandPath)
	if err != nil {
		return err
	}
	acceptsBody := slices.ContainsFunc(command.Flags, func(f pluginsdk.FlagSpec) bool { return f.Body })
	if req.Body != nil && !acceptsBody {
		return fmt.Errorf("this apps command does not accept a body")
	}
	return nil
}

func appsTimeout(req pluginsdk.ExecuteRequest) (time.Duration, error) {
	if req.Flags["wait-timeout"] == "" {
		return 30 * time.Minute, nil
	}
	timeout, err := time.ParseDuration(req.Flags["wait-timeout"])
	if err != nil || timeout <= 0 || timeout > 24*time.Hour {
		return 0, fmt.Errorf("--wait-timeout must be positive and at most 24h")
	}
	return timeout, nil
}

type appsOperation struct {
	method   string
	segments []string
	fields   map[string]string
	required []string
}

func appsRoute(req pluginsdk.ExecuteRequest, cmd string) (string, []string, url.Values, json.RawMessage, error) {
	operations := map[string]appsOperation{
		"list":                    {method: "GET", segments: []string{"apps"}},
		"show":                    {method: "GET", segments: []string{"apps", ":id"}},
		"create":                  {method: "POST", segments: []string{"apps"}, fields: map[string]string{"name": "name", "stack-id": "stackId"}, required: []string{"name"}},
		"delete":                  {method: "DELETE", segments: []string{"apps", ":id"}},
		"bind-manifest":           {method: "PUT", segments: []string{"apps", ":app-id", "manifest"}, fields: map[string]string{"manifest-id": "manifestId"}, required: []string{"manifestId"}},
		"unbind-manifest":         {method: "DELETE", segments: []string{"apps", ":app-id", "manifest"}},
		"manifests/list":          {method: "GET", segments: []string{"manifests"}},
		"manifests/show":          {method: "GET", segments: []string{"manifests", ":id"}},
		"manifests/update":        {method: "PATCH", segments: []string{"manifests", ":id"}, fields: map[string]string{"name": "name"}, required: []string{"name"}},
		"manifests/delete":        {method: "DELETE", segments: []string{"manifests", ":id"}},
		"manifests/download":      {method: "GET", segments: []string{"manifests", ":id", "versions", ":version"}},
		"manifests/versions/list": {method: "GET", segments: []string{"manifests", ":manifest-id", "versions"}},
		"manifests/versions/show": {method: "GET", segments: []string{"manifests", ":manifest-id", "versions", ":version"}},
		"deployments/list":        {method: "GET", segments: []string{"deployments"}},
		"deployments/show":        {method: "GET", segments: []string{"deployments", ":id"}},
		"deployments/download":    {method: "GET", segments: []string{"deployments", ":id"}},
		"deployments/logs":        {method: "GET", segments: []string{"deployments", ":id", "logs"}},
		"deployments/create":      {method: "POST", segments: []string{"deployments"}, fields: map[string]string{"app-id": "appId", "manifest-id": "manifestId", "manifest-version": "manifestVersion"}, required: []string{"appId", "manifestId", "manifestVersion"}},
		"variables/list":          {method: "GET", segments: []string{"apps", ":id", "variables"}},
		"variables/create":        {method: "POST", segments: []string{"apps", ":id", "variables"}},
		"variables/delete":        {method: "DELETE", segments: []string{"apps", ":app-id", "variables", ":id"}},
	}
	op, ok := operations[cmd]
	if !ok {
		return "", nil, nil, nil, fmt.Errorf("unsupported apps command")
	}
	for i, segment := range op.segments {
		if flag, ok := strings.CutPrefix(segment, ":"); ok {
			op.segments[i] = req.Flags[flag]
		}
	}
	q, err := appsQuery(req, cmd)
	if err != nil {
		return "", nil, nil, nil, err
	}
	var body json.RawMessage
	if op.fields != nil {
		body, err = appsBody(req, op.fields, op.required...)
	}
	if cmd == "variables/create" {
		body, err = appsVariableBody(req)
	}
	return op.method, op.segments, q, body, err
}

func appsQuery(req pluginsdk.ExecuteRequest, cmd string) (url.Values, error) {
	q := url.Values{}
	if strings.HasSuffix(cmd, "list") {
		n, err := strconv.ParseUint(req.Flags["page-size"], 10, 32)
		if err != nil || n == 0 {
			return nil, fmt.Errorf("--page-size must be positive")
		}
		q.Set("pageSize", req.Flags["page-size"])
		if req.Flags["cursor"] != "" {
			q.Set("cursor", req.Flags["cursor"])
		}
	}
	if v := req.Flags["version"]; v != "" && v != "latest" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("--version must be positive or latest")
		}
	}
	if cmd == "delete" {
		q.Set("wait", "false")
	}
	if cmd == "deployments/list" && req.Flags["app-id"] != "" {
		q.Set("appId", req.Flags["app-id"])
	}
	if req.Flags["include-state"] == "true" {
		q.Set("include", "state")
	}
	return q, nil
}

func appsResult(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, cmd string, body, result json.RawMessage, err error, timeout time.Duration) (json.RawMessage, error) {
	if strings.HasPrefix(cmd, "variables/") && cmd != "variables/delete" {
		return appsVariableResult(ctx, body, result, err)
	}
	if err != nil {
		return result, err
	}
	if req.Flags["wait"] != "true" {
		if cmd == "deployments/create" && appsCreationFailed(result) {
			return result, fmt.Errorf("deployment creation returned an errored status")
		}
		return result, nil
	}
	id, err := appsDeploymentID(result, cmd)
	if err != nil || id == "" {
		return result, err
	}
	polled, waitErr := appsWait(ctx, client, id, timeout)
	if polled == nil {
		return result, waitErr
	}
	return polled, waitErr
}

func appsCreationFailed(result json.RawMessage) bool {
	var response struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	return json.Unmarshal(result, &response) == nil && response.Data.Status == "errored"
}

func appsDeploymentID(result json.RawMessage, cmd string) (string, error) {
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		DestroyDeploymentID string `json:"destroyDeploymentId"`
	}
	if json.Unmarshal(result, &envelope) != nil {
		return "", fmt.Errorf("invalid deployment response")
	}
	if cmd == "delete" {
		return envelope.DestroyDeploymentID, nil
	}
	if envelope.Data.ID == "" {
		return "", fmt.Errorf("deployment response is missing its ID")
	}
	return envelope.Data.ID, nil
}

func appsVariableResult(ctx context.Context, body, result json.RawMessage, err error) (json.RawMessage, error) {
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("variable operation failed (sensitive response suppressed)")
	}
	var payload struct {
		Variable struct {
			Value string `json:"value"`
		} `json:"variable"`
	}
	if body != nil {
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("invalid variable payload")
		}
	}
	return appsRedact(result, payload.Variable.Value)
}

func appsObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	values := map[string]json.RawMessage{}
	if raw != nil {
		if json.Unmarshal(raw, &values) != nil || values == nil {
			return nil, fmt.Errorf("body must be a JSON object")
		}
	}
	return values, nil
}
func appsBody(req pluginsdk.ExecuteRequest, fields map[string]string, required ...string) (json.RawMessage, error) {
	values, err := appsObject(req.Body)
	if err != nil {
		return nil, err
	}
	for flag, key := range fields {
		if err := appsBodyFlag(req, values, flag, key); err != nil {
			return nil, err
		}
	}
	for key, raw := range values {
		if !slices.Contains(slices.Collect(maps.Values(fields)), key) {
			return nil, fmt.Errorf("unknown request field %q", key)
		}
		if err := appsValidateField(key, raw, slices.Contains(required, key)); err != nil {
			return nil, err
		}
	}
	for _, key := range required {
		if _, ok := values[key]; !ok {
			return nil, fmt.Errorf("%s is required", key)
		}
	}
	return json.Marshal(values)
}
func appsBodyFlag(req pluginsdk.ExecuteRequest, values map[string]json.RawMessage, flag, key string) error {
	if req.Flags[flag] == "" && !req.ChangedFlags[flag] {
		return nil
	}
	if _, ok := values[key]; ok {
		return fmt.Errorf("provide %s using either body or --%s", key, flag)
	}
	if key == "manifestVersion" {
		n, err := strconv.ParseInt(req.Flags[flag], 10, 64)
		if err != nil || n <= 0 {
			return fmt.Errorf("manifestVersion must be a positive integer")
		}
		values[key] = json.RawMessage(strconv.FormatInt(n, 10))
		return nil
	}
	raw, err := json.Marshal(req.Flags[flag])
	if err != nil {
		return err
	}
	values[key] = raw
	return nil
}
func appsValidateField(key string, raw json.RawMessage, required bool) error {
	if key == "manifestVersion" {
		var n int64
		if json.Unmarshal(raw, &n) != nil || n <= 0 {
			return fmt.Errorf("manifestVersion must be a positive integer")
		}
		return nil
	}
	var value string
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil {
		return fmt.Errorf("%s must be a string", key)
	}
	if required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be empty", key)
	}
	return nil
}

func appsVariableBody(req pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if req.Body != nil {
		outer, err := appsObject(req.Body)
		if err != nil {
			return nil, err
		}
		if len(outer) != 1 || outer["variable"] == nil {
			return nil, fmt.Errorf("body requires a variable object")
		}
		req.Body = outer["variable"]
	}
	inner, err := appsBody(req, map[string]string{"key": "key", "value": "value", "description": "description"}, "key", "value")
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"variable": inner})
}
func appsText(data []byte) error {
	if len(data) == 0 || len(data) > appsFileLimit || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return fmt.Errorf("YAML must be non-empty UTF-8 text within 4 MiB")
	}
	return nil
}
func appsReadFile(ctx context.Context, path string) (data []byte, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Reject devices and FIFOs before opening: opening a FIFO can block.
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > appsFileLimit {
		return nil, fmt.Errorf("--path must be a regular file within 4 MiB")
	}
	f, err := os.Open(path) // #nosec G304 -- explicit user-selected local YAML input, bounded to a regular file
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() || stat.Size() > appsFileLimit {
		return nil, fmt.Errorf("--path must be a regular file within 4 MiB")
	}
	data, err = io.ReadAll(io.LimitReader(f, appsFileLimit+1))
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, appsText(data)
}
func appsUpload(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, cmd string) (json.RawMessage, error) {
	q := url.Values{}
	segments := []string{"manifests"}
	if cmd == "manifests/create" {
		if strings.TrimSpace(req.Flags["name"]) == "" {
			return nil, fmt.Errorf("--name is required")
		}
		q.Set("name", req.Flags["name"])
	} else {
		segments = append(segments, req.Flags["manifest-id"], "versions")
	}
	data, contentType, err := appsUploadContent(ctx, req)
	if err != nil {
		return nil, err
	}
	raw, _, err := appsRaw(ctx, client, http.MethodPost, httpclient.Path(segments...), q, data, contentType, "application/json")
	if err != nil {
		return nil, err
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("deploy returned invalid JSON")
	}
	return raw, nil
}

func appsUploadContent(ctx context.Context, req pluginsdk.ExecuteRequest) ([]byte, string, error) {
	contentType := "application/json"
	data := []byte(req.Body)
	if req.Flags["path"] != "" {
		if req.Body != nil {
			return nil, "", fmt.Errorf("choose --path or --data")
		}
		var err error
		data, err = appsReadFile(ctx, req.Flags["path"])
		if err != nil {
			return nil, "", err
		}
		contentType = "application/yaml"
	} else {
		obj, err := appsObject(req.Body)
		if err != nil || req.Body == nil {
			return nil, "", fmt.Errorf("provide --path or --data manifest content")
		}
		if raw, ok := obj["yaml"]; ok {
			var yaml string
			if len(obj) != 1 || json.Unmarshal(raw, &yaml) != nil {
				return nil, "", fmt.Errorf("YAML wrapper must contain only a yaml string")
			}
			data = []byte(yaml)
			if err := appsText(data); err != nil {
				return nil, "", err
			}
			contentType = "application/yaml"
		}
	}
	return data, contentType, nil
}

// appsRaw handles historical YAML media types without changing the public SDK.
func appsRaw(ctx context.Context, client *httpclient.Client, method, path string, q url.Values, body []byte, contentType, accept string) (raw []byte, media string, err error) {
	u := client.Endpoint() + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", accept)
	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := client.HTTPClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { err = errors.Join(err, res.Body.Close()) }()
	raw, err = io.ReadAll(io.LimitReader(res.Body, appsFileLimit+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > appsFileLimit {
		return nil, "", fmt.Errorf("deploy payload exceeds 4 MiB")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", &httpclient.Error{StatusCode: res.StatusCode}
	}
	return raw, strings.Split(res.Header.Get("Content-Type"), ";")[0], nil
}
func appsDownload(ctx context.Context, client *httpclient.Client, req pluginsdk.ExecuteRequest, path string, q url.Values) (result json.RawMessage, err error) {
	accept := "application/yaml"
	if strings.HasPrefix(path, "/manifests/") {
		accept = "application/x-yaml"
	}
	raw, media, err := appsRaw(ctx, client, http.MethodGet, path, q, nil, "", accept)
	if err != nil {
		return nil, err
	}
	if media != "application/yaml" && media != "application/x-yaml" && media != "text/yaml" {
		return nil, fmt.Errorf("deploy did not return the requested YAML payload")
	}
	if err := appsText(raw); err != nil {
		return nil, err
	}
	out := req.Flags["out"]
	if out == "" {
		return json.Marshal(map[string]string{"yaml": string(raw)})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := appsWriteFile(ctx, out, raw); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"path": out, "bytes": len(raw), "mediaType": media})
}
func appsWriteFile(ctx context.Context, out string, raw []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(out), ".fctl-apps-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer func() {
		if removeErr := os.Remove(temp); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if _, err = f.Write(raw); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(temp, out); err != nil {
		return err
	}
	return nil
}

func appsWait(ctx context.Context, client *httpclient.Client, id string, timeout time.Duration) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var last json.RawMessage
	for {
		result, err := client.Do(ctx, http.MethodGet, httpclient.Path("deployments", id), nil, nil, nil)
		if err != nil {
			return last, err
		}
		last = result
		var response struct {
			Data struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		if json.Unmarshal(result, &response) != nil || response.Data.ID != id || response.Data.Status == "" {
			return result, fmt.Errorf("invalid deployment polling metadata")
		}
		switch response.Data.Status {
		case "applied", "planned_and_finished":
			return result, nil
		case "errored":
			return result, fmt.Errorf("deployment %s failed", id)
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return last, fmt.Errorf("wait for deployment %s: %w", id, ctx.Err())
		case <-timer.C:
		}
	}
}
func appsRedact(raw json.RawMessage, secrets ...string) (json.RawMessage, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("invalid variable response")
	}
	return json.Marshal(appsRedactValue(value, secrets))
}
func appsRedactValue(value any, secrets []string) any {
	switch x := value.(type) {
	case map[string]any:
		for key, child := range x {
			if strings.EqualFold(key, "value") {
				x[key] = "REDACTED"
			} else {
				x[key] = appsRedactValue(child, secrets)
			}
		}
	case []any:
		for i, child := range x {
			x[i] = appsRedactValue(child, secrets)
		}
	case string:
		return appsRedactString(x, secrets)
	}
	return value
}
func appsRedactString(value string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "REDACTED")
		}
	}
	return value
}
