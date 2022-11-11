package debugutil

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/formancehq/fctl/cmd/config"
	"github.com/spf13/viper"
)

func GetHttpClient() *http.Client {
	return NewHTTPClient(viper.GetBool(config.InsecureTlsFlag), viper.GetBool(config.DebugFlag))
}

type RoundTripperFn func(req *http.Request) (*http.Response, error)

func (fn RoundTripperFn) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func debugRoundTripper(rt http.RoundTripper) RoundTripperFn {
	return func(req *http.Request) (*http.Response, error) {
		data, err := httputil.DumpRequest(req, true)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))

		rsp, err := rt.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		data, err = httputil.DumpResponse(rsp, true)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))

		return rsp, nil
	}
}

func NewHTTPClient(insecureTLS, debug bool) *http.Client {
	var transport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureTLS,
		},
	}
	if debug {
		transport = debugRoundTripper(transport)
	}
	return &http.Client{
		Transport: transport,
	}
}
