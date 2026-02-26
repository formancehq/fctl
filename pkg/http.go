package fctl

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/cobra"
)

func GetHttpClient(cmd *cobra.Command) *http.Client {
	return &http.Client{
		Transport: NewHTTPTransport(cmd),
	}
}

func printBody(data []byte) {
	if len(data) == 0 {
		return
	}
	raw := make(map[string]any)
	if err := json.Unmarshal(data, &raw); err == nil {
		f := colorjson.NewFormatter()
		f.Indent = 2
		colorized, err := f.Marshal(raw)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(colorized))
	} else {
		fmt.Println(string(data))
	}
}

func debugRoundTripper(rt http.RoundTripper) RoundTripperFn {
	return func(req *http.Request) (*http.Response, error) {
		data, err := httputil.DumpRequest(req, false)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))

		if req.Body != nil {
			data, err = io.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}
			_ = req.Body.Close()
			req.Body = io.NopCloser(bytes.NewBuffer(data))
			printBody(data)
		}

		rsp, err := rt.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		data, err = httputil.DumpResponse(rsp, false)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))

		if rsp.Body != nil {
			data, err = io.ReadAll(rsp.Body)
			if err != nil {
				panic(err)
			}
			_ = rsp.Body.Close()
			rsp.Body = io.NopCloser(bytes.NewBuffer(data))
			printBody(data)
		}

		return rsp, nil
	}
}

func NewHTTPTransport(cmd *cobra.Command) http.RoundTripper {
	var transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: GetBool(cmd, InsecureTlsFlag),
		},
	}
	var roundTripper http.RoundTripper = transport

	if GetBool(cmd, HTTPCloseOnErrorFlag) {
		roundTripper = &closeOnErrorRoundTripper{
			transport: transport,
		}
	}

	if GetBool(cmd, DebugFlag) {
		roundTripper = debugRoundTripper(roundTripper)
	}

	return newInjectHTTPHeadersRoundTripper(
		http.Header{
			"User-Agent": []string{fmt.Sprintf("fctl/%s", getVersion(cmd))},
		},
		roundTripper,
	)
}

var _ http.RoundTripper = (*RoundTripperFn)(nil)

type RoundTripperFn func(req *http.Request) (*http.Response, error)

func (fn RoundTripperFn) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

var _ http.RoundTripper = (*closeOnErrorRoundTripper)(nil)

type closeOnErrorRoundTripper struct {
	transport *http.Transport
}

func (rt *closeOnErrorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rsp, err := rt.transport.RoundTrip(req)
	if err != nil || rsp.StatusCode >= 400 {
		_ = rsp.Body.Close()
		rt.transport.CloseIdleConnections()
	}
	return rsp, err
}

var _ http.RoundTripper = (*injectHTTPHeadersRoundTripper)(nil)

type injectHTTPHeadersRoundTripper struct {
	headers http.Header
	next    http.RoundTripper
}

func (rt *injectHTTPHeadersRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for key, values := range rt.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return rt.next.RoundTrip(req)
}

func newInjectHTTPHeadersRoundTripper(headers http.Header, next http.RoundTripper) http.RoundTripper {
	return &injectHTTPHeadersRoundTripper{
		headers: headers,
		next:    next,
	}
}
