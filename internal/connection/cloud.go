package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/presentation"
)

func cloudCoordinator(dir, name string, entry Entry) cloud.Coordinator {
	return func(ctx context.Context, work func(*cloud.Session, func(*cloud.Session) error) (*oauth2.Token, error)) (*oauth2.Token, error) {
		var token *oauth2.Token
		err := withLock(ctx, dir, "auth-"+name+".lock", func() error {
			store, err := Load(dir)
			if err != nil {
				return err
			}
			current, exists := store.Connections[name]
			if !exists || current.Session == nil || current.Options != entry.Options {
				return fmt.Errorf("cloud connection changed or logged out during authentication")
			}
			expected := current.Revision
			token, err = work(current.Session, func(session *cloud.Session) error {
				saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				return SaveSession(saveCtx, dir, name, &expected, session)
			})
			return err
		})
		return token, err
	}
}

func (s *Settings) membershipClient(ctx context.Context, cmd *cobra.Command, client *http.Client, options Options, entry Entry, name, dir string) (*api.Client, error) {
	if options.AuthMode != "cloud" {
		return nil, fmt.Errorf("cloud commands require a Cloud connection; run fctl login")
	}
	if cloudIdentity(options) != cloudIdentity(entry.Options) {
		return nil, fmt.Errorf("cloud identity settings changed; log in again with the chosen issuer and client")
	}
	if entry.Session == nil {
		return nil, fmt.Errorf("connection is not logged in; run fctl login")
	}
	if entry.Session.Options.Stack != "" {
		return nil, fmt.Errorf("this older session targets a single stack; run fctl login to use Cloud management")
	}
	authenticated, issuer, identity, err := cloud.MembershipClient(ctx, client, entry.Session, cmd.ErrOrStderr(), s.BrowserOpener(), cloudCoordinator(dir, name, entry))
	if err != nil {
		return nil, err
	}
	resolved, err := api.New(issuer, authenticated)
	if err != nil {
		return nil, err
	}
	organization := options.Organization
	if organization == "" && len(identity.Organizations) == 1 {
		organization = identity.Organizations[0]
	}
	stack := options.Stack
	if stack == "" && len(identity.Stacks[organization]) == 1 {
		stack = identity.Stacks[organization][0]
	}
	organizations, err := json.Marshal(identity.Organizations)
	if err != nil {
		return nil, err
	}
	stacks, err := json.Marshal(identity.Stacks)
	if err != nil {
		return nil, err
	}
	return resolved.WithContext(map[string]string{"organization": organization, "stack": stack, "organizations": string(organizations), "stacks": string(stacks)}), nil
}

// ApplicationClient resolves an application grant independently of stack Auth.
func (s *Settings) ApplicationClient(ctx context.Context, cmd *cobra.Command, alias string) (*api.Client, error) {
	if err := presentation.ValidateFormat(s.Output); err != nil {
		return nil, err
	}
	if s.Timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	options, entry, name, dir, err := s.Resolve(cmd)
	if err != nil {
		return nil, err
	}
	if err := Validate(options); err != nil {
		return nil, err
	}
	if options.AuthMode != "cloud" || entry.Session == nil {
		return nil, fmt.Errorf("cloud apps require a logged-in Cloud connection; run fctl login")
	}
	if cloudIdentity(options) != cloudIdentity(entry.Options) || entry.Session.Options.Stack != "" {
		return nil, fmt.Errorf("cloud identity settings changed or use an older session; run fctl login")
	}
	base := s.HTTPClient(cmd.ErrOrStderr())
	coordinator := cloudCoordinator(dir, name, entry)
	_, _, identity, err := cloud.MembershipClient(ctx, base, entry.Session, cmd.ErrOrStderr(), s.BrowserOpener(), coordinator)
	if err != nil {
		return nil, err
	}
	org := options.Organization
	if org == "" && len(identity.Organizations) == 1 {
		org = identity.Organizations[0]
	}
	if org == "" {
		return nil, fmt.Errorf("select an organization with --organization; available IDs: %s", strings.Join(identity.Organizations, ", "))
	}
	if alias == "" {
		alias = "deploy"
	}
	client, endpoint, err := cloud.ApplicationClient(ctx, base, entry.Session, org, alias, cmd.ErrOrStderr(), s.BrowserOpener(), coordinator)
	if err != nil {
		return nil, err
	}
	resolved, err := api.New(endpoint, client)
	if err != nil {
		return nil, err
	}
	return resolved.WithContext(map[string]string{"organization": org, "application": alias}), nil
}
