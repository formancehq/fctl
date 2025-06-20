package fctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/formancehq/go-libs/logging"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/v2/pkg/client"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"golang.org/x/oauth2"
)

type ErrInvalidAuthentication struct {
	err error
}

func (e ErrInvalidAuthentication) Error() string {
	return e.err.Error()
}

func (e ErrInvalidAuthentication) Unwrap() error {
	return e.err
}

func (e ErrInvalidAuthentication) Is(err error) bool {
	_, ok := err.(*ErrInvalidAuthentication)
	return ok
}

func IsInvalidAuthentication(err error) bool {
	return errors.Is(err, &ErrInvalidAuthentication{})
}

func newErrInvalidAuthentication(err error) *ErrInvalidAuthentication {
	return &ErrInvalidAuthentication{
		err: err,
	}
}

const AuthClient = "fctl"

type persistedProfile struct {
	MembershipURI       string                    `json:"membershipURI"`
	Token               *oidc.AccessTokenResponse `json:"token"`
	DefaultOrganization string                    `json:"defaultOrganization"`
	DefaultStack        string                    `json:"defaultStack"`
}

type Profile struct {
	membershipURI string
	token         *oidc.AccessTokenResponse

	defaultOrganization string
	defaultStack        string

	config *Config
}

func (p *Profile) ServicesBaseUrl(stack *membershipclient.Stack) *url.URL {
	baseUrl, err := url.Parse(stack.Uri)
	if err != nil {
		panic(err)
	}
	return baseUrl
}

func (p *Profile) ApiUrl(stack *membershipclient.Stack, service string) *url.URL {
	url := p.ServicesBaseUrl(stack)
	url.Path = "/api/" + service
	return url
}

func (p *Profile) UpdateToken(token *oidc.AccessTokenResponse) {
	p.token = token
}

func (p *Profile) SetMembershipURI(v string) {
	p.membershipURI = v
}

func (p *Profile) MarshalJSON() ([]byte, error) {
	return json.Marshal(persistedProfile{
		MembershipURI:       p.membershipURI,
		Token:               p.token,
		DefaultOrganization: p.defaultOrganization,
		DefaultStack:        p.defaultStack,
	})
}

func (p *Profile) UnmarshalJSON(data []byte) error {
	cfg := &persistedProfile{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	*p = Profile{
		membershipURI:       cfg.MembershipURI,
		token:               cfg.Token,
		defaultOrganization: cfg.DefaultOrganization,
		defaultStack:        cfg.DefaultStack,
	}
	return nil
}

func (p *Profile) GetMembershipURI() string {
	return p.membershipURI
}

func (p *Profile) GetDefaultOrganization() string {
	return p.defaultOrganization
}

func (p *Profile) GetDefaultStack() string {
	return p.defaultStack
}

func (p *Profile) GetToken(ctx context.Context, httpClient *http.Client) (*oauth2.Token, error) {
	logging.FromContext(ctx).Debug("Check token from profile")
	if p.token == nil {
		return nil, errors.New("not authenticated")
	}
	logging.FromContext(ctx).Debug("Has been authenticated")
	if p.token != nil {
		claims := &oidc.AccessTokenClaims{}
		_, err := oidc.ParseToken(p.token.AccessToken, claims)
		if err != nil {
			return nil, newErrInvalidAuthentication(errors.Wrap(err, "parsing token"))
		}
		logging.FromContext(ctx).Debugf("Token has expired ? %s in %s", BoolToString(claims.Expiration.AsTime().Before(time.Now())), time.Since(claims.Expiration.AsTime()).String())
		if claims.Expiration.AsTime().Before(time.Now()) {
			relyingParty, err := GetAuthRelyingParty(httpClient, p.membershipURI)
			if err != nil {
				return nil, err
			}

			newToken, err := rp.RefreshAccessToken(relyingParty, p.token.RefreshToken, "", "")
			if err != nil {
				return nil, newErrInvalidAuthentication(errors.Wrap(err, "refreshing token"))
			}

			p.UpdateToken(&oidc.AccessTokenResponse{
				AccessToken:  newToken.AccessToken,
				TokenType:    newToken.TokenType,
				RefreshToken: newToken.RefreshToken,
				IDToken:      newToken.Extra("id_token").(string),
			})
			if err := p.config.Persist(); err != nil {
				return nil, err
			}
			logging.FromContext(ctx).Debug("Token refreshed and persisted")
		}
	}
	claims := &oidc.AccessTokenClaims{}
	_, err := oidc.ParseToken(p.token.AccessToken, claims)
	if err != nil {
		return nil, newErrInvalidAuthentication(err)
	}
	return &oauth2.Token{
		AccessToken:  p.token.AccessToken,
		TokenType:    p.token.TokenType,
		RefreshToken: p.token.RefreshToken,
		Expiry:       claims.Expiration.AsTime(),
	}, nil
}

func (p *Profile) GetClaims() (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	parser := jwt.Parser{}
	_, _, err := parser.ParseUnverified(p.token.AccessToken, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (p *Profile) GetUserInfo(cmd *cobra.Command) (*userClaims, error) {
	claims := &userClaims{}
	if p.token == nil || p.token.IDToken == "" {
		return nil, errors.New("not authenticated")
	}

	_, err := oidc.ParseToken(p.token.IDToken, claims)
	if err != nil {
		return nil, err
	}

	mbClient, err := NewMembershipClient(cmd, p.config)
	if err != nil {
		return nil, err
	}
	token, err := mbClient.GetProfile().GetToken(cmd.Context(), mbClient.GetConfig().HTTPClient)
	if err != nil {
		return nil, err
	}
	relyingParty, err := GetAuthRelyingParty(mbClient.GetConfig().HTTPClient, p.membershipURI)
	if err != nil {
		return nil, err
	}

	ui, err := rp.Userinfo(token.AccessToken, token.TokenType, claims.Subject, relyingParty)
	if err != nil {
		return nil, err
	}

	claims.Email = ui.Email
	claims.Subject = ui.Subject

	return claims, nil
}

func (p *Profile) GetStackToken(ctx context.Context, httpClient *http.Client, stack *membershipclient.Stack) (*oauth2.Token, error) {

	form := url.Values{
		"grant_type":         []string{string(oidc.GrantTypeTokenExchange)},
		"audience":           []string{fmt.Sprintf("stack://%s/%s", stack.OrganizationId, stack.Id)},
		"subject_token":      []string{p.token.AccessToken},
		"subject_token_type": []string{"urn:ietf:params:oauth:token-type:access_token"},
	}

	membershipDiscoveryConfiguration, err := client.Discover(p.membershipURI, httpClient)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, membershipDiscoveryConfiguration.TokenEndpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(AuthClient, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ret, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = ret.Body.Close()
	}()

	if ret.StatusCode != http.StatusOK {
		data, err := io.ReadAll(ret.Body)
		if err != nil {
			panic(err)
		}
		return nil, errors.New(string(data))
	}

	securityToken := oauth2.Token{}
	if err := json.NewDecoder(ret.Body).Decode(&securityToken); err != nil {
		return nil, err
	}

	apiUrl := p.ApiUrl(stack, "auth")
	form = url.Values{
		"grant_type": []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  []string{securityToken.AccessToken},
		"scope":      []string{"openid email"},
	}

	stackDiscoveryConfiguration, err := client.Discover(apiUrl.String(), httpClient)
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, stackDiscoveryConfiguration.TokenEndpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	now := time.Now()
	ret, err = httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if ret.StatusCode != http.StatusOK {
		data, err := io.ReadAll(ret.Body)
		if err != nil {
			panic(err)
		}
		return nil, errors.New(string(data))
	}

	stackToken := &oauth2.Token{}
	if err := json.NewDecoder(ret.Body).Decode(stackToken); err != nil {
		return nil, err
	}

	if stackToken.Expiry.IsZero() {
		stackToken.Expiry = now.Add(time.Duration(stackToken.ExpiresIn) * time.Second)
	}

	return stackToken, nil
}

func (p *Profile) SetDefaultOrganization(o string) {
	p.defaultOrganization = o
}

func (p *Profile) SetDefaultStack(s string) {
	p.defaultStack = s
}

func (p *Profile) IsConnected() bool {
	return p.token != nil
}

type CurrentProfile Profile

func ListProfiles(cmd *cobra.Command, toComplete string) ([]string, error) {
	config, err := GetConfig(cmd)
	if err != nil {
		return []string{}, nil
	}

	ret := make([]string, 0)
	for p := range config.GetProfiles() {
		if strings.HasPrefix(p, toComplete) {
			ret = append(ret, p)
		}
	}
	sort.Strings(ret)
	return ret, nil
}
