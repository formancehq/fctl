package plugin

import (
	"fmt"
	"net/http"
)

// Choice discovery has no authority to mutate a service resource, even if a
// malformed manifest points a selector at the wrong command.
type readOnlyTransport struct{ base http.RoundTripper }

func (r readOnlyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodGet {
		return nil, fmt.Errorf("choice discovery refused a mutating request")
	}
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(request)
}

func choiceClient(client *http.Client) *http.Client {
	copy := *client
	copy.Transport = readOnlyTransport{base: client.Transport}
	return &copy
}
