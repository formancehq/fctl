package fctl

import (
	"net/http"
	"net/http/httputil"
)

type RoundTripperFn func(req *http.Request) (*http.Response, error)

func (fn RoundTripperFn) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func DebugRoundTripper(rt http.RoundTripper) RoundTripperFn {
	return func(req *http.Request) (*http.Response, error) {
		_, err := httputil.DumpRequest(req, true)
		if err != nil {
			panic(err)
		}

		rsp, err := rt.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		_, err = httputil.DumpResponse(rsp, true)
		if err != nil {
			panic(err)
		}

		return rsp, nil
	}
}
