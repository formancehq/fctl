package plugin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	pluginURLTimeout      = 30 * time.Second
	maxPluginURLRedirects = 5
)

// URL sources use a separate public transport with no stack authentication,
// cookie jar, or inherited default-client middleware.
func newPluginURLClient() *http.Client {
	return &http.Client{
		Timeout:       pluginURLTimeout,
		CheckRedirect: checkPluginURLRedirect,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   pluginURLTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}

func readPluginURL(ctx context.Context, source string, client *http.Client) (data []byte, err error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPS input URL")
	}
	if err := validatePluginURL(u); err != nil {
		return nil, err
	}
	if strings.Contains(source, "#") {
		return nil, fmt.Errorf("HTTPS input URL must not contain a fragment")
	}
	u.Scheme = "https"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create HTTPS input request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		// net/http includes the URL in its outer error. Keep the cause for
		// cancellation checks without exposing credentials or query values.
		if failure, ok := errors.AsType[*url.Error](err); ok {
			err = failure.Err
		}
		return nil, fmt.Errorf("fetch HTTPS input: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTPS input returned HTTP status %d", resp.StatusCode)
	}
	return readBoundedPluginInput(ctx, resp.Body)
}

func validatePluginURL(u *url.URL) error {
	if !strings.EqualFold(u.Scheme, "https") || u.Hostname() == "" || u.Opaque != "" {
		return fmt.Errorf("input URL must be an absolute HTTPS URL")
	}
	if u.User != nil {
		return fmt.Errorf("HTTPS input URL must not contain credentials")
	}
	if u.Fragment != "" || u.RawFragment != "" {
		return fmt.Errorf("HTTPS input URL must not contain a fragment")
	}
	return nil
}

func checkPluginURLRedirect(req *http.Request, via []*http.Request) error {
	if err := validatePluginURL(req.URL); err != nil {
		return err
	}
	if req.Response != nil && strings.Contains(req.Response.Header.Get("Location"), "#") {
		return fmt.Errorf("HTTPS input URL must not contain a fragment")
	}
	if len(via) > maxPluginURLRedirects {
		return fmt.Errorf("HTTPS input exceeded %d redirects", maxPluginURLRedirects)
	}
	req.URL.Scheme = "https"
	return nil
}
