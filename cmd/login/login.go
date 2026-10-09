// Package login creates or renews Cloud identities using the Membership device flow.
package login

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func NewCommand(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "login", Short: "Log in to the Cloud (creates a profile automatically)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return runLogin(s, cmd)
	}}
}

func runLogin(s *connection.Settings, cmd *cobra.Command) error {
	client, err := loginHTTPClient(s, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	opts, entry, name, dir, err := s.LoginProfile(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	session, err := cloud.LoginIdentity(cmd.Context(), client, cloud.Options{Issuer: opts.Issuer, ClientID: opts.ClientID}, cmd.ErrOrStderr(), s.BrowserOpener())
	if err != nil {
		return err
	}
	expected := entry.Revision
	if err := connection.SaveLoginSession(cmd.Context(), dir, name, &expected, opts, session); err != nil {
		return err
	}
	return command.WriteJSON(cmd.OutOrStdout(), []byte(`{"loggedIn":true}`))
}

func loginHTTPClient(s *connection.Settings, out io.Writer) (*http.Client, error) {
	if s.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	return s.HTTPClient(out), nil
}

func NewLogoutCommand(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "Remove the selected profile's local Cloud tokens", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		_, _, name, dir, err := s.Resolve(cmd)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("select a saved profile")
		}
		if err := connection.Update(cmd.Context(), dir, func(store *connection.Store) error {
			entry, exists := store.Connections[name]
			if !exists {
				return fmt.Errorf("profile no longer exists")
			}
			store.Connections[name] = connection.NewEntry(entry.Options)
			return nil
		}); err != nil {
			return err
		}
		return command.WriteJSON(cmd.OutOrStdout(), []byte(`{"loggedIn":false}`))
	}}
}
