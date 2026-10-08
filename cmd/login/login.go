// Package login connects saved Cloud profiles using the Membership device flow.
package login

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func NewCommand(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "login", Short: "Log in to a saved Cloud connection", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return runLogin(s, cmd)
	}}
}

func runLogin(s *connection.Settings, cmd *cobra.Command) error {
	opts, entry, name, dir, err := s.Resolve(cmd)
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("save a Cloud connection with connections add before logging in")
	}
	if err := connection.Validate(opts); err != nil {
		return err
	}
	if opts.AuthMode != "cloud" {
		return fmt.Errorf("login requires auth-mode=cloud")
	}
	if opts != entry.Options {
		return fmt.Errorf("cloud settings changed; replace the connection before logging in")
	}
	client, err := loginHTTPClient(s)
	if err != nil {
		return err
	}
	session, err := cloud.Login(cmd.Context(), client, cloud.Options{Issuer: opts.Issuer, ClientID: opts.ClientID, Organization: opts.Organization, Stack: opts.Stack}, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	expected := entry.Revision
	if err := connection.SaveSession(cmd.Context(), dir, name, &expected, session); err != nil {
		return err
	}
	return command.WriteJSON(cmd.OutOrStdout(), []byte(`{"loggedIn":true}`))
}

func loginHTTPClient(s *connection.Settings) (*http.Client, error) {
	if s.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	return &http.Client{Timeout: s.Timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, nil
}

func NewLogoutCommand(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "Remove the selected connection's local Cloud tokens", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		_, _, name, dir, err := s.Resolve(cmd)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("select a saved connection")
		}
		if err := connection.Update(cmd.Context(), dir, func(store *connection.Store) error {
			entry, exists := store.Connections[name]
			if !exists {
				return fmt.Errorf("connection no longer exists")
			}
			store.Connections[name] = connection.NewEntry(entry.Options)
			return nil
		}); err != nil {
			return err
		}
		return command.WriteJSON(cmd.OutOrStdout(), []byte(`{"loggedIn":false}`))
	}}
}
