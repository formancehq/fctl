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

type RoundTripperFn func(req *http.Request) (*http.Response, error)

func (fn RoundTripperFn) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
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

	var transport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: GetBool(cmd, InsecureTlsFlag),
		},
	}
	if GetBool(cmd, DebugFlag) {
		transport = debugRoundTripper(transport)
	}

	return newInjectHTTPHeadersRoundTripper(
		http.Header{
			"User-Agent": []string{fmt.Sprintf("fctl/%s", getVersion(cmd))},
		},
		transport,
	)
}

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
