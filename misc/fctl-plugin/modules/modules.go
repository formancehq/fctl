// Package modules supplies the historical Stack modules through the public plugin SDK.
package modules

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

// Version is the legacy bundle revision, independent of backend service versions.
const Version = "1.0.0"

// Factories returns independent factories. The host supplies endpoint and authentication.
func Factories() map[string]func(*http.Client) pluginsdk.Plugin {
	return map[string]func(*http.Client) pluginsdk.Plugin{
		"orchestration":  newOrchestration,
		"payments":       newPayments,
		"reconciliation": newReconciliation,
		"wallets":        newWallets,
		"webhooks":       newWebhooks,
	}
}

type operation struct {
	command  string
	spec     pluginsdk.CommandSpec
	method   string
	segments []string          // $0, $1, ... bind arguments; @name binds a flag.
	query    map[string]string // flag to wire name
	body     func(pluginsdk.ExecuteRequest) (json.RawMessage, error)
	run      func(context.Context, *httpclient.Client, pluginsdk.ExecuteRequest, operation, json.RawMessage) (json.RawMessage, error)
}

type plugin struct {
	client     *http.Client
	manifest   pluginsdk.Manifest
	operations map[string]operation
}

func newPlugin(name string, client *http.Client, operations []operation) pluginsdk.Plugin {
	p := &plugin{client: client, operations: make(map[string]operation)}
	p.manifest = pluginsdk.Manifest{Name: name, Version: Version, Service: name,
		ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{Use: name, Short: "Manage legacy " + name, Target: "stack"}}
	for _, op := range operations {
		p.operations[op.command] = op
		addCommand(&p.manifest.Root, strings.Fields(op.command), op.spec)
	}
	applyAliases(name, &p.manifest.Root, nil)
	return p
}

func addCommand(root *pluginsdk.CommandSpec, path []string, spec pluginsdk.CommandSpec) {
	if len(path) == 1 {
		root.Subcommands = append(root.Subcommands, spec)
		return
	}
	for i := range root.Subcommands {
		if pluginsdk.CommandName(root.Subcommands[i]) == path[0] {
			addCommand(&root.Subcommands[i], path[1:], spec)
			return
		}
	}
	root.Subcommands = append(root.Subcommands, pluginsdk.CommandSpec{Use: path[0], Short: "Manage " + path[0]})
	addCommand(&root.Subcommands[len(root.Subcommands)-1], path[1:], spec)
}

func (p *plugin) GetManifest(context.Context) (pluginsdk.Manifest, error) {
	// Return independent metadata to callers that modify their local copy.
	data, err := json.Marshal(p.manifest)
	if err != nil {
		return pluginsdk.Manifest{}, err
	}
	var manifest pluginsdk.Manifest
	err = json.Unmarshal(data, &manifest)
	return manifest, err
}

func (p *plugin) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	r, err := pluginsdk.NormalizeRequest(p.manifest, request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	op := p.operations[strings.Join(r.CommandPath[1:], " ")]
	body, err := prepareOperation(op, r)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	if p.client == nil {
		return pluginsdk.ExecuteResponse{}, fmt.Errorf("%s requires an injected HTTP client", p.manifest.Name)
	}
	client, err := httpclient.New(r.Endpoint, p.client)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	var data json.RawMessage
	if op.run != nil {
		data, err = op.run(ctx, client, r, op, body)
	} else {
		data, err = perform(ctx, client, r, op, body)
	}
	if err != nil {
		var failure *httpclient.Error
		if errors.As(err, &failure) && len(failure.Body) > 0 {
			data = failure.Body
		}
	}
	return pluginsdk.ExecuteResponse{Data: data}, err
}

func prepareOperation(op operation, r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil && !acceptsBody(op.spec.Flags) {
		return nil, fmt.Errorf("this command does not accept a JSON body")
	}
	if _, err := route(op.segments, r); err != nil {
		return nil, err
	}
	if _, err := queryValues(op.query, r); err != nil {
		return nil, err
	}
	if op.body != nil {
		return op.body(r)
	}
	return nil, nil
}

func acceptsBody(flags []pluginsdk.FlagSpec) bool {
	for _, flag := range flags {
		if flag.Body {
			return true
		}
	}
	return false
}

func perform(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	path, err := route(op.segments, r)
	if err != nil {
		return nil, err
	}
	query, err := queryValues(op.query, r)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	key := r.Flags["idempotency-key"]
	if key == "" {
		key = r.Flags["ik"]
	}
	if key != "" {
		headers.Set("Idempotency-Key", key)
	}
	return client.Do(ctx, op.method, path, query, body, headers)
}

func identifier(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." {
		return fmt.Errorf("identifier must be nonempty and cannot be a dot segment")
	}
	return nil
}

func route(segments []string, r pluginsdk.ExecuteRequest) (string, error) {
	resolved := make([]string, len(segments))
	for i, segment := range segments {
		value := segment
		if strings.HasPrefix(segment, "$") {
			n, err := strconv.Atoi(segment[1:])
			if err != nil || n < 0 || n >= len(r.Args) {
				return "", fmt.Errorf("missing route argument %s", segment)
			}
			value = r.Args[n]
		} else if strings.HasPrefix(segment, "@") {
			value = r.Flags[segment[1:]]
			// Wallet names resolve through a read before the final request.
			if segment == "@id" && value == "" && r.Flags["name"] != "" {
				value = "resolved-wallet"
			}
		}
		if err := identifier(value); err != nil {
			return "", fmt.Errorf("%s: %w", segment, err)
		}
		resolved[i] = value
	}
	return httpclient.Path(resolved...), nil
}

func queryValues(bindings map[string]string, r pluginsdk.ExecuteRequest) (url.Values, error) {
	query := url.Values{}
	for flag, wire := range bindings {
		value := r.Flags[flag]
		if value == "" {
			continue
		}
		if flag == "page-size" {
			n, err := strconv.ParseUint(value, 10, 32)
			if err != nil || n == 0 || n > 1000 {
				return nil, fmt.Errorf("--page-size must be between 1 and 1000")
			}
		}
		if strings.Contains(flag, "created-at") || flag == "at" || flag == "timestamp" {
			if err := timestamp(value); err != nil {
				return nil, fmt.Errorf("--%s: %w", flag, err)
			}
		}
		query.Set(wire, value)
	}
	return query, nil
}

func timestamp(value string) error {
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return fmt.Errorf("expected RFC3339 timestamp: %w", err)
	}
	return nil
}

func str(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "string", Usage: usage}
}
func boolean(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "bool", Default: "false", Usage: usage}
}
func pagination() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{str("cursor", "Opaque pagination cursor"), {Name: "page-size", Type: "uint32", Default: "100", Usage: "Page size (1-1000)"}}
}
func dataFlag() pluginsdk.FlagSpec {
	f := str("data", "JSON payload, @file or - (read by the host)")
	f.Body = true
	return f
}

func leaf(command, use, method, path string, minArgs, maxArgs int, flags ...pluginsdk.FlagSpec) operation {
	op := operation{command: command, method: method, segments: strings.Split(strings.Trim(path, "/"), "/"),
		spec: pluginsdk.CommandSpec{Use: use, Short: strings.ToUpper(command[:1]) + command[1:], Runnable: true, Args: pluginsdk.ArgsSpec{Min: minArgs, Max: maxArgs}, Flags: flags}}
	for i := range minArgs {
		op.spec.Inputs = append(op.spec.Inputs, pluginsdk.InputSpec{Title: fmt.Sprintf("%s argument %d", command, i+1), Kind: "text", Argument: new(i), Required: true})
	}
	return op
}

func confirmed(op operation) operation {
	op.spec.Confirm = true
	op.spec.Flags = append(op.spec.Flags, boolean("confirm", "Confirm this operation"))
	return op
}

func payload(op operation, fields ...string) operation {
	// Historical file/stdin/URL input stays a host-owned boundary. Direct
	// callers and forms can still supply Body without the optional source.
	source := op.spec.Args.Max
	op.spec.Files = &pluginsdk.FileSpec{ReadArgument: &source, ReadFormat: "yaml"}
	op.spec.Args.Max++
	op.spec.Use += " [<file>|-]"
	op.spec.Flags = append(op.spec.Flags, dataFlag())
	op.spec.Inputs = append(op.spec.Inputs, pluginsdk.InputSpec{Title: "JSON payload", Kind: "text", ValueType: "string", Flag: "data", Required: true})
	op.body = func(r pluginsdk.ExecuteRequest) (json.RawMessage, error) { return objectBody(r.Body, fields...) }
	return op
}

func objectBody(body json.RawMessage, required ...string) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if len(body) == 0 || json.Unmarshal(body, &object) != nil || object == nil {
		return nil, fmt.Errorf("a JSON object payload is required (--data)")
	}
	for _, field := range required {
		v := strings.TrimSpace(string(object[field]))
		if v == "" || v == "null" || v == `""` {
			return nil, fmt.Errorf("payload field %q is required", field)
		}
	}
	return body, nil
}

func marshal(value any) (json.RawMessage, error) { return json.Marshal(value) }

// list accepts the SDK's string flag representation as JSON or CSV.
func list(value string) ([]string, error) {
	if value == "" {
		return []string{}, nil
	}
	var values []string
	if strings.HasPrefix(strings.TrimSpace(value), "[") {
		if err := json.Unmarshal([]byte(value), &values); err != nil {
			return nil, fmt.Errorf("expected a JSON string array: %w", err)
		}
	} else {
		var err error
		values, err = csv.NewReader(strings.NewReader(value)).Read()
		if err != nil {
			return nil, fmt.Errorf("expected comma-separated values: %w", err)
		}
	}
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("list entries cannot be empty")
		}
	}
	return values, nil
}

func pairs(value string) (map[string]string, error) {
	if strings.HasPrefix(strings.TrimSpace(value), "{") {
		var out map[string]string
		if err := json.Unmarshal([]byte(value), &out); err != nil || out == nil {
			return nil, fmt.Errorf("expected a JSON object with string values")
		}
		return out, nil
	}
	values, err := list(value)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, v := range values {
		key, val, ok := strings.Cut(v, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("expected key=value, got %q", v)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		out[key] = val
	}
	return out, nil
}
