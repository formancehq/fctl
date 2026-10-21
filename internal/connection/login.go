package connection

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/cloud"
)

// LoginProfile resolves Cloud settings without writing the store or requiring
// a target. Explicit
// local profiles are rejected; an implicit active local profile remains intact.
func (s *Settings) LoginProfile(ctx context.Context, cmd *cobra.Command) (Options, Entry, string, string, error) {
	dir, err := s.DirectoryPath()
	if err != nil {
		return Options{}, Entry{}, "", "", err
	}
	requested := cmp.Or(s.Name, os.Getenv("FCTL_PROFILE"))
	store, err := Load(dir)
	if err != nil {
		return Options{}, Entry{}, "", "", err
	}
	name := loginName(requested, store)
	if err := ValidateName(name); err != nil {
		return Options{}, Entry{}, "", "", err
	}
	entry, exists := store.Connections[name]
	if exists && entry.Options.AuthMode != "cloud" {
		return Options{}, Entry{}, "", "", fmt.Errorf("login requires a Cloud profile; %q uses %s", name, entry.Options.AuthMode)
	}
	options := s.loginOptions(cmd, entry.Options)
	if options.AuthMode != "cloud" {
		return Options{}, Entry{}, "", "", fmt.Errorf("login requires auth-mode=cloud")
	}
	if err := Validate(options); err != nil {
		return Options{}, Entry{}, "", "", err
	}
	if err := ctx.Err(); err != nil {
		return Options{}, Entry{}, "", "", err
	}
	return options, entry, name, dir, nil
}

func loginName(requested string, store Store) string {
	if requested != "" {
		return requested
	}
	if current, exists := store.Connections[store.Active]; exists && current.Options.AuthMode == "cloud" {
		return store.Active
	}
	return "cloud"
}

func (s *Settings) loginOptions(cmd *cobra.Command, saved Options) Options {
	resolved := Settings{Options: saved}
	fields := resolved.fields()
	for i, setting := range s.fields() {
		if cmd.Root().PersistentFlags().Changed(setting.name) {
			*fields[i].value = *setting.value
		} else if value, exists := os.LookupEnv("FCTL_" + strings.ToUpper(strings.ReplaceAll(setting.name, "-", "_"))); exists {
			*fields[i].value = value
		}
	}
	resolved.Options.AuthMode = cmp.Or(resolved.Options.AuthMode, "cloud")
	resolved.Options.Issuer = cmp.Or(resolved.Options.Issuer, cloud.DefaultIssuer)
	resolved.Options.ClientID = cmp.Or(resolved.Options.ClientID, "fctl")
	return resolved.Options
}
