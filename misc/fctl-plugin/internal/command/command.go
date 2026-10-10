// Package command builds and executes declarative SDK commands without service-specific rules.
package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type Operation struct {
	Command       string
	Spec          pluginsdk.CommandSpec
	Method        string
	Segments      []string          // $0, $1, ... bind arguments; @name binds a flag.
	Query         map[string]string // flag to wire name
	Body          func(pluginsdk.ExecuteRequest) (json.RawMessage, error)
	ValidateRoute func(pluginsdk.ExecuteRequest, Operation) error
	Run           func(context.Context, *httpclient.Client, pluginsdk.ExecuteRequest, Operation, json.RawMessage) (json.RawMessage, error)
}

type plugin struct {
	client     *http.Client
	manifest   pluginsdk.Manifest
	operations map[string]Operation
}

func New(name, version string, client *http.Client, operations []Operation, aliases map[string][]string) pluginsdk.Plugin {
	p := &plugin{client: client, operations: make(map[string]Operation)}
	p.manifest = pluginsdk.Manifest{Name: name, Version: version, Service: name,
		ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{Use: name, Short: "Manage legacy " + name, Target: "stack"}}
	for _, op := range operations {
		p.operations[op.Command] = op
		addCommand(&p.manifest.Root, strings.Fields(op.Command), op.Spec)
	}
	applyAliases(&p.manifest.Root, nil, aliases)
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
	if op.Run != nil {
		data, err = op.Run(ctx, client, r, op, body)
	} else {
		data, err = Perform(ctx, client, r, op, body)
	}
	if err != nil {
		var failure *httpclient.Error
		if errors.As(err, &failure) && len(failure.Body) > 0 {
			data = failure.Body
		}
	}
	return pluginsdk.ExecuteResponse{Data: data}, err
}

func prepareOperation(op Operation, r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if r.Body != nil && !acceptsBody(op.Spec.Flags) {
		return nil, fmt.Errorf("this command does not accept a JSON body")
	}
	if op.ValidateRoute != nil {
		if err := op.ValidateRoute(r, op); err != nil {
			return nil, err
		}
	} else if _, err := Route(op.Segments, r); err != nil {
		return nil, err
	}
	if _, err := QueryValues(op.Query, r); err != nil {
		return nil, err
	}
	if op.Body != nil {
		return op.Body(r)
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
