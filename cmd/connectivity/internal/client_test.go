package internal

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v4/oidc"
	oidcclient "github.com/formancehq/go-libs/v4/oidc/client"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestClientFactoryNonInteractiveModeRejectsUnusableTokensBeforeAuthentication(t *testing.T) {
	tests := map[string]*fctl.AccessToken{
		"missing": nil,
		"expired": accessTokenExpiringAt(time.Now().Add(-time.Minute)),
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			var terminal, commandOutput bytes.Buffer
			originalLogger := pterm.DefaultLogger
			pterm.DefaultLogger.Writer = &terminal
			t.Cleanup(func() { pterm.DefaultLogger = originalLogger })

			authenticationStarted := false
			factory := newClientFactory(clientFactoryDependencies{
				loadAndAuthenticateCurrentProfile: testProfileLoader(),
				resolveStackID: func(*cobra.Command, fctl.Profile) (string, string, error) {
					return "org", "stack", nil
				},
				readStackToken: func(*cobra.Command, string, string, string) (*fctl.AccessToken, error) {
					return token, nil
				},
				newStackClientsFromFlags: func(*cobra.Command, oidcclient.RelyingParty, fctl.Dialog, string, fctl.Profile) (*fctl.StackClients, error) {
					authenticationStarted = true
					return nil, nil
				},
			})
			command := &cobra.Command{}
			command.SetContext(WithNonInteractive(context.Background()))
			command.SetOut(&commandOutput)
			command.SetErr(&commandOutput)

			client, err := factory(command)
			if err == nil {
				t.Fatal("factory error = nil, want unusable-token error")
			}
			if client != nil {
				t.Fatalf("client = %#v, want nil", client)
			}
			if authenticationStarted {
				t.Fatal("interactive authentication boundary was called")
			}
			if terminal.Len() != 0 || commandOutput.Len() != 0 {
				t.Fatalf("completion emitted output: terminal=%q command=%q", terminal.String(), commandOutput.String())
			}
		})
	}
}

func TestClientFactoryNonInteractiveModeUsesSilentDialogWithValidToken(t *testing.T) {
	var terminal bytes.Buffer
	originalLogger := pterm.DefaultLogger
	pterm.DefaultLogger.Writer = &terminal
	t.Cleanup(func() { pterm.DefaultLogger = originalLogger })

	factory := newClientFactory(clientFactoryDependencies{
		loadAndAuthenticateCurrentProfile: testProfileLoader(),
		resolveStackID: func(*cobra.Command, fctl.Profile) (string, string, error) {
			return "org", "stack", nil
		},
		readStackToken: func(*cobra.Command, string, string, string) (*fctl.AccessToken, error) {
			return accessTokenExpiringAt(time.Now().Add(time.Hour)), nil
		},
		newStackClientsFromFlags: func(_ *cobra.Command, _ oidcclient.RelyingParty, dialog fctl.Dialog, _ string, _ fctl.Profile) (*fctl.StackClients, error) {
			dialog.Info("must stay silent")
			return &fctl.StackClients{URI: "https://stack.example", HTTPClient: &http.Client{}}, nil
		},
	})
	command := &cobra.Command{}
	command.SetContext(WithNonInteractive(context.Background()))

	client, err := factory(command)
	if err != nil {
		t.Fatalf("factory error = %v", err)
	}
	if client == nil {
		t.Fatal("client = nil, want Connectivity client")
	}
	if terminal.Len() != 0 {
		t.Fatalf("non-interactive dialog output = %q, want silence", terminal.String())
	}
}

func TestClientFactoryNormalModeRetainsInteractiveDialog(t *testing.T) {
	var terminal bytes.Buffer
	originalLogger := pterm.DefaultLogger
	pterm.DefaultLogger.Writer = &terminal
	t.Cleanup(func() { pterm.DefaultLogger = originalLogger })

	factory := newClientFactory(clientFactoryDependencies{
		loadAndAuthenticateCurrentProfile: testProfileLoader(),
		newStackClientsFromFlags: func(_ *cobra.Command, _ oidcclient.RelyingParty, dialog fctl.Dialog, _ string, _ fctl.Profile) (*fctl.StackClients, error) {
			dialog.Info("normal authentication path")
			return &fctl.StackClients{URI: "https://stack.example", HTTPClient: &http.Client{}}, nil
		},
	})

	client, err := factory(&cobra.Command{})
	if err != nil {
		t.Fatalf("factory error = %v", err)
	}
	if client == nil {
		t.Fatal("client = nil, want Connectivity client")
	}
	if !strings.Contains(terminal.String(), "normal authentication path") {
		t.Fatalf("normal dialog output = %q, want interactive message", terminal.String())
	}
}

func testProfileLoader() func(*cobra.Command) (*fctl.Config, *fctl.Profile, string, oidcclient.RelyingParty, error) {
	return func(*cobra.Command) (*fctl.Config, *fctl.Profile, string, oidcclient.RelyingParty, error) {
		return &fctl.Config{}, &fctl.Profile{}, "profile", nil, nil
	}
}

func accessTokenExpiringAt(expiration time.Time) *fctl.AccessToken {
	return &fctl.AccessToken{TokenWithClaims: fctl.TokenWithClaims[fctl.AccessTokenClaims]{
		Claims: fctl.AccessTokenClaims{TokenClaims: oidc.TokenClaims{Expiration: oidc.Time(expiration.Unix())}},
	}}
}
