package stack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/zitadel/oidc/pkg/client"
	"golang.org/x/oauth2"
)

func GetToken(ctx context.Context, profile fctl.Profile, organization, stack string) (string, error) {

	apiUrl := fctl.MustApiUrl(profile, organization, stack, "auth")
	form := url.Values{
		"grant_type": []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  []string{profile.Token.AccessToken},
		"scope":      []string{"openid email"},
	}

	discoveryConfiguration, err := client.Discover(apiUrl.String(), &http.Client{
		Transport: fctl.DebugRoundTripper(http.DefaultTransport),
	})
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

	token := oauth2.Token{}
	if err := json.NewDecoder(ret.Body).Decode(&token); err != nil {
		return "", err
	}

	return token.AccessToken, nil
}
