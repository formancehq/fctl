package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

// This complements subprocess tests: the server-side paths are measured by
// coverage in the host test process, while real executables prove isolation.
func inProcessPlugin(t *testing.T, factory func(*http.Client) pluginsdk.Plugin, httpClient *http.Client, endpoint string) *Client {
	t.Helper()
	protocol, _ := goplugin.TestPluginGRPCConn(t, false, goplugin.PluginSet{pluginName: &grpcPlugin{factory: factory}})
	t.Cleanup(func() {
		if err := protocol.Close(); err != nil {
			t.Error(err)
		}
	})
	dispensed, err := protocol.Dispense(pluginName)
	if err != nil {
		t.Fatal(err)
	}
	remote, ok := dispensed.(*remotePlugin)
	if !ok {
		t.Fatal("invalid fixture plugin")
	}
	target, err := parseEndpoint(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{remote: remote, http: httpClient, endpoint: target, lifetime: t.Context()}
}

func TestGRPCServerRoundTripAndHostHTTP(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeResponse(t, w, ` {"id":90071992547409930001} `)
	}))
	defer server.Close()
	client := inProcessPlugin(t, func(client *http.Client) pluginsdk.Plugin { return &helperPlugin{http: client} }, server.Client(), server.URL)
	manifest, err := client.GetManifest(t.Context())
	if err != nil || !reflect.DeepEqual(manifest, helperManifest()) {
		t.Fatalf("manifest = %+v, %v", manifest, err)
	}
	request := helperRequest("http")
	response, err := client.Execute(t.Context(), request)
	if err != nil || string(response.Data) != ` {"id":90071992547409930001} ` {
		t.Fatalf("HTTP = %s, %v", response.Data, err)
	}
	response, err = client.Execute(t.Context(), helperRequest("error"))
	if failure, ok := errors.AsType[*httpclient.Error](err); !ok || failure.StatusCode != 409 || len(response.Data) == 0 {
		t.Fatalf("partial error = %s, %v", response.Data, err)
	}
}

type metadataPlugin struct {
	manifest pluginsdk.Manifest
	err      error
}

func (p *metadataPlugin) GetManifest(context.Context) (pluginsdk.Manifest, error) {
	return p.manifest, p.err
}

func (*metadataPlugin) Execute(context.Context, pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	return pluginsdk.ExecuteResponse{}, errors.New("not executable")
}

func TestGRPCRejectsIncompatibleAndBrokenMetadata(t *testing.T) {
	t.Parallel()
	manifest := helperManifest()
	manifest.ProtocolVersion++
	client := inProcessPlugin(t, func(*http.Client) pluginsdk.Plugin { return &metadataPlugin{manifest: manifest} }, nil, "")
	if _, err := client.GetManifest(t.Context()); err == nil {
		t.Fatal("incompatible manifest accepted")
	}
	broken := inProcessPlugin(t, func(*http.Client) pluginsdk.Plugin { return &metadataPlugin{err: errors.New("broken metadata")} }, nil, "")
	if _, err := broken.GetManifest(t.Context()); err == nil {
		t.Fatal("metadata failure was lost")
	}
}

func TestRPCRejectsInvalidPayloadAndPreservesErrorKinds(t *testing.T) {
	t.Parallel()
	server := &pluginServer{}
	if _, err := server.Execute(t.Context(), wrapperspb.Bytes([]byte(`{}`))); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing broker = %v", err)
	}
	if _, err := server.Execute(t.Context(), wrapperspb.Bytes([]byte(`invalid`))); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad JSON = %v", err)
	}
	for _, original := range []error{context.Canceled, context.DeadlineExceeded} {
		if !errors.Is(encodeError(original).err(), original) {
			t.Fatalf("context error changed: %v", original)
		}
	}
	if _, err := pack(make(chan int)); err == nil {
		t.Fatal("invalid outgoing payload accepted")
	}
	if err := unpack(nil, new(resultWire)); err == nil {
		t.Fatal("missing incoming payload accepted")
	}
	if err := validateResult(resultWire{Data: []byte(`invalid`)}); err == nil {
		t.Fatal("invalid response JSON accepted")
	}
}

func TestGRPCDescriptorsSupportInterceptors(t *testing.T) {
	t.Parallel()
	decode := func(value any) error {
		message, ok := value.(*wrapperspb.BytesValue)
		if !ok {
			return errors.New("invalid decoded request")
		}
		message.Value = []byte(`{}`)
		return nil
	}
	interceptor := func(ctx context.Context, message any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod != "/"+pluginService+"/GetManifest" {
			return nil, errors.New("invalid method metadata")
		}
		return handler(ctx, message)
	}
	server := &pluginServer{factory: func(*http.Client) pluginsdk.Plugin { return &helperPlugin{} }}
	if _, err := manifestHandler(server, t.Context(), decode, interceptor); err != nil {
		t.Fatal(err)
	}
	if _, err := manifestHandler(server, t.Context(), func(any) error { return errors.New("decode failed") }, nil); err == nil {
		t.Fatal("request decode failure lost")
	}
}
