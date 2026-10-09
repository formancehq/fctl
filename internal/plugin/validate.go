package plugin

import (
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

var coreFlags = map[string]bool{
	"help": true, "connection": true, "config-dir": true, "timeout": true, "output": true,
	"stack-url": true, "ledger-url": true, "auth-url": true, "connectivity-url": true, "auth-mode": true, "token-url": true,
	"client-id": true, "scopes": true, "issuer": true, "organization": true, "stack": true,
	"no-browser": true, "debug": true, "color": true, "no-input": true,
}
var coreRoots = map[string]bool{"help": true, "completion": true, "version": true, "connections": true, "login": true, "logout": true}

type validationScope struct {
	names   map[string]bool
	shorts  map[string]bool
	body    string
	confirm bool
}

func validateManifest(m pluginsdk.Manifest) error {
	if m.ProtocolVersion != pluginsdk.ProtocolVersion {
		return fmt.Errorf("unsupported plugin protocol version %d", m.ProtocolVersion)
	}
	if !identifier.MatchString(m.Name) || !identifier.MatchString(m.Service) || strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("plugin name, service and version are required and must be valid")
	}
	fields := strings.Fields(m.Root.Use)
	if len(fields) == 0 || fields[0] != m.Name || coreRoots[fields[0]] {
		return fmt.Errorf("invalid or reserved plugin root %q", m.Root.Use)
	}
	if err := validateCommand(m.Root, validationScope{names: map[string]bool{}, shorts: map[string]bool{"h": true, "o": true}}); err != nil {
		return err
	}
	return validateInteraction(m)
}

func validateCommand(spec pluginsdk.CommandSpec, inherited validationScope) error {
	if err := validateCommandHeader(spec); err != nil {
		return err
	}
	scope := validationScope{names: maps.Clone(inherited.names), shorts: maps.Clone(inherited.shorts), body: inherited.body, confirm: inherited.confirm}
	next := validationScope{names: maps.Clone(inherited.names), shorts: maps.Clone(inherited.shorts), body: inherited.body, confirm: inherited.confirm}
	for _, flag := range spec.Flags {
		if err := validateFlag(flag, &scope); err != nil {
			return fmt.Errorf("command %q: %w", spec.Use, err)
		}
		if flag.Persistent {
			if err := validateFlag(flag, &next); err != nil {
				return err
			}
		}
	}
	if spec.Confirm && !scope.confirm {
		return fmt.Errorf("command %q requires a boolean confirm flag", spec.Use)
	}
	return validateChildren(spec, next)
}

func validateCommandHeader(spec pluginsdk.CommandSpec) error {
	if spec.Service != "" && !identifier.MatchString(spec.Service) {
		return fmt.Errorf("invalid command service %q", spec.Service)
	}
	fields := strings.Fields(spec.Use)
	if len(fields) == 0 || !identifier.MatchString(fields[0]) || fields[0] == "help" {
		return fmt.Errorf("invalid plugin command use %q", spec.Use)
	}
	if strings.TrimSpace(spec.Use) != spec.Use || strings.ContainsAny(spec.Use, "\t\r\n") {
		return fmt.Errorf("invalid plugin command whitespace %q", spec.Use)
	}
	if spec.Args.Min < 0 || spec.Args.Max < spec.Args.Min {
		return fmt.Errorf("invalid argument bounds for %q", spec.Use)
	}
	if !spec.Runnable && (spec.Confirm || spec.Args.Min != 0 || spec.Args.Max != 0) {
		return fmt.Errorf("non-runnable command %q has execution constraints", spec.Use)
	}
	return nil
}

func validateChildren(spec pluginsdk.CommandSpec, next validationScope) error {
	names := map[string]bool{}
	for _, child := range spec.Subcommands {
		if err := validateCommand(child, next); err != nil {
			return err
		}
		name := pluginsdk.CommandName(child)
		if names[name] {
			return fmt.Errorf("duplicate command %q under %q", name, spec.Use)
		}
		names[name] = true
	}
	return nil
}

func validateFlag(flag pluginsdk.FlagSpec, scope *validationScope) error {
	if !identifier.MatchString(flag.Name) || coreFlags[flag.Name] || scope.names[flag.Name] {
		return fmt.Errorf("invalid, reserved or duplicate flag %q", flag.Name)
	}
	if flag.Shorthand != "" && (!shorthandName.MatchString(flag.Shorthand) || scope.shorts[flag.Shorthand]) {
		return fmt.Errorf("invalid or duplicate shorthand %q", flag.Shorthand)
	}
	if err := validateDefault(flag); err != nil {
		return err
	}
	if err := validateSpecialFlag(flag, scope); err != nil {
		return err
	}
	scope.names[flag.Name] = true
	if flag.Shorthand != "" {
		scope.shorts[flag.Shorthand] = true
	}
	return nil
}

var shorthandName = regexp.MustCompile(`^[a-zA-Z]$`)

func validateSpecialFlag(flag pluginsdk.FlagSpec, scope *validationScope) error {
	if flag.RequireTrue && flag.Type != "bool" {
		return fmt.Errorf("a required enabled flag must be boolean")
	}
	if flag.Body {
		if flag.Type != "string" || scope.body != "" {
			return fmt.Errorf("body must be a single string flag")
		}
		scope.body = flag.Name
	}
	if flag.Name == "confirm" {
		if flag.Type != "bool" || flag.Default != "false" {
			return fmt.Errorf("confirm must be a boolean defaulting to false")
		}
		scope.confirm = true
	}
	return nil
}

func validateDefault(flag pluginsdk.FlagSpec) error {
	switch flag.Type {
	case "string":
		return nil
	case "bool":
		if flag.Default == "true" || flag.Default == "false" {
			return nil
		}
	case "uint32":
		if _, err := strconv.ParseUint(flag.Default, 10, 32); err == nil {
			return nil
		}
	default:
		return fmt.Errorf("unsupported flag type %q", flag.Type)
	}
	return fmt.Errorf("invalid default for flag %q", flag.Name)
}
