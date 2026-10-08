// Package pluginsdk defines the public contract for embedded and future external plugins.
// It has no dependency on Cobra, profile storage or the core's authentication.
package pluginsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"
)

const ProtocolVersion = 1

type Plugin interface {
	GetManifest(context.Context) (Manifest, error)
	Execute(context.Context, ExecuteRequest) (ExecuteResponse, error)
}

type Manifest struct {
	Name            string      `json:"name"`
	Version         string      `json:"version"`
	Service         string      `json:"service"`
	ProtocolVersion int         `json:"protocolVersion"`
	Root            CommandSpec `json:"root"`
}

type CommandSpec struct {
	Use         string        `json:"use"`
	Short       string        `json:"short"`
	Long        string        `json:"long,omitempty"`
	Example     string        `json:"example,omitempty"`
	Args        ArgsSpec      `json:"args"`
	Flags       []FlagSpec    `json:"flags,omitempty"`
	Subcommands []CommandSpec `json:"subcommands,omitempty"`
	Runnable    bool          `json:"runnable"`
	Confirm     bool          `json:"confirm,omitempty"`
}

type ArgsSpec struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type FlagSpec struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Type       string `json:"type"`
	Default    string `json:"default"`
	Usage      string `json:"usage"`
	Required   bool   `json:"required,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
	Body       bool   `json:"body,omitempty"`
}

// ExecuteRequest is transport-neutral. Endpoint is resolved by the host;
// Body contains JSON already read from inline input, a file or stdin.
// Authentication is supplied by the plugin instance's injected HTTP client.
type ExecuteRequest struct {
	CommandPath  []string          `json:"commandPath"`
	Args         []string          `json:"args,omitempty"`
	Flags        map[string]string `json:"flags,omitempty"`
	ChangedFlags map[string]bool   `json:"changedFlags,omitempty"`
	Body         json.RawMessage   `json:"body,omitempty"`
	Endpoint     string            `json:"endpoint"`
}

// A response can accompany an error, for example to preserve bulk partial results.
type ExecuteResponse struct {
	Data json.RawMessage `json:"data,omitempty"`
}

func CommandName(command CommandSpec) string {
	words := strings.Fields(command.Use)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}

func FindCommand(manifest Manifest, path []string) (CommandSpec, error) {
	command, _, err := find(manifest, path)
	return command, err
}

func find(manifest Manifest, path []string) (CommandSpec, []FlagSpec, error) {
	if len(path) == 0 || len(strings.Fields(manifest.Root.Use)) == 0 || path[0] != CommandName(manifest.Root) {
		return CommandSpec{}, nil, fmt.Errorf("unknown plugin command")
	}
	command := manifest.Root
	var inherited []FlagSpec
	for _, name := range path[1:] {
		for _, flag := range command.Flags {
			if flag.Persistent {
				inherited = append(inherited, flag)
			}
		}
		child, found := findChild(command, name)
		if !found {
			return CommandSpec{}, nil, fmt.Errorf("unknown plugin command %q", strings.Join(path, " "))
		}
		command = child
	}
	return command, append(inherited, command.Flags...), nil
}

func findChild(command CommandSpec, name string) (CommandSpec, bool) {
	for _, child := range command.Subcommands {
		if CommandName(child) == name {
			return child, true
		}
	}
	return CommandSpec{}, false
}

// NormalizeRequest also protects direct SDK callers, independently of the CLI adapter.
func NormalizeRequest(manifest Manifest, request ExecuteRequest) (ExecuteRequest, error) {
	if manifest.ProtocolVersion != ProtocolVersion {
		return ExecuteRequest{}, fmt.Errorf("unsupported plugin protocol version %d", manifest.ProtocolVersion)
	}
	command, flags, err := find(manifest, request.CommandPath)
	if err != nil {
		return ExecuteRequest{}, err
	}
	if !command.Runnable {
		return ExecuteRequest{}, fmt.Errorf("command is not executable")
	}
	if len(request.Args) < command.Args.Min || len(request.Args) > command.Args.Max {
		return ExecuteRequest{}, fmt.Errorf("command expects %d through %d arguments", command.Args.Min, command.Args.Max)
	}
	request.Flags, err = normalizeFlags(flags, request)
	if err != nil {
		return ExecuteRequest{}, err
	}
	if command.Confirm && request.Flags["confirm"] != "true" {
		return ExecuteRequest{}, fmt.Errorf("%s requires --confirm", strings.Join(request.CommandPath, " "))
	}
	if request.Body != nil && (!json.Valid(request.Body) || len(request.Body) > 4<<20) {
		return ExecuteRequest{}, fmt.Errorf("request body must be valid JSON within 4 MiB")
	}
	return request, nil
}

func normalizeFlags(flags []FlagSpec, request ExecuteRequest) (map[string]string, error) {
	request.Flags = maps.Clone(request.Flags)
	if request.Flags == nil {
		request.Flags = make(map[string]string)
	}
	known := make(map[string]FlagSpec, len(flags))
	for _, flag := range flags {
		if _, exists := known[flag.Name]; exists {
			return nil, fmt.Errorf("duplicate plugin flag %q", flag.Name)
		}
		known[flag.Name] = flag
		if _, exists := request.Flags[flag.Name]; !exists {
			request.Flags[flag.Name] = flag.Default
		}
		if flag.Body && request.Body != nil {
			flag.Required = false
		}
		if err := validateFlag(flag, request.Flags[flag.Name]); err != nil {
			return nil, err
		}
	}
	if err := validateKnown(request.Flags, known); err != nil {
		return nil, err
	}
	if err := validateKnown(request.ChangedFlags, known); err != nil {
		return nil, err
	}

	return request.Flags, nil
}

func validateKnown[T any](values map[string]T, known map[string]FlagSpec) error {
	for name := range values {
		if _, exists := known[name]; !exists {
			return fmt.Errorf("unknown plugin flag %q", name)
		}
	}
	return nil
}

func validateFlag(flag FlagSpec, value string) error {
	if flag.Required && value == "" {
		return fmt.Errorf("--%s is required", flag.Name)
	}
	switch flag.Type {
	case "string":
		return nil
	case "bool":
		if value == "true" || value == "false" {
			return nil
		}
	case "uint32":
		if _, err := strconv.ParseUint(value, 10, 32); err == nil {
			return nil
		}
	default:
		return fmt.Errorf("unsupported flag type %q", flag.Type)
	}
	return fmt.Errorf("invalid --%s %s value", flag.Name, flag.Type)
}
