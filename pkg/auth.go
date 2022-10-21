package fctl

import (
	"context"
	"fmt"
	"io"

	"github.com/formancehq/auth/authclient"
)

// TODO: Make configurable at build
const AuthClient = "fctl"

func NewAuthClientFromContext(ctx context.Context) (*authclient.APIClient, error) {
	token, err := CurrentProfileFromContext(ctx).GetStackToken(ctx)
	if err != nil {
		return nil, err
	}

	organization, err := FindOrganizationId(ctx)
	if err != nil {
		return nil, err
	}

	stack, err := FindStackId(ctx, organization)
	if err != nil {
		return nil, err
	}

	profile := CurrentProfileFromContext(ctx)
	config := authclient.NewConfiguration()
	config.Servers = authclient.ServerConfigurations{{
		URL: MustApiUrl(*profile, organization, stack, "auth").String(),
	}}
	config.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	config.HTTPClient = NewHTTPClientFromContext(ctx)

	return authclient.NewAPIClient(config), nil
}

func PrintAuthClient(out io.Writer, c authclient.Client) {
	fmt.Fprintf(out, "ID: %s\r\n", c.Id)
	fmt.Fprintf(out, "Name: %s\r\n", c.Name)
	if c.Public != nil && *c.Public {
		fmt.Fprintf(out, "Public: yes\r\n")
	}
	if c.Trusted != nil && *c.Trusted {
		fmt.Fprintf(out, "Trusted: yes\r\n")
	}
	if len(c.Secrets) > 0 {
		fmt.Fprintf(out, "Secrets: \r\n")
		for _, secret := range c.Secrets {
			fmt.Fprintf(out, "\t -\r\n")
			fmt.Fprintf(out, "\t\tID: %s\r\n", secret.Id)
			fmt.Fprintf(out, "\t\tName: %s\r\n", secret.Name)
			fmt.Fprintf(out, "\t\tLastDigits: %s\r\n", secret.LastDigits)
		}
	}
	if len(c.RedirectUris) > 0 {
		fmt.Fprintf(out, "Redirect URIS: \r\n")
		for _, redirectUri := range c.RedirectUris {
			fmt.Fprintf(out, "\t- %s\r\n", redirectUri)
		}
	}
	if len(c.PostLogoutRedirectUris) > 0 {
		fmt.Fprintf(out, "Post Logout Redirect URIS: \r\n")
		for _, postLogoutRedirectUri := range c.PostLogoutRedirectUris {
			fmt.Fprintf(out, "\t- %s\r\n", postLogoutRedirectUri)
		}
	}
	if len(c.Scopes) > 0 {
		fmt.Fprintf(out, "Scopes: \r\n")
		for _, scope := range c.Scopes {
			fmt.Fprintf(out, "\t- %s\r\n", scope)
		}
	}
}

func PrintAuthClientSecret(out io.Writer, c *authclient.Secret) {
	fmt.Fprintf(out, "Name: %s\r\n", c.Name)
	fmt.Fprintf(out, "ID: %s\r\n", c.Id)
	fmt.Fprintf(out, "Last Digits: %s\r\n", c.LastDigits)
	fmt.Fprintf(out, "Clear: %s\r\n", c.Clear)
}
