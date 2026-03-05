package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/formancehq/fctl/pkg/pluginsdk/pluginpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/servicepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// NewClient creates a gRPC BucketServiceClient from the plugin request flags.
func NewClient(flags map[string]string) (servicepb.BucketServiceClient, *grpc.ClientConn, error) {
	serverAddr := flags["server"]
	if serverAddr == "" {
		serverAddr = "localhost:8888"
	}

	var creds credentials.TransportCredentials
	if flags["insecure"] == "true" {
		creds = insecure.NewCredentials()
	} else {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		caCertPath := flags["tls-ca-cert"]
		if caCertPath != "" {
			caPEM, err := os.ReadFile(caCertPath)
			if err != nil {
				return nil, nil, fmt.Errorf("reading CA cert: %w", err)
			}
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caPEM) {
				return nil, nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = certPool
		}
		creds = credentials.NewTLS(tlsConfig)
	}

	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return servicepb.NewBucketServiceClient(conn), conn, nil
}

// ContextWithAuth returns a context with auth metadata if a token is provided.
func ContextWithAuth(ctx context.Context, req *pluginpb.ExecuteRequest) context.Context {
	token := req.Flags["auth-token"]
	if token == "" && req.AuthContext != nil {
		token = req.AuthContext.AccessToken
	}
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}
	return ctx
}

// GetFlag returns a flag value or the default.
func GetFlag(flags map[string]string, name, defaultValue string) string {
	if v, ok := flags[name]; ok && v != "" {
		return v
	}
	return defaultValue
}
