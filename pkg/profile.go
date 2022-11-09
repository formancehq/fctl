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

type Profile struct {
	MembershipURI  string        `json:"membershipURI"`
	BaseServiceURI string        `json:"baseServiceURI"`
	Token          *oauth2.Token `json:"token"`

	httpClient *http.Client
}

func (p *Profile) GetToken(ctx context.Context, httpClient *http.Client) (*oauth2.Token, error) {

	if p.Token != nil && p.Token.Expiry.Before(time.Now()) {
		relyingParty, err := rp.NewRelyingPartyOIDC(p.MembershipURI, AuthClient, "",
			"", []string{"openid", "email", "offline_access"}, rp.WithHTTPClient(httpClient))
		if err != nil {
			return nil, err
		}

		newToken, err := relyingParty.
			OAuthConfig().
			TokenSource(context.WithValue(ctx, oauth2.HTTPClient, httpClient), p.Token).
			Token()
		if err != nil {
			return nil, err
		}

		p.Token = newToken

		// TODO: Persist config at upper level
		//if err := configManager.
		//	UpdateConfig(ConfigFromContext(ctx)); err != nil {
		//	return nil, err
		//}
	}
	return p.Token, nil
}

func (p *Profile) GetStackToken(ctx context.Context, httpClient *http.Client, organizationID, stackID string) (string, error) {
	apiUrl := MustApiUrl(*p, organizationID, stackID, "auth")
	form := url.Values{
		"grant_type": []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  []string{p.Token.AccessToken},
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
