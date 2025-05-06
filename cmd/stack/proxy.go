package stack

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	proxyPortFlag = "port"
)

type StackProxyStore struct {
	ProxyUrl       string `json:"proxyUrl"`
	StackUrl       string `json:"stackUrl"`
	OrganizationID string `json:"organizationId"`
	StackID        string `json:"stackId"`
}

type StackProxyController struct {
	store   *StackProxyStore
	profile *fctl.Profile
}

var _ fctl.Controller[*StackProxyStore] = (*StackProxyController)(nil)

func NewDefaultStackProxyStore() *StackProxyStore {
	return &StackProxyStore{}
}

func NewStackProxyController() *StackProxyController {
	return &StackProxyController{
		store: NewDefaultStackProxyStore(),
	}
}

func NewProxyCommand() *cobra.Command {
	cmd := fctl.NewStackCommand("proxy",
		fctl.WithShortDescription("Start a local proxy server to access the stack with authentication"),
		fctl.WithDescription("Start a local proxy server that adds authentication headers to requests to the stack"),
		fctl.WithIntFlag(proxyPortFlag, 55001, "Port to use for the local proxy server"),
		fctl.WithController(NewStackProxyController()),
	)

	return cmd
}

func (c *StackProxyController) GetStore() *StackProxyStore {
	return c.store
}

func (c *StackProxyController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetOrganizationStore(cmd)
	c.profile = store.Config.GetProfile(fctl.GetCurrentProfileName(cmd, store.Config))

	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	stack, err := fctl.ResolveStack(cmd, store.Config, organizationID)
	if err != nil {
		return nil, err
	}

	c.store.OrganizationID = organizationID
	c.store.StackID = stack.Id
	c.store.StackUrl = c.profile.ServicesBaseUrl(stack).String()

	port := fctl.GetInt(cmd, proxyPortFlag)
	c.store.ProxyUrl = fmt.Sprintf("http://localhost:%d", port)

	stackBaseURL, err := url.Parse(c.store.StackUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing stack URL: %v", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Target Stack URL: %s\r\n", stackBaseURL.String())

	proxy := httputil.NewSingleHostReverseProxy(stackBaseURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = stackBaseURL.Host

		targetURL := &url.URL{
			Scheme:   stackBaseURL.Scheme,
			Host:     stackBaseURL.Host,
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
		}

		sourceURL := &url.URL{
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
		}

		fmt.Fprintf(cmd.OutOrStdout(), "[%s] Proxying %s %s to %s\r\n",
			time.Now().Format(time.RFC3339),
			req.Method,
			sourceURL.String(),
			targetURL.String())
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		targetURL := &url.URL{
			Scheme:   stackBaseURL.Scheme,
			Host:     stackBaseURL.Host,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "[%s] ERROR proxying request to %s: %v\r\n",
			time.Now().Format(time.RFC3339),
			targetURL.String(),
			err)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		errorMsg := fmt.Sprintf("Proxy Error: %v\nTarget URL: %s",
			err,
			targetURL.String())
		io.WriteString(w, errorMsg)
	}

	transport := &tokenTransport{
		wrapped:  fctl.NewHTTPTransport(cmd, map[string][]string{}),
		profile:  c.profile,
		stack:    stack,
		cmd:      cmd,
		tokenMux: &sync.RWMutex{},
	}

	proxy.Transport = transport

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Create error channel to handle server errors from the main goroutine
	serverErrors := make(chan error, 1)

	go func() {
		fmt.Fprintf(cmd.OutOrStdout(), "Starting proxy server at %s -> %s\r\n", c.store.ProxyUrl, c.store.StackUrl)
		fmt.Fprintf(cmd.OutOrStdout(), "Press Ctrl+C to stop the server\r\n")

		// Check if port is available before starting the server
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			serverErrors <- fmt.Errorf("port %d is already in use or unavailable: %w", port, err)
			return
		}

		// Use the existing listener instead of closing it and calling ListenAndServe
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Handle either server error or stop signal
	select {
	case err := <-serverErrors:
		fmt.Fprintf(cmd.ErrOrStderr(), "Server error: %v\r\n", err)
		return nil, err
	case <-stop:
		fmt.Fprintf(cmd.OutOrStdout(), "\r\nShutting down server...\r\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error during server shutdown: %v\r\n", err)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Server stopped successfully\r\n")
	}

	// Create a dummy renderable to avoid nil pointer dereference
	return &EmptyRenderable{}, nil
}

// EmptyRenderable is a dummy implementation of the Renderable interface
type EmptyRenderable struct{}

func (r *EmptyRenderable) Render(cmd *cobra.Command, args []string) error {
	return nil
}

type tokenTransport struct {
	wrapped     http.RoundTripper
	profile     *fctl.Profile
	stack       *membershipclient.Stack
	cmd         *cobra.Command
	token       *oauth2.Token
	tokenMux    *sync.RWMutex
	tokenExpiry time.Time
}

func (t *tokenTransport) isTokenValid() bool {
	// First, grab a read lock to check for nil token and expiry zero-value.
	t.tokenMux.RLock()
	if t.token == nil {
		t.tokenMux.RUnlock()
		return false
	}

	// Check if token is expired
	if !t.tokenExpiry.IsZero() {
		valid := time.Until(t.tokenExpiry) > 30*time.Second
		t.tokenMux.RUnlock()
		return valid
	}

	t.tokenMux.RUnlock()
	return false
}

func (t *tokenTransport) refreshToken(ctx context.Context) error {
	t.tokenMux.Lock()
	defer t.tokenMux.Unlock()

	httpClient := fctl.GetHttpClient(t.cmd, map[string][]string{})
	newToken, err := t.profile.GetStackToken(ctx, httpClient, t.stack)
	if err != nil {
		return err
	}

	t.token = newToken
	t.tokenExpiry = newToken.Expiry

	fmt.Fprintf(t.cmd.OutOrStdout(), "[%s] Token refreshed, will expire at %s\r\n",
		time.Now().Format(time.RFC3339),
		t.tokenExpiry.Format(time.RFC3339))

	return nil
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.isTokenValid() {
		if err := t.refreshToken(req.Context()); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	t.tokenMux.RLock()
	token := t.token
	t.tokenMux.RUnlock()

	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))

	resp, err := t.wrapped.RoundTrip(reqCopy)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(t.cmd.ErrOrStderr(), "[%s] Error closing response body: %v\r\n",
				time.Now().Format(time.RFC3339), err)
		}

		if err := t.refreshToken(req.Context()); err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		t.tokenMux.RLock()
		token = t.token
		t.tokenMux.RUnlock()

		newReqCopy := req.Clone(req.Context())
		newReqCopy.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))

		return t.wrapped.RoundTrip(newReqCopy)
	}

	return resp, nil
}
