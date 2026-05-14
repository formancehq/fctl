package target

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type ProxyInput struct {
	TargetURL      string
	Transport      http.RoundTripper
	AllowedOrigins []string
	LogWriter      io.Writer
	ErrorWriter    io.Writer
}

func NewProxyHandler(input ProxyInput) (http.Handler, error) {
	targetURL, err := url.Parse(input.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("parse target URL: %w", err)
	}
	if targetURL.Scheme == "" || targetURL.Host == "" {
		return nil, fmt.Errorf("target URL must include scheme and host")
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	if input.Transport != nil {
		proxy.Transport = input.Transport
	} else {
		proxy.Transport = http.DefaultTransport
	}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		if input.LogWriter != nil {
			_, _ = fmt.Fprintf(input.LogWriter, "[%s] Proxying %s %s\n", time.Now().Format(time.RFC3339), req.Method, req.URL.String())
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if input.ErrorWriter != nil {
			_, _ = fmt.Fprintf(input.ErrorWriter, "[%s] ERROR proxying %s: %v\n", time.Now().Format(time.RFC3339), r.URL.String(), err)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprintf(w, "Proxy Error: %v\n", err)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		clearCORSHeaders(resp.Header)
		return nil
	}

	var handler http.Handler = proxy
	if len(input.AllowedOrigins) > 0 {
		handler = CORSMiddleware(input.AllowedOrigins, proxy)
	}
	return handler, nil
}

func CORSMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originAllowed(origin, allowedOrigins) {
			if allowsWildcard(allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Count")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func originAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}
	for _, allowedOrigin := range allowedOrigins {
		allowedOrigin = strings.TrimSpace(allowedOrigin)
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

func allowsWildcard(allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if strings.TrimSpace(allowedOrigin) == "*" {
			return true
		}
	}
	return false
}

func clearCORSHeaders(header http.Header) {
	header.Del("Access-Control-Allow-Origin")
	header.Del("Access-Control-Allow-Methods")
	header.Del("Access-Control-Allow-Headers")
	header.Del("Access-Control-Allow-Credentials")
	header.Del("Access-Control-Max-Age")
	header.Del("Access-Control-Expose-Headers")
	header.Del("Access-Control-Request-Method")
	header.Del("Access-Control-Request-Headers")
}
