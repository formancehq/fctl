package stacktools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ProxyOptions struct {
	Endpoint       string
	Client         *http.Client
	Port           uint32
	AllowedOrigins []string
	Timeout        time.Duration
}

// ProxyHandler strips browser credentials before the authenticated transport runs.
func ProxyHandler(options ProxyOptions) (http.Handler, error) {
	target, err := endpoint(options.Endpoint)
	if err != nil {
		return nil, err
	}
	client, err := requestClient(options.Client, options.Timeout)
	if err != nil {
		return nil, err
	}
	origins := slices.Clone(options.AllowedOrigins)
	proxy := &httputil.ReverseProxy{
		Transport: client.Transport,
		ErrorLog:  log.New(io.Discard, "", 0),
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
			for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "Origin", "Referer", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
				r.Out.Header.Del(name)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "stack request failed", http.StatusBadGateway)
		},
		ModifyResponse: func(response *http.Response) error {
			for name := range response.Header {
				if name == "Set-Cookie" || name == "Www-Authenticate" || name == "Authorization" || len(name) >= 15 && name[:15] == "Access-Control-" {
					response.Header.Del(name)
				}
			}
			return nil
		},
	}
	return proxyRequests(proxy, origins, options.Timeout), nil
}

func proxyRequests(proxy *httputil.ReverseProxy, origins []string, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopbackAuthority(r.Host) {
			http.Error(w, "host is not allowed", http.StatusForbidden)
			return
		}
		if !allowOrigin(w, r, origins) {
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		r = r.WithContext(ctx)
		r.Body = http.MaxBytesReader(w, r.Body, MaxMessageSize)
		proxy.ServeHTTP(w, r)
	})
}

func loopbackAuthority(authority string) bool {
	host, port, err := net.SplitHostPort(authority)
	if err == nil {
		if !validProxyPort(port) || (strings.HasPrefix(authority, "[") && !strings.Contains(host, ":")) {
			return false
		}
	} else {
		host = authority
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
			if !strings.Contains(host, ":") {
				return false
			}
		} else if strings.Contains(host, ":") {
			return false
		}
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validProxyPort(port string) bool {
	if port == "" || strings.IndexFunc(port, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return false
	}
	_, err := strconv.ParseUint(port, 10, 16)
	return err == nil
}

func allowOrigin(w http.ResponseWriter, r *http.Request, origins []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if !slices.Contains(origins, origin) && !slices.Contains(origins, "*") {
		http.Error(w, "origin is not allowed", http.StatusForbidden)
		return false
	}
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
	return true
}

// ServeProxy binds only loopback and drains the server on command cancellation.
func ServeProxy(ctx context.Context, options ProxyOptions, ready func(string) error) error {
	if options.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	handler, err := ProxyHandler(options)
	if err != nil {
		return err
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.FormatUint(uint64(options.Port), 10)))
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: options.Timeout, IdleTimeout: time.Minute, BaseContext: func(_ net.Listener) context.Context { return ctx }}
	if ready != nil {
		if err := ready(listener.Addr().String()); err != nil {
			return errors.Join(err, listener.Close())
		}
	}
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		err := server.Shutdown(cleanup)
		if err != nil {
			err = errors.Join(err, server.Close())
		}
		serveErr := <-result
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(ctx.Err(), err, serveErr)
	}
}
