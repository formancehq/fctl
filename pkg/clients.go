package fctl

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/go-libs/logging"

	"github.com/formancehq/fctl/membershipclient"
)

func getVersion(cmd *cobra.Command) string {
	for cmd != nil {
		if cmd.Version != "" {
			return cmd.Version
		}
		cmd = cmd.Parent()
	}
	return "cmd.Version"
}

type MembershipClient struct {
	profile *Profile
	*membershipclient.APIClient
}

func (c *MembershipClient) GetProfile() *Profile {
	return c.profile
}

func (c *MembershipClient) RefreshIfNeeded(cmd *cobra.Command) error {
	logging.Debug("Refreshing membership client")
	token, err := c.profile.GetToken(cmd.Context(), c.GetConfig().HTTPClient)
	if err != nil {
		return err
	}
	config := c.GetConfig()
	config.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	c.APIClient = membershipclient.NewAPIClient(config)
	return err
}

func NewMembershipClient(cmd *cobra.Command, cfg *Config) (*MembershipClient, error) {
	profile := GetCurrentProfile(cmd, cfg)
	httpClient := GetHttpClient(cmd, map[string][]string{})
	configuration := membershipclient.NewConfiguration()
	configuration.HTTPClient = httpClient
	configuration.UserAgent = "fctl/" + getVersion(cmd)
	configuration.Servers[0].URL = profile.GetMembershipURI()
	client := &MembershipClient{
		APIClient: membershipclient.NewAPIClient(configuration),
		profile:   profile,
	}
	err := client.RefreshIfNeeded(cmd)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func MembershipServerInfo(ctx context.Context, client *membershipclient.DefaultAPIService) (*membershipclient.ServerInfo, error) {
	serverInfo, response, err := client.GetServerInfo(ctx).Execute()
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	return serverInfo, nil
}

func NewStackClient(cmd *cobra.Command, cfg *Config, stack *membershipclient.Stack) (*formance.Formance, error) {
	return formance.New(
		formance.WithServerURL(stack.Uri),
		formance.WithClient(
			&http.Client{
				Transport: NewStackHTTPTransport(
					cmd,
					GetCurrentProfile(cmd, cfg),
					stack,
					map[string][]string{
						"User-Agent": {"fctl/" + getVersion(cmd)},
					},
				),
			},
		),
	), nil
}

type stackHttpTransport struct {
	profile             *Profile
	authHttpClient      *http.Client
	stack               *membershipclient.Stack
	token               *oauth2.Token
	underlyingTransport http.RoundTripper
}

func (s *stackHttpTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if s.token == nil || time.Now().After(s.token.Expiry.Add(-10*time.Second)) {
		token, err := s.profile.GetStackToken(request.Context(), s.authHttpClient, s.stack)
		if err != nil {
			return nil, err
		}
		s.token = token
	}

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token.AccessToken))

	return s.underlyingTransport.RoundTrip(request)
}

func NewStackHTTPTransport(cmd *cobra.Command, profile *Profile, stack *membershipclient.Stack, defaultHeaders map[string][]string) *stackHttpTransport {
	return &stackHttpTransport{
		underlyingTransport: NewHTTPTransport(cmd, defaultHeaders),
		authHttpClient:      GetHttpClient(cmd, map[string][]string{}),
		profile:             profile,
		stack:               stack,
	}
}

var _ http.RoundTripper = &stackHttpTransport{}
