package fctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/go-libs/collectionutils"
	"github.com/formancehq/go-libs/v3/oidc"
	"github.com/formancehq/go-libs/v3/oidc/client"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	OrganizationScopes = []string{
		"organization:Read",
		"organization:Create",
		"organization:Update",
		"organization:Delete",
		"organization:ListUsers",
		"organization:ReadUser",
		"organization:CreateUser",
		"organization:UpdateUser",
		"organization:DeleteUser",
		"organization:ListPolicies",
		"organization:ReadPolicy",
		"organization:CreatePolicy",
		"organization:UpdatePolicy",
		"organization:DeletePolicy",
		"organization:ListInvitations",
		"organization:ReadInvitation",
		"organization:CreateInvitation",
		"organization:UpdateInvitation",
		"organization:AcceptInvitation",
		"organization:RejectInvitation",
		"organization:DeleteInvitation",
		"organization:ListRegions",
		"organization:ReadRegion",
		"organization:CreateRegion",
		"organization:UpdateRegion",
		"organization:DeleteRegion",
		"organization:ListStacks",
		"organization:ReadStack",
		"organization:CreateStack",
		"organization:UpdateStack",
		"organization:DeleteStack",
		"organization:EnableStack",
		"organization:DisableStack",
		"organization:RestoreStack",
		"organization:UpgradeStack",
		"organization:ListStackUsers",
		"organization:ReadStackUser",
		"organization:CreateStackUser",
		"organization:UpdateStackUser",
		"organization:DeleteStackUser",
		"organization:ListStackModules",
		"organization:EnableStackModule",
		"organization:DisableStackModule",
		"organization:ListClients",
		"organization:ReadClient",
		"organization:CreateClient",
		"organization:UpdateClient",
		"organization:DeleteClient",
		"organization:ReadAuthProvider",
		"organization:UpdateAuthProvider",
		"organization:DeleteAuthProvider",
		"organization:ReadLogs",
		"organization:ListFeatures",
		"organization:ReadFeature",
		"organization:UpdateFeatures",
	}
	StackScopes = []string{
		"stack:Read",
		"stack:Write",
	}
)

type AuthenticationOption func(url.Values)

func AuthenticateWithScopes(scopes ...string) AuthenticationOption {
	return func(values url.Values) {
		values.Set("scope", strings.Join(scopes, " "))
	}
}

func AuthenticateWithIDTokenHint(idToken string) AuthenticationOption {
	return func(values url.Values) {
		values.Set("id_token_hint", idToken)
	}
}

func AuthenticateWithPrompt(prompt ...string) AuthenticationOption {
	return func(values url.Values) {
		values.Set("prompt", strings.Join(prompt, " "))
	}
}

func AuthenticateWithOrganizationID(organization string) AuthenticationOption {
	return func(values url.Values) {
		values.Set("organization_id", organization)
	}
}

func AuthenticateWithResource(resource string) AuthenticationOption {
	return func(values url.Values) {
		values.Set("resource", resource)
	}
}

type TokenOption func(url.Values)

func RequestResource(resource string) TokenOption {
	return func(values url.Values) {
		values.Set("resource", resource)
	}
}

func initiateDeviceAuthorizationFlow(relyingParty client.RelyingParty, options ...AuthenticationOption) (*oidc.DeviceAuthorizationResponse, error) {
	form := url.Values{}
	for _, option := range options {
		option(form)
	}

	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest(http.MethodPost, relyingParty.GetDeviceAuthorizationEndpoint(), body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(AuthClient, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := relyingParty.HttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	oidcResponse := oidc.DeviceAuthorizationResponse{}
	if err := json.NewDecoder(httpResponse.Body).Decode(&oidcResponse); err != nil {
		return nil, err
	}

	return &oidcResponse, nil
}

func Authenticate(
	ctx context.Context,
	relyingParty client.RelyingParty,
	dialog Dialog,
	authenticationOptions []AuthenticationOption,
	tokenOptions []TokenOption,
) (*Tokens, error) {

	deviceCode, err := initiateDeviceAuthorizationFlow(relyingParty, authenticationOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device authorization flow: %w", err)
	}

	uri, err := url.Parse(deviceCode.VerificationURI)
	if err != nil {
		panic(err)
	}
	query := uri.Query()
	query.Set("user_code", deviceCode.UserCode)
	uri.RawQuery = query.Encode()

	if err := OpenURL(uri.String()); err != nil {
		if !errors.Is(err, ErrOpeningBrowser) {
			return nil, err
		}

		dialog.Info("No browser detected")
		dialog.Info("Please open the following URL in your browser:" + uri.String())
	} else {
		dialog.Info("A browser window has been opened on " + uri.String())
	}
	dialog.Info("Waiting for authentication...")

	rsp, err := client.DeviceAccessToken[*IDTokenClaims](
		ctx,
		deviceCode.DeviceCode,
		time.Duration(deviceCode.Interval)*time.Second,
		relyingParty,
		collectionutils.Map(tokenOptions, func(option TokenOption) func(url.Values) {
			return option
		})...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	idTokenClaims := &IDTokenClaims{}
	if _, err := oidc.ParseToken(rsp.IDToken, idTokenClaims); err != nil {
		return nil, fmt.Errorf("failed to parse id token: %w", err)
	}

	accessTokenClaims := AccessTokenClaims{}
	if _, err := oidc.ParseToken(rsp.AccessToken, &accessTokenClaims); err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	return &Tokens{
		Access: AccessToken{
			TokenWithClaims: TokenWithClaims[AccessTokenClaims]{
				Token:  rsp.AccessToken,
				Claims: accessTokenClaims,
			},
			Refresh: rsp.RefreshToken,
		},
		ID: IDToken{
			Token:  rsp.IDToken,
			Claims: *idTokenClaims,
		},
	}, nil
}

func Refresh(ctx context.Context, relyingParty client.RelyingParty, token AccessToken) (*AccessToken, error) {
	newToken, err := client.RefreshTokens[*IDTokenClaims](ctx, relyingParty, token.Refresh, "", "")
	if err != nil {
		return nil, newErrInvalidAuthentication(err)
	}

	claims := AccessTokenClaims{}
	_, err = oidc.ParseToken(newToken.AccessToken, &claims)
	if err != nil {
		return nil, newErrInvalidAuthentication(err)
	}

	token.Token = newToken.AccessToken
	token.Refresh = newToken.RefreshToken
	token.Claims = claims

	return &AccessToken{
		TokenWithClaims: TokenWithClaims[AccessTokenClaims]{
			Token:  newToken.AccessToken,
			Claims: claims,
		},
		Refresh: newToken.RefreshToken,
	}, nil
}

func FetchStackToken(ctx context.Context, httpClient *http.Client, stackURI, token string) (*oauth2.Token, error) {

	form := url.Values{
		"grant_type": []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  []string{token},
		"scope":      []string{strings.Join([]string{oidc.ScopeOpenID, oidc.ScopeEmail}, " ")},
	}

	stackDiscoveryConfiguration, err := client.Discover[oidc.DiscoveryConfiguration](ctx, stackURI+"/api/auth", httpClient)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stackDiscoveryConfiguration.TokenEndpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	now := time.Now()
	ret, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch ret.StatusCode {
	case http.StatusUnauthorized:
		return nil, newErrUnauthorized()
	case http.StatusForbidden:
		return nil, newErrForbidden()
	case http.StatusOK:
		stackToken := &oauth2.Token{}
		if err := json.NewDecoder(ret.Body).Decode(stackToken); err != nil {
			return nil, err
		}

		if stackToken.Expiry.IsZero() {
			stackToken.Expiry = now.Add(time.Duration(stackToken.ExpiresIn) * time.Second).Time
		}

		return stackToken, nil
	default:
		return nil, newUnexpectedStatusCodeError(ret.StatusCode)
	}
}

func UserInfo(cmd *cobra.Command, relyingParty client.RelyingParty, token AccessToken) (*UserClaims, error) {
	ui, err := client.Userinfo[*oidc.UserInfo](cmd.Context(), token.Token, "Bearer", relyingParty)
	if err != nil {
		return nil, err
	}

	claims := &UserClaims{}
	claims.Email = ui.Email
	claims.Subject = ui.Subject

	return claims, nil
}

type TokenWithClaims[T any] struct {
	Token  string `json:"token"`
	Claims T      `json:"claims"`
}

type AccessTokenClaims struct {
	oidc.TokenClaims
	Scopes         oidc.SpaceDelimitedArray `json:"scope,omitempty"`
	OrganizationID string                   `json:"organization_id"`
}

type AccessToken struct {
	TokenWithClaims[AccessTokenClaims]
	Refresh string `json:"refreshToken"`
}

func (t AccessToken) ToOAuth2() *oauth2.Token {
	return &oauth2.Token{
		AccessToken: t.Token,
		TokenType:   "Bearer",
		Expiry:      t.Claims.Expiration.AsTime().Time,
	}
}

func (t AccessToken) Expired() bool { // todo: define a delta on time
	return t.Claims.Expiration.AsTime().Before(time.Now())
}

type IDToken = TokenWithClaims[IDTokenClaims]

type Tokens struct {
	Access AccessToken `json:"accessToken"`
	ID     IDToken     `json:"idToken"`
}

type IDTokenClaims struct {
	oidc.TokenClaims
	NotBefore       oidc.Time `json:"nbf,omitempty"`
	AccessTokenHash string    `json:"at_hash,omitempty"`
	CodeHash        string    `json:"c_hash,omitempty"`
	SessionID       string    `json:"sid,omitempty"`
	oidc.UserInfoProfile
	oidc.UserInfoEmail
	oidc.UserInfoPhone
	Address       *oidc.UserInfoAddress `json:"address,omitempty"`
	Organizations []OrganizationAccess  `json:"org"`
}

func (i IDTokenClaims) GetExpiration() time.Time {
	return i.Expiration.AsTime()
}

func (i IDTokenClaims) GetIssuedAt() time.Time {
	return i.IssuedAt.AsTime()
}

func (i IDTokenClaims) GetAuthTime() time.Time {
	return i.AuthTime.AsTime()
}

func (i IDTokenClaims) GetAccessTokenHash() string {
	return i.AccessTokenHash
}

func (i IDTokenClaims) GetOrganizationAccess(id string) *OrganizationAccess {
	for _, organization := range i.Organizations {
		if organization.ID == id {
			return &organization
		}
	}
	return nil
}

func (i IDTokenClaims) HasOrganizationAccess(id string) bool {
	return i.GetOrganizationAccess(id) != nil
}

func (i IDTokenClaims) HasApplicationsAccess(organizationID string, alias string) bool {
	organizationAccess := i.GetOrganizationAccess(organizationID)
	if organizationAccess == nil {
		return false
	}
	for _, application := range organizationAccess.Applications {
		if application.Alias == alias {
			return true
		}
	}
	return false
}

func (i IDTokenClaims) HasStackAccess(organizationID string, stackID string) bool {
	organizationAccess := i.GetOrganizationAccess(organizationID)
	if organizationAccess == nil {
		return false
	}
	return organizationAccess.GetStackAccess(stackID) != nil
}

type StackAccess struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	URI         string   `json:"uri"`
	Scopes      []string `json:"scopes"`
}
type OrganizationAccess struct {
	ID           string              `json:"id"`
	DisplayName  string              `json:"displayName"`
	Stacks       []StackAccess       `json:"stacks"`
	Applications []ApplicationAccess `json:"applications"`
}

func (o *OrganizationAccess) GetStackAccess(stackID string) *StackAccess {
	if o == nil {
		return nil
	}
	for _, stack := range o.Stacks {
		if stack.ID == stackID {
			return &stack
		}
	}
	return nil
}

type OrganizationsClaim []OrganizationAccess
type UserClaims struct {
	Email   string             `json:"email"`
	Subject string             `json:"sub"`
	Org     OrganizationsClaim `json:"org"`
}

type ApplicationAccess struct {
	Alias string `json:"alias"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}
