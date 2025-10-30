package fctl

import (
	"context"
	"fmt"
	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/membershipclient"
	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/formancehq/go-libs/v3/oidc"
	"github.com/formancehq/go-libs/v3/oidc/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"slices"
	"strings"
	"sync"
	"time"
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

func EnsureOrganizationAccess(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID string,
) (*AccessToken, error) {
	if !profile.RootTokens.ID.Claims.HasOrganizationAccess(organizationID) {
		return nil, fmt.Errorf("no access to organization %s found in your authentication profile, "+
			"please log in again and/or check you still have access to the organization", organizationID)
	}

	authenticate := func() (*Tokens, error) {
		return Authenticate(
			cmd.Context(),
			relyingParty,
			dialog,
			[]AuthenticationOption{
				AuthenticateWithOrganizationID(organizationID),
				AuthenticateWithIDTokenHint(profile.RootTokens.ID.Token),
				AuthenticateWithScopes(append(OrganizationScopes, oidc.ScopeOpenID, oidc.ScopeOfflineAccess)...),
			},
			[]TokenOption{},
		)
	}

	originalOrganizationToken, err := ReadOrganizationToken(cmd, profileName, organizationID)
	if err != nil {
		return nil, err
	}
	organizationToken := originalOrganizationToken
	if organizationToken == nil {
		tokens, err := authenticate()
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate for organization: %w", err)
		}

		organizationToken = &tokens.Access
	} else if organizationToken.Expired() { // todo: define a delta on time
		refreshed, err := Refresh(cmd.Context(), relyingParty, *organizationToken)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, oidcErr) {
				return nil, fmt.Errorf("failed to refresh stack token: %w", err)
			}

			if oidcErr.ErrorType != oidc.InvalidToken {
				return nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			tokens, err := authenticate()
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate for stack: %w", err)
			}

			organizationToken = &tokens.Access
		} else {
			organizationToken = refreshed
		}
	}

	if organizationToken != originalOrganizationToken {
		if err := WriteOrganizationToken(cmd, profileName, *organizationToken); err != nil {
			return nil, err
		}
	}

	return organizationToken, nil
}

func EnsureStackAccess(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID, stackID string,
) (*AccessToken, *StackAccess, error) {
	if !profile.RootTokens.ID.Claims.HasOrganizationAccess(organizationID) {
		return nil, nil, fmt.Errorf("no access to organization %s found in your authentication profile, "+
			"please log in again and/or check you still have access to the organization", organizationID)
	}

	if !profile.RootTokens.ID.Claims.HasStackAccess(organizationID, stackID) {
		return nil, nil, fmt.Errorf("no access to stack %s on organization %s found in your authentication profile, "+
			"please log in again and/or check you still have access to the organization", stackID, organizationID)
	}

	stackAccess := profile.RootTokens.ID.Claims.
		GetOrganizationAccess(organizationID).
		GetStackAccess(stackID)
	stackScopes := collectionutils.Filter(stackAccess.Scopes, func(s string) bool {
		return slices.Contains(StackScopes, s)
	})
	if len(stackScopes) == 0 {
		return nil, nil, fmt.Errorf("no access to stack %s on organization %s found in your authentication profile, "+
			"please log in again and/or check you still have access to the organization", stackID, organizationID)
	}
	resource := "stack://" + organizationID + "/" + stackID + "|" + strings.Join(stackScopes, " ")

	authenticate := func() (*Tokens, error) {
		return Authenticate(
			cmd.Context(),
			relyingParty,
			dialog,
			[]AuthenticationOption{
				AuthenticateWithOrganizationID(organizationID),
				AuthenticateWithIDTokenHint(profile.RootTokens.ID.Token),
				AuthenticateWithResource(resource),
				AuthenticateWithScopes(oidc.ScopeOpenID, oidc.ScopeOfflineAccess),
			},
			[]TokenOption{
				RequestResource(resource),
			},
		)
	}

	originalToken, err := ReadStackToken(cmd, profileName, organizationID, stackID)
	if err != nil {
		return nil, nil, err
	}

	stackToken := originalToken
	if stackToken == nil {
		tokens, err := authenticate()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to authenticate for stack: %w", err)
		}

		stackToken = &tokens.Access
	} else if stackToken.Expired() {
		refreshed, err := Refresh(cmd.Context(), relyingParty, *stackToken)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, &oidcErr) {
				return nil, nil, fmt.Errorf("failed to refresh stack token: %w", err)
			}

			if oidcErr.ErrorType != oidc.InvalidToken {
				return nil, nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			tokens, err := authenticate()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to authenticate for stack: %w", err)
			}

			stackToken = &tokens.Access
		} else {
			stackToken = refreshed
		}
	}

	if stackToken != originalToken {
		if err := WriteStackToken(cmd, profileName, stackID, *stackToken); err != nil {
			return nil, nil, err
		}
	}

	return stackToken, stackAccess, nil
}

func EnsureAppAccess(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID string,
	appAlias string,
	appScopes []string,
) (*AccessToken, error) {
	if !profile.RootTokens.ID.Claims.HasApplicationsAccess(organizationID, appAlias) {
		return nil, fmt.Errorf("no access to application '%s' on organization %s found in your authentication profile, "+
			"please log in again and/or check you still have access to the organization", appAlias, organizationID)
	}

	resource := "app://" + appAlias + "|" + strings.Join(appScopes, " ")

	authenticate := func() (*Tokens, error) {
		return Authenticate(
			cmd.Context(),
			relyingParty,
			dialog,
			[]AuthenticationOption{
				AuthenticateWithOrganizationID(organizationID),
				AuthenticateWithIDTokenHint(profile.RootTokens.ID.Token),
				AuthenticateWithResource(resource),
				AuthenticateWithScopes(oidc.ScopeOpenID, oidc.ScopeOfflineAccess),
			},
			[]TokenOption{
				RequestResource(resource),
			},
		)
	}

	originalAppToken, err := ReadAppToken(cmd, profileName, organizationID, appAlias)
	if err != nil {
		return nil, err
	}

	appToken := originalAppToken
	if appToken == nil {
		tokens, err := authenticate()
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate for organization: %w", err)
		}

		appToken = &tokens.Access
	} else if appToken.Expired() { // todo: define a delta on time
		refreshed, err := Refresh(cmd.Context(), relyingParty, *appToken)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, oidcErr) {
				return nil, fmt.Errorf("failed to refresh app token: %w", err)
			}

			if oidcErr.ErrorType != oidc.InvalidToken {
				return nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			tokens, err := authenticate()
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate for stack: %w", err)
			}

			appToken = &tokens.Access
		} else {
			appToken = refreshed
		}
	}

	if appToken != originalAppToken {
		if err := WriteAppToken(cmd, profileName, appAlias, *appToken); err != nil {
			return nil, err
		}
	}

	return appToken, nil
}

func NewMembershipClientForOrganization(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID string,
) (*membershipclient.APIClient, error) {

	organizationToken, err := EnsureOrganizationAccess(
		cmd,
		relyingParty,
		dialog,
		profileName,
		profile,
		organizationID,
	)
	if err != nil {
		return nil, err
	}

	configuration := membershipclient.NewConfiguration()
	configuration.DefaultHeader = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", organizationToken.Token),
	}
	configuration.UserAgent = "fctl/" + getVersion(cmd)
	configuration.Servers[0].URL = profile.GetMembershipURI()

	return membershipclient.NewAPIClient(configuration), nil
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

func NewStackClient(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID, stackID string,
) (*formance.Formance, error) {

	stackToken, stackAccess, err := EnsureStackAccess(
		cmd,
		relyingParty,
		dialog,
		profileName,
		profile,
		organizationID,
		stackID,
	)
	if err != nil {
		return nil, err
	}

	token, err := FetchStackToken(cmd.Context(), relyingParty.HttpClient(), stackAccess.URI, stackToken.Token)
	if err != nil {
		return nil, err
	}

	return formance.New(
		formance.WithServerURL(stackAccess.URI),
		formance.WithClient(oauth2.NewClient(
			context.WithValue(cmd.Context(), oauth2.HTTPClient, relyingParty.HttpClient()),
			oauth2.StaticTokenSource(token),
		)),
	), nil
}

// todo: deploy use membership token, we have to rely on membership applications
func NewAppDeployClient(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID string,
) (*deployserverclient.DeployServer, error) {

	appToken, err := EnsureAppAccess(
		cmd,
		relyingParty,
		dialog,
		profileName,
		profile,
		organizationID,
		"deploy",
		[]string{},
	)
	if err != nil {
		return nil, err
	}

	configuration := membershipclient.NewConfiguration()
	configuration.DefaultHeader = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", appToken.Token),
	}
	configuration.UserAgent = "fctl/" + getVersion(cmd)
	configuration.Servers[0].URL = profile.GetMembershipURI()

	return deployserverclient.New(
		deployserverclient.WithServerURL(collectionutils.Filter(appToken.Claims.Audience, func(audience string) bool {
			return audience != AuthClient
		})[0]),
		deployserverclient.WithClient(oauth2.NewClient(
			context.WithValue(cmd.Context(), oauth2.HTTPClient, relyingParty.HttpClient()),
			oauth2.StaticTokenSource(appToken.ToOAuth2()),
		)),
	), nil
}

type stackTokenSource struct {
	mu sync.Mutex

	// Membership token
	stackToken AccessToken

	// Token obtained from stack auth server
	accessToken *oauth2.Token

	stackAccess  *StackAccess
	relyingParty client.RelyingParty
	onRefresh    func(newToken AccessToken) error
}

func (t *stackTokenSource) Token() (*oauth2.Token, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.accessToken == nil || t.accessToken.Expiry.Before(time.Now()) {
		if t.stackToken.Expired() {
			newStackToken, err := Refresh(context.Background(), t.relyingParty, t.stackToken)
			if err != nil {
				return nil, err
			}
			t.stackToken = *newStackToken
			if err := t.onRefresh(*newStackToken); err != nil {
				return nil, err
			}
		}

		token, err := FetchStackToken(context.Background(), t.relyingParty.HttpClient(), t.stackAccess.URI, t.stackToken.Token)
		if err != nil {
			return nil, err
		}

		t.accessToken = token
	}

	return t.accessToken, nil
}

var _ oauth2.TokenSource = &stackTokenSource{}

func NewStackTokenSource(
	stackToken AccessToken,
	stackAccess *StackAccess,
	relyingParty client.RelyingParty,
	onRefresh func(newToken AccessToken) error,
) oauth2.TokenSource {
	return &stackTokenSource{
		stackToken:   stackToken,
		stackAccess:  stackAccess,
		relyingParty: relyingParty,
		onRefresh:    onRefresh,
	}
}
