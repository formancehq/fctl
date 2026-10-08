// Package cloudtools attaches hosted utilities to the manifest-built Cloud tree.
package cloudtools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/stacktools"
)

type resolver func(context.Context, *cobra.Command, string) (*api.Client, error)
type tokenReader func(context.Context, *http.Client) (string, error)

// AddTo requires the Cloud and stack groups to have been installed by the registry.
func AddTo(root *cobra.Command, settings *connection.Settings) error {
	if settings == nil {
		return fmt.Errorf("connection settings are required")
	}
	return addTo(root, settings.Client, cloud.StackAccessToken)
}

func addTo(root *cobra.Command, resolve resolver, token tokenReader) error {
	cloudCommand := child(root, "cloud")
	stack := child(cloudCommand, "stack")
	if cloudCommand == nil || stack == nil {
		return fmt.Errorf("cloud and stack commands must be installed before hosted utilities")
	}
	for _, name := range []string{"proxy", "mcp"} {
		if child(stack, name) != nil {
			return fmt.Errorf("stack command %s already exists", name)
		}
	}
	if child(cloudCommand, "generate-personal-token") != nil || child(cloudCommand, "gpt") != nil {
		return fmt.Errorf("personal token command already exists")
	}
	stack.AddCommand(proxyCommand(resolve), mcpCommand(resolve))
	cloudCommand.AddCommand(tokenCommand(resolve, token))
	return nil
}

func child(parent *cobra.Command, name string) *cobra.Command {
	if parent == nil {
		return nil
	}
	for _, candidate := range parent.Commands() {
		if candidate.Name() == name || candidate.HasAlias(name) {
			return candidate
		}
	}
	return nil
}

func proxyCommand(resolve resolver) *cobra.Command {
	var port uint32
	var origins []string
	cmd := &cobra.Command{Use: "proxy", Short: "Serve an authenticated stack proxy on loopback", Args: cobra.NoArgs}
	cmd.Flags().Uint32Var(&port, "port", 55001, "Loopback port (0 chooses an available port)")
	cmd.Flags().StringSliceVar(&origins, "allowed-origins", nil, "Allowed browser origins (comma-separated)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if port > 65535 {
			return fmt.Errorf("port must be between 0 and 65535")
		}
		timeout, err := cmd.Flags().GetDuration("timeout")
		if err != nil {
			return err
		}
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		client, err := resolve(cmd.Context(), cmd, "stack")
		if err != nil {
			return err
		}
		options := stacktools.ProxyOptions{Endpoint: client.Endpoint(), Client: client.HTTPClient(), Port: port, AllowedOrigins: origins, Timeout: timeout}
		return stacktools.ServeProxy(cmd.Context(), options, func(address string) error {
			_, err := fmt.Fprintf(cmd.ErrOrStderr(), "Stack proxy listening on http://%s\n", address)
			return err
		})
	}
	return cmd
}

func mcpCommand(resolve resolver) *cobra.Command {
	group := &cobra.Command{Use: "mcp", Short: "Stack MCP integrations"}
	var transport string
	serve := &cobra.Command{Use: "serve", Short: "Bridge stack MCP over stdio", Args: cobra.NoArgs}
	serve.Flags().StringVar(&transport, "transport", "stdio", "MCP transport (stdio)")
	serve.RunE = func(cmd *cobra.Command, _ []string) error {
		if transport != "stdio" {
			return fmt.Errorf("unsupported MCP transport: only stdio is supported")
		}
		timeout, err := cmd.Flags().GetDuration("timeout")
		if err != nil {
			return err
		}
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		client, err := resolve(cmd.Context(), cmd, "stack")
		if err != nil {
			return err
		}
		return stacktools.ServeMCP(cmd.Context(), stacktools.MCPOptions{Endpoint: client.Endpoint(), Client: client.HTTPClient(), Timeout: timeout, Input: cmd.InOrStdin(), Output: cmd.OutOrStdout()})
	}
	group.AddCommand(serve)
	return group
}

func tokenCommand(resolve resolver, read tokenReader) *cobra.Command {
	cmd := &cobra.Command{Use: "generate-personal-token", Aliases: []string{"gpt"}, Short: "Print the selected stack's bearer token", Args: cobra.NoArgs}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := resolve(cmd.Context(), cmd, "stack")
		if err != nil {
			return err
		}
		value, err := read(cmd.Context(), client.HTTPClient())
		if err != nil {
			return err
		}
		data, err := json.Marshal(struct {
			Token string `json:"token"`
		}{Token: value})
		if err != nil {
			return err
		}
		if err := command.ConfigureOutput(cmd, "json", "never"); err != nil {
			return err
		}
		return command.WriteJSON(cmd.OutOrStdout(), data)
	}
	return cmd
}
