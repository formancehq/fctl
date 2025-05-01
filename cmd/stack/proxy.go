package stack

import (
	"context"
	"fmt"
	"io"
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
	"github.com/golang-jwt/jwt"
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
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithIntFlag(proxyPortFlag, 55001, "Port to use for the local proxy server"),
		fctl.WithController(NewStackProxyController()),
	)

	// Override RunE to use our custom handling
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Run the original command but handle the result differently
		if err := originalRunE(cmd, args); err != nil {
			return err
		}

		// Prevent returning to cobra by exiting directly
		os.Exit(0)
		return nil // Never reached
	}

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

	fmt.Fprintf(cmd.OutOrStdout(), "Target Stack URL: %s\n", stackBaseURL.String())

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

		fmt.Fprintf(cmd.OutOrStdout(), "[%s] Proxying %s %s to %s\n",
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

		fmt.Fprintf(cmd.ErrOrStderr(), "[%s] ERROR proxying request to %s: %v\n",
			time.Now().Format(time.RFC3339),
			targetURL.String(),
			err)

		w.WriteHeader(http.StatusBadGateway)
		w.Header().Set("Content-Type", "text/plain")
		errorMsg := fmt.Sprintf("Proxy Error: %v\nTarget URL: %s",
			err,
			targetURL.String())
		io.WriteString(w, errorMsg)
	}

	transport := &tokenTransport{
		wrapped:  http.DefaultTransport,
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

	go func() {
		fmt.Fprintf(cmd.OutOrStdout(), "Starting proxy server at %s -> %s\n", c.store.ProxyUrl, c.store.StackUrl)
		fmt.Fprintf(cmd.OutOrStdout(), "Press Ctrl+C to stop the server\n")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(cmd.ErrOrStderr(), "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Fprintln(cmd.OutOrStdout(), "\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error during server shutdown: %v\n", err)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Server stopped successfully")
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
	t.tokenMux.RLock()
	defer t.tokenMux.RUnlock()

	if t.token == nil {
		return false
	}

	if t.tokenExpiry.IsZero() {
		parser := jwt.Parser{}
		claims := jwt.MapClaims{}
		_, _, err := parser.ParseUnverified(t.token.AccessToken, claims)
		if err != nil {
			return false
		}

		if exp, ok := claims["exp"].(float64); ok {
			t.tokenExpiry = time.Unix(int64(exp), 0)
		} else {
			return false
		}
	}

	return time.Until(t.tokenExpiry) > 30*time.Second
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
	t.tokenExpiry = time.Time{}

	parser := jwt.Parser{}
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(newToken.AccessToken, claims)
	if err == nil {
		if exp, ok := claims["exp"].(float64); ok {
			t.tokenExpiry = time.Unix(int64(exp), 0)
			fmt.Fprintf(t.cmd.OutOrStdout(), "[%s] Token refreshed, will expire at %s\n",
				time.Now().Format(time.RFC3339),
				t.tokenExpiry.Format(time.RFC3339))
		}
	}

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
			fmt.Fprintf(t.cmd.ErrOrStderr(), "[%s] Error closing response body: %v\n",
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
