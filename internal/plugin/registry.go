// Package plugin adapts serializable plugin manifests to the host CLI.
package plugin

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"sync"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

// Factory injects the host's authenticated HTTP client into a plugin instance.
type Factory func(*http.Client) pluginsdk.Plugin

type registration struct {
	manifest pluginsdk.Manifest
	factory  Factory
}

// Registry accepts embedded plugins during startup and freezes before attachment.
// Its zero value is ready for registration. It never executes service requests.
type Registry struct {
	mu      sync.Mutex
	frozen  bool
	entries []registration
}

// Register reads and validates metadata without resolving an authenticated client.
func (r *Registry) Register(ctx context.Context, metadata pluginsdk.Plugin, factory Factory) error {
	if metadata == nil || factory == nil {
		return fmt.Errorf("plugin metadata and factory are required")
	}
	manifest, err := metadata.GetManifest(ctx)
	if err != nil {
		return fmt.Errorf("get plugin manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return fmt.Errorf("plugin registry is frozen")
	}
	for _, entry := range r.entries {
		if entry.manifest.Name == manifest.Name || pluginsdk.CommandName(entry.manifest.Root) == pluginsdk.CommandName(manifest.Root) {
			return fmt.Errorf("duplicate plugin root %q", manifest.Root.Use)
		}
	}
	r.entries = append(r.entries, registration{cloneManifest(manifest), factory})
	return nil
}

// Freeze prevents further registrations. List returns independent metadata copies.
func (r *Registry) Freeze() { r.mu.Lock(); defer r.mu.Unlock(); r.frozen = true }

func (r *Registry) List() []pluginsdk.Manifest {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]pluginsdk.Manifest, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, cloneManifest(entry.manifest))
	}
	return result
}

func (r *Registry) snapshot() []registration {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frozen = true
	entries := slices.Clone(r.entries)
	for i := range entries {
		entries[i].manifest = cloneManifest(entries[i].manifest)
	}
	return entries
}

func cloneManifest(manifest pluginsdk.Manifest) pluginsdk.Manifest {
	manifest.Root = cloneCommand(manifest.Root)
	return manifest
}
func cloneCommand(command pluginsdk.CommandSpec) pluginsdk.CommandSpec {
	command.Flags = slices.Clone(command.Flags)
	command.Inputs = cloneInputs(command.Inputs)
	command.Subcommands = slices.Clone(command.Subcommands)
	for i := range command.Subcommands {
		command.Subcommands[i] = cloneCommand(command.Subcommands[i])
	}
	return command
}

func cloneInputs(inputs []pluginsdk.InputSpec) []pluginsdk.InputSpec {
	result := slices.Clone(inputs)
	for i := range result {
		input := &result[i]
		input.Options = slices.Clone(input.Options)
		if input.Argument != nil {
			value := *input.Argument
			input.Argument = &value
		}
		if input.AlternativeArgument != nil {
			value := *input.AlternativeArgument
			input.AlternativeArgument = &value
		}
		if input.Source != nil {
			input.Source = cloneChoiceSource(*input.Source)
		}
	}
	return result
}

func cloneChoiceSource(source pluginsdk.ChoiceSource) *pluginsdk.ChoiceSource {
	source.CommandPath = slices.Clone(source.CommandPath)
	source.Args = slices.Clone(source.Args)
	source.LabelFields = slices.Clone(source.LabelFields)
	source.ExcludeTrueFields = slices.Clone(source.ExcludeTrueFields)
	source.Flags = maps.Clone(source.Flags)
	source.MatchFields = maps.Clone(source.MatchFields)
	for field, values := range source.MatchFields {
		source.MatchFields[field] = slices.Clone(values)
	}
	return &source
}
