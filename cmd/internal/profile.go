package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/zitadel/oidc/pkg/client"
	"github.com/zitadel/oidc/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/pkg/http"
	"github.com/zitadel/oidc/pkg/oidc"
	"golang.org/x/oauth2"
)

const AuthClient = "fctl"

type persistedProfile struct {
	MembershipURI  string        `json:"membershipURI"`
	BaseServiceURI string        `json:"baseServiceURI"`
	Token          *oauth2.Token `json:"token"`
}

type Profile struct {
	membershipURI  string
	baseServiceURI string
	token          *oauth2.Token

	config *Config
}

func (p *Profile) ServicesBaseUrl(organization, stack string) *url.URL {
	baseUrl, err := url.Parse(p.baseServiceURI)
	if err != nil {
		panic(err)
	}
	baseUrl.Host = fmt.Sprintf("%s-%s.%s", organization, stack, baseUrl.Host)
	return baseUrl
}

func (p *Profile) ApiUrl(organization, stack, service string) *url.URL {
	url := p.ServicesBaseUrl(organization, stack)
	url.Path = "/api/" + service
	return url
}

func (p *Profile) UpdateToken(token *oauth2.Token) {
	p.token = token
	p.token.Expiry = p.token.Expiry.UTC()
}

func (p *Profile) MarshalJSON() ([]byte, error) {
	return json.Marshal(persistedProfile{
		MembershipURI:  p.membershipURI,
		BaseServiceURI: p.baseServiceURI,
		Token:          p.token,
	})
}

func (p *Profile) UnmarshalJSON(data []byte) error {
	cfg := &persistedProfile{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	*p = Profile{
		membershipURI:  cfg.MembershipURI,
		baseServiceURI: cfg.BaseServiceURI,
		token:          cfg.Token,
	}
	return nil
}

func (p *Profile) GetMembershipURI() string {
	return p.membershipURI
}

func (p *Profile) GetBaseServiceURI() string {
	return p.baseServiceURI
}

func (p *Profile) GetToken(ctx context.Context, httpClient *http.Client) (*oauth2.Token, error) {
	if p.token == nil {
		return nil, errors.New("not authenticated")
	}
	if p.token != nil && p.token.Expiry.Before(time.Now()) {
		relyingParty, err := rp.NewRelyingPartyOIDC(p.membershipURI, AuthClient, "",
			"", []string{"openid", "email", "offline_access", "supertoken"}, rp.WithHTTPClient(httpClient))
		if err != nil {
			return nil, err
		}

		newToken, err := relyingParty.
			OAuthConfig().
			TokenSource(context.WithValue(ctx, oauth2.HTTPClient, httpClient), p.token).
			Token()
		if err != nil {
			return nil, err
		}

		p.UpdateToken(newToken)
		if err := p.config.Persist(); err != nil {
			return nil, err
		}
	}
	return p.token, nil
}

func (p *Profile) GetUserInfo(ctx context.Context, relyingParty rp.RelyingParty) (oidc.UserInfo, error) {

	req, err := http.NewRequest(http.MethodGet, relyingParty.UserinfoEndpoint(), nil)
	if err != nil {
		return nil, err
	}

	token, err := p.GetToken(ctx, relyingParty.HttpClient())
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	userinfo := oidc.NewUserInfo()
	if err := httphelper.HttpRequest(relyingParty.HttpClient(), req, &userinfo); err != nil {
		return nil, err
	}
	return userinfo, nil
}

func (p *Profile) GetStackToken(ctx context.Context, httpClient *http.Client, organizationID, stackID string) (string, error) {

	form := url.Values{
		"grant_type":         []string{string(oidc.GrantTypeTokenExchange)},
		"audience":           []string{fmt.Sprintf("stack://%s/%s", organizationID, stackID)},
		"subject_token":      []string{p.token.AccessToken},
		"subject_token_type": []string{"urn:ietf:params:oauth:token-type:access_token"},
	}

	membershipDiscoveryConfiguration, err := client.Discover(p.membershipURI, httpClient)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, membershipDiscoveryConfiguration.TokenEndpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(AuthClient, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ret, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if ret.StatusCode != http.StatusOK {
		data, err := io.ReadAll(ret.Body)
		if err != nil {
			panic(err)
		}
		return "", errors.New(string(data))
	}

	securityToken := oauth2.Token{}
	if err := json.NewDecoder(ret.Body).Decode(&securityToken); err != nil {
		return "", err
	}

	apiUrl := p.ApiUrl(organizationID, stackID, "auth")
	form = url.Values{
		"grant_type": []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  []string{securityToken.AccessToken},
		"scope":      []string{"openid email"},
	}

	stackDiscoveryConfiguration, err := client.Discover(apiUrl.String(), httpClient)
	if err != nil {
		return "", err
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, stackDiscoveryConfiguration.TokenEndpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("fctl", "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ret, err = httpClient.Do(req)
	if err != nil {
		return "", err
	}

	if ret.StatusCode != http.StatusOK {
		data, err := io.ReadAll(ret.Body)
		if err != nil {
			panic(err)
		}
		return "", errors.New(string(data))
	}

	stackToken := oauth2.Token{}
	if err := json.NewDecoder(ret.Body).Decode(&stackToken); err != nil {
		return "", err
	}

	return stackToken.AccessToken, nil
}

type CurrentProfile Profile
