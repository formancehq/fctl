package fctl

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-jose/go-jose/v4"
	"github.com/pterm/pterm"
	"golang.org/x/oauth2"

	"github.com/formancehq/go-libs/v3/oidc/client"
	httphelper "github.com/formancehq/go-libs/v3/oidc/http"
)

func GetAuthRelyingParty(ctx context.Context, httpClient *http.Client, membershipURI string) (client.RelyingParty, error) {
	pterm.Debug.Println("Getting auth relying party on membership URI:", membershipURI)
	return client.NewRelyingPartyOIDC(
		ctx,
		membershipURI,
		AuthClient,
		"",
		"",
		[]string{},
		client.WithHTTPClient(httpClient),
	)
}

// lazyRelyingParty defers the OIDC discovery call until a method (other than
// HttpClient) is actually invoked. This avoids hitting the .well-known/openid-configuration
// endpoint when a cached stack API token makes it unnecessary.
//
// Uses sync.Mutex instead of sync.Once so that transient discovery failures
// (e.g. network timeouts) are retried on the next call rather than cached
// permanently.
type lazyRelyingParty struct {
	ctx           context.Context
	httpClient    *http.Client
	membershipURI string

	mu       sync.Mutex
	delegate client.RelyingParty
}

func (l *lazyRelyingParty) init() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.delegate != nil {
		return nil
	}
	var err error
	l.delegate, err = GetAuthRelyingParty(l.ctx, l.httpClient, l.membershipURI)
	return err
}

func (l *lazyRelyingParty) mustDelegate() client.RelyingParty {
	if err := l.init(); err != nil {
		panic(fmt.Sprintf("OIDC discovery failed: %v", err))
	}
	return l.delegate
}

// HttpClient returns the http client without triggering OIDC discovery.
func (l *lazyRelyingParty) HttpClient() *http.Client {
	return l.httpClient
}

func (l *lazyRelyingParty) OAuthConfig() *oauth2.Config {
	return l.mustDelegate().OAuthConfig()
}
func (l *lazyRelyingParty) Issuer() string {
	return l.mustDelegate().Issuer()
}
func (l *lazyRelyingParty) IsPKCE() bool {
	return l.mustDelegate().IsPKCE()
}
func (l *lazyRelyingParty) CookieHandler() *httphelper.CookieHandler {
	return l.mustDelegate().CookieHandler()
}
func (l *lazyRelyingParty) IsOAuth2Only() bool {
	return l.mustDelegate().IsOAuth2Only()
}
func (l *lazyRelyingParty) Signer() jose.Signer {
	return l.mustDelegate().Signer()
}
func (l *lazyRelyingParty) GetEndSessionEndpoint() string {
	return l.mustDelegate().GetEndSessionEndpoint()
}
func (l *lazyRelyingParty) GetRevokeEndpoint() string {
	return l.mustDelegate().GetRevokeEndpoint()
}
func (l *lazyRelyingParty) UserinfoEndpoint() string {
	return l.mustDelegate().UserinfoEndpoint()
}
func (l *lazyRelyingParty) GetDeviceAuthorizationEndpoint() string {
	return l.mustDelegate().GetDeviceAuthorizationEndpoint()
}
func (l *lazyRelyingParty) GetIntrospectionEndpoint() string {
	return l.mustDelegate().GetIntrospectionEndpoint()
}
func (l *lazyRelyingParty) IDTokenVerifier() *client.Verifier {
	return l.mustDelegate().IDTokenVerifier()
}
func (l *lazyRelyingParty) ErrorHandler() func(http.ResponseWriter, *http.Request, string, string, string) {
	return l.mustDelegate().ErrorHandler()
}

var _ client.RelyingParty = (*lazyRelyingParty)(nil)

func NewLazyRelyingParty(ctx context.Context, httpClient *http.Client, membershipURI string) client.RelyingParty {
	return &lazyRelyingParty{
		ctx:           ctx,
		httpClient:    httpClient,
		membershipURI: membershipURI,
	}
}
