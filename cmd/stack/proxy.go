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
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	proxyPortFlag      = "port"
	allowedOriginsFlag = "allowed-origins"
)

type ProxyStore struct {
}

type ProxyController struct {
	store *ProxyStore
}

var _ fctl.Controller[*ProxyStore] = (*ProxyController)(nil)

func NewDefaultStackProxyStore() *ProxyStore {
	return &ProxyStore{}
}

func NewStackProxyController() *ProxyController {
	return &ProxyController{
		store: NewDefaultStackProxyStore(),
	}
}

func NewProxyCommand() *cobra.Command {
	cmd := fctl.NewStackCommand("proxy",
		fctl.WithShortDescription("Start a local proxy server to access the stack with authentication"),
		fctl.WithDescription("Start a local proxy server that adds authentication headers to requests to the stack"),
		fctl.WithIntFlag(proxyPortFlag, 55001, "Port to use for the local proxy server"),
		fctl.WithStringSliceFlag(allowedOriginsFlag, []string{}, "Allowed origins for CORS (comma-separated). If not specified, CORS is disabled."),
		fctl.WithController(NewStackProxyController()),
	)

	return cmd
}

func (c *ProxyController) GetStore() *ProxyStore {
	return c.store
}

// corsMiddleware creates a CORS middleware that handles preflight requests and adds CORS headers
func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		originAllowed := false
		allowWildcard := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" {
				originAllowed = true
				allowWildcard = true
				break
			} else if allowedOrigin == origin {
				originAllowed = true
				break
			}
		}

		// Set CORS headers only for allowed origins
		if originAllowed {
			if allowWildcard {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// TODO: forward services provided headers - also check where headers are cleared (line ~208)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Count")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *ProxyController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackToken, stackAccess, err := fctl.EnsureStackAccess(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	port := fctl.GetInt(cmd, proxyPortFlag)
	allowedOrigins := fctl.GetStringSlice(cmd, allowedOriginsFlag)

	stackBaseURL, err := url.Parse(stackAccess.URI)
	if err != nil {
		return nil, fmt.Errorf("error parsing stack URL: %v", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Target Stack URL: %s\r\n", stackBaseURL.String())

	// Only show CORS info if origins are specified
	if len(allowedOrigins) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CORS enabled - Allowed Origins: %s\r\n", strings.Join(allowedOrigins, ", "))
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CORS disabled\r\n")
	}

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

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] Proxying %s %s to %s\r\n",
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

		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[%s] ERROR proxying request to %s: %v\r\n",
			time.Now().Format(time.RFC3339),
			targetURL.String(),
			err)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		errorMsg := fmt.Sprintf("Proxy Error: %v\nTarget URL: %s",
			err,
			targetURL.String())
		_, _ = io.WriteString(w, errorMsg)
	}

	proxy.Transport = &oauth2.Transport{
		Base: fctl.GetHttpClient(cmd).Transport,
		Source: fctl.NewStackTokenSource(
			*stackToken,
			stackAccess,
			relyingParty,
			func(newToken fctl.AccessToken) error {
				return fctl.WriteStackToken(cmd, profileName, stackID, newToken)
			},
			cmd,
			profileName,
			organizationID,
			stackID,
		),
	}

	// Clear any CORS headers from upstream to prevent conflicts with our CORS middleware
	// TODO: forward services provided headers - also check where headers are added (line ~97)
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Max-Age")
		resp.Header.Del("Access-Control-Expose-Headers")
		resp.Header.Del("Access-Control-Request-Method")
		resp.Header.Del("Access-Control-Request-Headers")

		return nil
	}

	// Only wrap with CORS middleware if origins are specified
	var handler http.Handler = proxy
	if len(allowedOrigins) > 0 {
		handler = corsMiddleware(allowedOrigins, proxy)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Create error channel to handle server errors from the main goroutine
	serverErrors := make(chan error, 1)

	go func() {
		fmt.Fprintf(cmd.OutOrStdout(), "Starting proxy server at http://localhost:%d -> %s\r\n", port, stackAccess.URI)
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
