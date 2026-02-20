package fctl

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/membershipclient/v3"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/formancehq/go-libs/v3/oidc"
	"github.com/formancehq/go-libs/v3/oidc/client"
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

func EnsureMembershipAccess(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
) (*AccessToken, error) {
	if !profile.IsConnected() {
		return nil, fmt.Errorf("profile %s is not connected, please log in", profileName)
	}
	authenticate := func() (*Tokens, error) {
		return Authenticate(
			cmd.Context(),
			relyingParty,
			dialog,
			[]AuthenticationOption{
				AuthenticateWithScopes(
					oidc.ScopeOpenID,
					oidc.ScopeOfflineAccess,
					"accesses",
					"on_behalf",
				),
				AuthenticateWithPrompt("no-org"),
				AuthenticateWithIDTokenHint(profile.RootTokens.ID.Token),
			},
			[]TokenOption{},
		)
	}

	originalToken := &profile.RootTokens.Access

	token := originalToken
	if token == nil {
		tokens, err := authenticate()
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate for organization: %w", err)
		}

		token = &tokens.Access
	} else if token.Expired() { // todo: define a delta on time
		dialog.Info("Refreshing membership access token...")
		refreshed, err := Refresh(cmd.Context(), relyingParty, *token)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, &oidcErr) {
				return nil, fmt.Errorf("failed to refresh membership token: %w", err)
			}

			if oidcErr.ErrorType != oidc.InvalidToken && oidcErr.ErrorType != oidc.InvalidRequest {
				return nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			dialog.Info("Membership token expired, requesting new authentication...")
			tokens, err := authenticate()
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate for membership: %w", err)
			}

			token = &tokens.Access
		} else {
			token = refreshed
		}
	}

	if token != originalToken {
		profile.RootTokens.Access = *token
		if err := WriteProfile(cmd, profileName, profile); err != nil {
			return nil, err
		}
	}

	return token, nil
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
		dialog.Info("Refreshing organization access token...")
		refreshed, err := Refresh(cmd.Context(), relyingParty, *organizationToken)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, &oidcErr) {
				return nil, fmt.Errorf("failed to refresh organization token: %w", err)
			}

			if oidcErr.ErrorType != oidc.InvalidToken && oidcErr.ErrorType != oidc.InvalidRequest {
				return nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			dialog.Info("Organization token expired, requesting new authentication...")
			tokens, err := authenticate()
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate for organization: %w", err)
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
			"please log in again and/or check you still have access to the organization or that the stack exists or is not deleted", stackID, organizationID)
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
		dialog.Info("Refreshing stack access token...")
		refreshed, err := Refresh(cmd.Context(), relyingParty, *stackToken)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, &oidcErr) {
				return nil, nil, fmt.Errorf("failed to refresh stack token: %w", err)
			}

			// Handle invalid_token, invalid_request (refresh_token not found), and other refresh errors
			// by requesting a new authentication with current profile credentials
			if oidcErr.ErrorType != oidc.InvalidToken && oidcErr.ErrorType != oidc.InvalidRequest {
				return nil, nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			// Refresh token is invalid/expired/not found, reauthenticate with current profile
			dialog.Info("Stack refresh token expired, requesting new authentication...")
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
		dialog.Info("Refreshing app access token...")
		refreshed, err := Refresh(cmd.Context(), relyingParty, *appToken)
		if err != nil {
			oidcErr := &oidc.Error{}
			if !errors.As(err, &oidcErr) {
				return nil, fmt.Errorf("failed to refresh app token: %w", err)
			}

			if oidcErr.ErrorType != oidc.InvalidToken && oidcErr.ErrorType != oidc.InvalidRequest {
				return nil, fmt.Errorf("received unexpected oauth2 error while refreshing token: %w", err)
			}

			dialog.Info("App token expired, requesting new authentication...")
			tokens, err := authenticate()
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate for app: %w", err)
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

func NewMembershipClient(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
) (*membershipclient.SDK, error) {

	accessToken, err := EnsureMembershipAccess(
		cmd,
		relyingParty,
		dialog,
		profileName,
		profile,
	)
	if err != nil {
		return nil, err
	}

	return membershipclient.New(
		membershipclient.WithServerURL(profile.GetMembershipURI()),
		membershipclient.WithClient(relyingParty.HttpClient()),
		membershipclient.WithSecurity(fmt.Sprintf("Bearer %s", accessToken.Token)),
	), nil
}

func NewMembershipClientForOrganization(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
	organizationID string,
) (*membershipclient.SDK, error) {

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

	return membershipclient.New(
		membershipclient.WithServerURL(profile.GetMembershipURI()),
		membershipclient.WithClient(relyingParty.HttpClient()),
		membershipclient.WithSecurity(fmt.Sprintf("Bearer %s", organizationToken.Token)),
	), nil
}

func NewMembershipClientForOrganizationFromFlags(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
) (string, *membershipclient.SDK, error) {

	organizationID, err := ResolveOrganizationID(cmd, profile)
	if err != nil {
		return "", nil, err
	}

	client, err := NewMembershipClientForOrganization(cmd, relyingParty, dialog, profileName, profile, organizationID)

	return organizationID, client, err
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

	return formance.New(
		formance.WithServerURL(stackAccess.URI),
		formance.WithClient(oauth2.NewClient(
			context.WithValue(cmd.Context(), oauth2.HTTPClient, relyingParty.HttpClient()),
			NewStackTokenSource(
				*stackToken,
				stackAccess,
				relyingParty,
				func(newToken AccessToken) error {
					return WriteStackToken(cmd, profileName, stackID, newToken)
				},
				cmd,
				profileName,
				organizationID,
				stackID,
			),
		)),
	), nil
}

func NewStackClientFromFlags(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
) (*formance.Formance, error) {

	organizationID, stackID, err := ResolveStackID(cmd, profile)
	if err != nil {
		return nil, err
	}

	return NewStackClient(cmd, relyingParty, dialog, profileName, profile, organizationID, stackID)
}

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
		[]string{
			"apps:Read",
			"apps:Write",
		},
	)
	if err != nil {
		return nil, err
	}

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

// todo: deploy use membership token, we have to rely on membership applications
func NewAppDeployClientFromFlags(
	cmd *cobra.Command,
	relyingParty client.RelyingParty,
	dialog Dialog,
	profileName string,
	profile Profile,
) (string, *deployserverclient.DeployServer, error) {

	organizationID, err := ResolveOrganizationID(cmd, profile)
	if err != nil {
		return "", nil, err
	}

	deployClient, err := NewAppDeployClient(cmd, relyingParty, dialog, profileName, profile, organizationID)
	if err != nil {
		return "", nil, err
	}

	return organizationID, deployClient, nil
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

	// Cache fields
	cmd            *cobra.Command
	profileName    string
	organizationID string
	stackID        string
}

func (t *stackTokenSource) Token() (*oauth2.Token, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.accessToken == nil || t.accessToken.Expiry.Before(time.Now()) {
		// Try to load from disk cache
		if t.cmd != nil {
			cached, err := ReadCachedStackAPIToken(t.cmd, t.profileName, t.organizationID, t.stackID)
			if err == nil && cached != nil && cached.Expiry.After(time.Now().Add(5*time.Second)) {
				t.accessToken = &oauth2.Token{
					AccessToken: cached.AccessToken,
					TokenType:   cached.TokenType,
					Expiry:      cached.Expiry,
				}
				return t.accessToken, nil
			}
		}

		if t.stackToken.Expired() {
			newStackToken, err := Refresh(context.Background(), t.relyingParty, t.stackToken)
			if err != nil {
				oidcErr := &oidc.Error{}
				if errors.As(err, &oidcErr) && (oidcErr.ErrorType == oidc.InvalidToken || oidcErr.ErrorType == oidc.InvalidRequest) {
					return nil, newErrInvalidAuthentication(fmt.Errorf("stack refresh token expired or not found, please run a command to re-authenticate"))
				}
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

		// Write to disk cache (best-effort)
		if t.cmd != nil {
			_ = WriteCachedStackAPIToken(t.cmd, t.profileName, t.organizationID, t.stackID, CachedStackAPIToken{
				AccessToken: token.AccessToken,
				TokenType:   token.TokenType,
				Expiry:      token.Expiry,
			})
		}
	}

	return t.accessToken, nil
}

var _ oauth2.TokenSource = &stackTokenSource{}

func NewStackTokenSource(
	stackToken AccessToken,
	stackAccess *StackAccess,
	relyingParty client.RelyingParty,
	onRefresh func(newToken AccessToken) error,
	cmd *cobra.Command,
	profileName string,
	organizationID string,
	stackID string,
) oauth2.TokenSource {
	return &stackTokenSource{
		stackToken:     stackToken,
		stackAccess:    stackAccess,
		relyingParty:   relyingParty,
		onRefresh:      onRefresh,
		cmd:            cmd,
		profileName:    profileName,
		organizationID: organizationID,
		stackID:        stackID,
	}
}

func MembershipServerInfo(ctx context.Context, apiClient *membershipclient.SDK) (*components.ServerInfo, error) {
	response, err := apiClient.GetServerInfo(ctx)
	if err != nil {
		return nil, err
	}

	if response.ServerInfo == nil {
		return nil, fmt.Errorf("unexpected response: no server info")
	}

	return response.ServerInfo, nil
}
