// Package stacktools provides authenticated stack utilities without CLI dependencies.
package stacktools

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

const MaxMessageSize = 10 * 1024 * 1024

func endpoint(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("stack endpoint must be an absolute URL without credentials, query or fragment")
	}
	if u.Scheme == "https" {
		return u, nil
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) {
		return u, nil
	}
	return nil, fmt.Errorf("stack endpoint requires HTTPS except on loopback")
}

func requestClient(client *http.Client, timeout time.Duration) (*http.Client, error) {
	if client == nil || timeout <= 0 {
		return nil, fmt.Errorf("HTTP client and positive timeout are required")
	}
	copyClient := *client
	copyClient.Timeout = timeout
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &copyClient, nil
}
