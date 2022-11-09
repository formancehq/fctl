package fctl

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/zitadel/oidc/pkg/client"
	"github.com/zitadel/oidc/pkg/client/rp"
	"golang.org/x/oauth2"
)

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

func (p *Profile) UpdateToken(token *oauth2.Token) {
	p.token = token
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
		membershipURI:  p.membershipURI,
		baseServiceURI: p.baseServiceURI,
		token:          p.token,
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

	if p.token != nil && p.token.Expiry.Before(time.Now()) {
		relyingParty, err := rp.NewRelyingPartyOIDC(p.membershipURI, AuthClient, "",
			"", []string{"openid", "email", "offline_access"}, rp.WithHTTPClient(httpClient))
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

		p.token = newToken
		if err := p.config.Persist(); err != nil {
			return nil, err
		}
	}
	return p.token, nil
}

func (p *Profile) GetStackToken(ctx context.Context, httpClient *http.Client, organizationID, stackID string) (string, error) {
	apiUrl := MustApiUrl(*p, organizationID, stackID, "auth")
	form := url.Values{
		"grant_type": []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  []string{p.token.AccessToken},
		"scope":      []string{"openid email"},
	}

	discoveryConfiguration, err := client.Discover(apiUrl.String(), httpClient)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discoveryConfiguration.TokenEndpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("fctl", "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ret, err := httpClient.Do(req)
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

	token := oauth2.Token{}
	if err := json.NewDecoder(ret.Body).Decode(&token); err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

type CurrentProfile Profile
