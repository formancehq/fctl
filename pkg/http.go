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

func GetHttpClient(cmd *cobra.Command, defaultHeaders map[string][]string) *http.Client {
	return NewHTTPClient(
		cmd,
		defaultHeaders,
	)
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
			req.Body.Close()
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
			rsp.Body.Close()
			rsp.Body = io.NopCloser(bytes.NewBuffer(data))
			printBody(data)
		}

		return rsp, nil
	}
}

func defaultHeadersRoundTripper(rt http.RoundTripper, headers map[string][]string) RoundTripperFn {
	return func(req *http.Request) (*http.Response, error) {
		for k, v := range headers {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
		return rt.RoundTrip(req)
	}
}

func NewHTTPClient(cmd *cobra.Command, defaultHeaders map[string][]string) *http.Client {
	return &http.Client{
		Transport: NewHTTPTransport(cmd, defaultHeaders),
	}
}

func NewHTTPTransport(cmd *cobra.Command, defaultHeaders map[string][]string) http.RoundTripper {

	var transport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: GetBool(cmd, InsecureTlsFlag),
		},
	}
	if GetBool(cmd, DebugFlag) {
		transport = debugRoundTripper(transport)
	}
	if len(defaultHeaders) > 0 {
		transport = defaultHeadersRoundTripper(transport, defaultHeaders)
	}
	return transport
}
