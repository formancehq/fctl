// Package connection resolves service endpoints independently of the Cloud.
package connection

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/browser"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/httpdebug"
	"github.com/formancehq/fctl/v4/internal/presentation"
)

type Settings struct {
	Name      string
	Directory string
	Options   Options
	Timeout   time.Duration
	Output    string
	NoBrowser bool
	Debug     bool
	Color     string
}

func (s *Settings) Bind(root *cobra.Command) {
	f := root.PersistentFlags()
	f.StringVar(&s.Name, "connection", "", "Saved connection name (FCTL_CONNECTION)")
	f.StringVar(&s.Directory, "config-dir", "", "Private v4 configuration directory (FCTL_CONFIG_DIR)")
	f.DurationVar(&s.Timeout, "timeout", 30*time.Second, "Timeout for each HTTP request")
	f.StringVarP(&s.Output, "output", "o", "auto", "Output format: auto, table or json (auto uses tables in a terminal)")
	f.StringVar(&s.Color, "color", "auto", "Color: auto, always or never (NO_COLOR disables automatic color)")
	f.BoolVarP(&s.Debug, "debug", "d", false, "Trace HTTP requests and responses on stderr (credentials redacted)")
	f.BoolVar(&s.NoBrowser, "no-browser", false, "Display login instructions without opening a browser")
	for _, setting := range s.fields() {
		f.StringVar(setting.value, setting.name, "", setting.help)
	}
}

type field struct {
	name  string
	value *string
	help  string
}

func (s *Settings) fields() []field {
	return []field{
		{"stack-url", &s.Options.StackURL, "Gateway base URL"},
		{"ledger-url", &s.Options.LedgerURL, "Standalone Ledger endpoint"},
		{"auth-url", &s.Options.AuthURL, "Standalone Auth endpoint"},
		{"auth-mode", &s.Options.AuthMode, "Authentication: none, client-credentials, cloud"},
		{"token-url", &s.Options.TokenURL, "OAuth2 token endpoint"},
		{"client-id", &s.Options.ClientID, "OAuth2 client ID; defaults to fctl for Cloud"},
		{"scopes", &s.Options.Scopes, "Space-separated OAuth2 scopes"},
		{"issuer", &s.Options.Issuer, "Membership OIDC issuer for Cloud login"},
		{"organization", &s.Options.Organization, "Cloud organization ID"},
		{"stack", &s.Options.Stack, "Cloud stack ID"},
	}
}

func (s *Settings) DirectoryPath() (string, error) {
	if dir := cmp.Or(s.Directory, os.Getenv("FCTL_CONFIG_DIR")); dir != "" {
		return dir, nil
	}
	return DefaultDirectory()
}

func (s *Settings) Resolve(cmd *cobra.Command) (Options, Entry, string, string, error) {
	dir, err := s.DirectoryPath()
	if err != nil {
		return Options{}, Entry{}, "", "", err
	}
	store, err := Load(dir)
	if err != nil {
		return Options{}, Entry{}, "", "", err
	}
	name := cmp.Or(s.Name, os.Getenv("FCTL_CONNECTION"), store.Active)
	entry := store.Connections[name]
	if name != "" {
		if err := ValidateName(name); err != nil {
			return Options{}, Entry{}, "", "", err
		}
		if _, ok := store.Connections[name]; !ok {
			return Options{}, Entry{}, "", "", fmt.Errorf("connection %q does not exist", name)
		}
	}
	resolved := Settings{Options: entry.Options}
	mode := os.Getenv("FCTL_AUTH_MODE")
	if cmd.Root().PersistentFlags().Changed("auth-mode") {
		mode = s.Options.AuthMode
	}
	if mode != "" && mode != entry.Options.AuthMode {
		// An explicit mode switch starts a fresh connection boundary. In
		// particular, local one-off commands must not inherit Cloud identity.
		resolved.Options = Options{}
	}
	saved := resolved.fields()
	for i, setting := range s.fields() {
		if cmd.Root().PersistentFlags().Changed(setting.name) {
			*saved[i].value = *setting.value
		} else if value, ok := os.LookupEnv("FCTL_" + strings.ToUpper(strings.ReplaceAll(setting.name, "-", "_"))); ok {
			*saved[i].value = value
		}
	}
	return resolved.Options, entry, name, dir, nil
}

func (s *Settings) Input(cmd *cobra.Command) Options {
	resolved := Settings{}
	fields := resolved.fields()
	for i, setting := range s.fields() {
		if cmd.Root().PersistentFlags().Changed(setting.name) {
			*fields[i].value = *setting.value
		} else {
			*fields[i].value = os.Getenv("FCTL_" + strings.ToUpper(strings.ReplaceAll(setting.name, "-", "_")))
		}
	}
	return resolved.Options
}

func (s *Settings) Client(ctx context.Context, cmd *cobra.Command, service string) (*api.Client, error) {
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
	client := s.HTTPClient(cmd.ErrOrStderr())
	if service == "cloud" {
		if cloudCommandTarget(cmd) == "organization" {
			return s.cloudOrganizationClient(ctx, cmd, client, options, entry, name, dir)
		}
		return s.membershipClient(ctx, cmd, client, options, entry, name, dir)
	}
	base, err := endpoint(options, service)
	if err != nil {
		return nil, err
	}
	switch options.AuthMode {
	case "none":
	case "client-credentials":
		client, err = credentialClient(ctx, client, options)
	case "cloud":
		options, err = s.selectCloudTarget(ctx, cmd, client, options, entry, name, dir)
		if err != nil {
			return nil, err
		}
		client, base, err = savedCloudClient(ctx, client, options, entry, name, dir, cmd.ErrOrStderr(), s.BrowserOpener())
		if service != "stack" {
			base = strings.TrimRight(base, "/") + "/api/" + service
		}
	default:
		return nil, fmt.Errorf("choose --auth-mode=none, client-credentials or cloud")
	}
	if err != nil {
		return nil, err
	}
	return api.New(base, client)
}

func savedCloudClient(ctx context.Context, client *http.Client, options Options, entry Entry, name, dir string, out io.Writer, open func(context.Context, string) error) (*http.Client, string, error) {
	if cloudIdentity(options) != cloudIdentity(entry.Options) {
		return nil, "", fmt.Errorf("cloud identity settings changed; log in again with the chosen issuer and client")
	}
	if entry.Session == nil {
		return nil, "", fmt.Errorf("connection is not logged in; run fctl login --connection %s", name)
	}
	coordinator := cloudCoordinator(dir, name, entry)
	if entry.Session.Options.Stack != "" {
		if options != entry.Options {
			return nil, "", fmt.Errorf("this older session targets a single stack; run fctl login before changing its target")
		}
		return cloud.Client(ctx, client, entry.Session, nil, coordinator)
	}
	return cloud.ClientForTarget(ctx, client, entry.Session, cloud.Options{Issuer: options.Issuer, ClientID: options.ClientID, Organization: options.Organization, Stack: options.Stack}, out, open, coordinator)
}

func (s *Settings) BrowserOpener() func(context.Context, string) error {
	if s.NoBrowser {
		return nil
	}
	return browser.Open
}

func cloudIdentity(options Options) Options {
	options.Organization, options.Stack = "", ""
	if options.Issuer == "" {
		options.Issuer = cloud.DefaultIssuer
	}
	if options.ClientID == "" {
		options.ClientID = "fctl"
	}
	return options
}

func Validate(o Options) error {
	for _, value := range []string{o.StackURL, o.LedgerURL, o.AuthURL, o.TokenURL, o.Issuer} {
		if value == "" {
			continue
		}
		if err := api.ValidateURL(value); err != nil {
			return err
		}
	}
	switch o.AuthMode {
	case "none":
		if err := validateAnonymous(o); err != nil {
			return err
		}
	case "client-credentials":
		if err := validateCredentials(o); err != nil {
			return err
		}
	case "cloud":
		return validateCloud(o)
	default:
		return fmt.Errorf("choose auth-mode none, client-credentials or cloud")
	}
	if o.AuthMode != "cloud" && o.StackURL == "" && o.LedgerURL == "" && o.AuthURL == "" {
		return fmt.Errorf("configure stack-url, ledger-url or auth-url")
	}
	return nil
}

func validateAnonymous(o Options) error {
	if o.ClientID != "" || o.TokenURL != "" || o.Scopes != "" || o.Issuer != "" || o.Organization != "" || o.Stack != "" {
		return fmt.Errorf("auth-mode=none cannot include OAuth2 or Cloud settings")
	}
	return nil
}

func validateCredentials(o Options) error {
	if o.ClientID == "" || o.TokenURL == "" {
		return fmt.Errorf("client-credentials requires client-id and token-url")
	}
	if o.Organization != "" || o.Stack != "" || o.Issuer != "" {
		return fmt.Errorf("client-credentials cannot include Cloud settings")
	}
	for _, endpoint := range []string{o.TokenURL, o.StackURL, o.LedgerURL, o.AuthURL} {
		if endpoint == "" {
			continue
		}
		if err := api.ValidateSecureURL(endpoint); err != nil {
			return err
		}
	}
	return nil
}

func validateCloud(o Options) error {
	if o.StackURL != "" || o.LedgerURL != "" || o.AuthURL != "" || o.TokenURL != "" || o.Scopes != "" {
		return fmt.Errorf("cloud service endpoints and scopes are supplied by Membership")
	}
	issuer := cmp.Or(o.Issuer, cloud.DefaultIssuer)
	return api.ValidateSecureURL(issuer)
}

func endpoint(o Options, service string) (string, error) {
	if o.AuthMode == "cloud" {
		return "", nil
	}
	if service == "stack" {
		if o.StackURL == "" {
			return "", fmt.Errorf("configure --stack-url for stack utilities")
		}
		return o.StackURL, nil
	}
	var direct string
	switch service {
	case "ledger":
		direct = o.LedgerURL
	case "auth":
		direct = o.AuthURL
	default:
		return "", fmt.Errorf("unsupported service %q", service)
	}
	if direct != "" {
		return direct, nil
	}
	if o.StackURL != "" {
		return strings.TrimRight(o.StackURL, "/") + "/api/" + service, nil
	}
	return "", fmt.Errorf("configure --%s-url or --stack-url", service)
}

func credentialClient(ctx context.Context, base *http.Client, o Options) (*http.Client, error) {
	secret := os.Getenv("FCTL_CLIENT_SECRET")
	if o.ClientID == "" || secret == "" {
		return nil, fmt.Errorf("client-credentials requires --client-id and FCTL_CLIENT_SECRET")
	}
	if o.TokenURL == "" {
		return nil, fmt.Errorf("client-credentials requires --token-url")
	}
	if err := api.ValidateURL(o.TokenURL); err != nil {
		return nil, err
	}
	config := clientcredentials.Config{ClientID: o.ClientID, ClientSecret: secret, TokenURL: o.TokenURL, Scopes: strings.Fields(o.Scopes)}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, base)
	client := oauth2.NewClient(ctx, safeSource{config.TokenSource(ctx)})
	client.Timeout = base.Timeout
	client.CheckRedirect = base.CheckRedirect
	return client, nil
}

type safeSource struct{ source oauth2.TokenSource }

func (s safeSource) Token() (*oauth2.Token, error) {
	token, err := s.source.Token()
	if err != nil {
		return nil, fmt.Errorf("OAuth2 authentication failed; check client credentials and token endpoint")
	}
	return token, nil
}

// HTTPClient supplies the shared transport for services and authentication.
func (s *Settings) HTTPClient(out io.Writer) *http.Client {
	client := &http.Client{Timeout: s.Timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	if s.Debug {
		client.Transport = httpdebug.New(nil, out)
	}
	return client
}
