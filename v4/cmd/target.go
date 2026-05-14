package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	targetcmd "github.com/formancehq/fctl/v4/internal/commands/target"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

func newTargetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "target",
		Short: "Inspect the active fctl v4 target",
	}
	command.AddCommand(newTargetInspectCommand())
	command.AddCommand(newTargetProxyCommand())
	return command
}

func newTargetInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the current target and inferred capabilities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			versions, err := rt.ComponentVersions(cmd.Context())
			if err != nil {
				return err
			}

			components := make([]targetInspectComponent, 0, len(versions))
			for _, version := range versions {
				apiVersions, _ := rt.Compatibility.APIVersionsFor(version.Product, version.Version)
				components = append(components, targetInspectComponent{
					Name:        string(version.Product),
					Version:     version.Version,
					Health:      version.Health,
					APIVersions: apiVersionsToStrings(apiVersions),
					APIPolicy:   string(rt.APIPolicyFor(version.Product)),
				})
			}
			output := targetInspectOutput{
				Context:    rt.ContextName,
				TargetURL:  rt.Target.URL,
				TargetKind: string(rt.Target.Kind),
				Components: components,
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}

			if !terminalOutputEnabled(cmd) {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Context: %s\nTarget: %s (%s)\n", output.Context, output.TargetURL, output.TargetKind); err != nil {
					return err
				}
				if len(output.Components) == 0 {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Components: none")
					return err
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Components:"); err != nil {
					return err
				}
				for _, component := range output.Components {
					health := "unhealthy"
					if component.Health {
						health = "healthy"
					}
					apiVersions := "<none>"
					if len(component.APIVersions) > 0 {
						apiVersions = fmt.Sprintf("%v", component.APIVersions)
					}
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s %s %s api=%s policy=%s\n",
						component.Name, component.Version, health, apiVersions, component.APIPolicy); err != nil {
						return err
					}
				}
				return nil
			}
			if err := writeStyledKeyValues(cmd,
				styledKeyValue{Label: "Context", Value: output.Context},
				styledKeyValue{Label: "Target", Value: output.TargetURL},
				styledKeyValue{Label: "Kind", Value: output.TargetKind},
			); err != nil {
				return err
			}
			if len(output.Components) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No components found."))
				return err
			}
			rows := make([][]string, 0, len(output.Components))
			for _, component := range output.Components {
				health := "unhealthy"
				if component.Health {
					health = "healthy"
				}
				apiVersions := "<none>"
				if len(component.APIVersions) > 0 {
					apiVersions = fmt.Sprintf("%v", component.APIVersions)
				}
				rows = append(rows, []string{component.Name, component.Version, health, apiVersions, component.APIPolicy})
			}
			return writeStyledRows(cmd, []string{"Component", "Version", "Health", "API", "Policy"}, rows)
		},
	}
}

func newTargetProxyCommand() *cobra.Command {
	var port int
	var allowedOrigins []string

	command := &cobra.Command{
		Use:   "proxy",
		Short: "Start a local proxy for the current stack target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			if rt.Target.Kind != runtime.TargetKindStack && rt.Target.Kind != runtime.TargetKindCloudStack {
				return fmt.Errorf("target proxy requires a stack context")
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			transport := http.DefaultTransport
			if httpClient.Transport != nil {
				transport = httpClient.Transport
			}
			handler, err := targetcmd.NewProxyHandler(targetcmd.ProxyInput{
				TargetURL:      rt.Target.URL,
				Transport:      transport,
				AllowedOrigins: allowedOrigins,
				LogWriter:      cmd.OutOrStdout(),
				ErrorWriter:    cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				return fmt.Errorf("listen on port %d: %w", port, err)
			}
			defer listener.Close()

			server := &http.Server{
				Handler:           handler,
				ReadHeaderTimeout: 10 * time.Second,
			}
			serverErrors := make(chan error, 1)
			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					serverErrors <- err
				}
			}()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), styledInfoLine(cmd, "Proxy", fmt.Sprintf("http://%s -> %s", listener.Addr().String(), rt.Target.URL)))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "Press Ctrl+C to stop the server"))

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			select {
			case err := <-serverErrors:
				return err
			case <-ctx.Done():
			}

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("shutdown proxy: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, "Server stopped successfully."))
			return nil
		},
	}
	command.Flags().IntVar(&port, "port", 55001, "Local proxy port")
	command.Flags().StringSliceVar(&allowedOrigins, "allowed-origins", nil, "Allowed CORS origins; CORS is disabled when empty")
	return command
}

type targetInspectOutput struct {
	Context    string                   `json:"context" yaml:"context"`
	TargetURL  string                   `json:"targetUrl" yaml:"targetUrl"`
	TargetKind string                   `json:"targetKind" yaml:"targetKind"`
	Components []targetInspectComponent `json:"components" yaml:"components"`
}

type targetInspectComponent struct {
	Name        string   `json:"name" yaml:"name"`
	Version     string   `json:"version" yaml:"version"`
	Health      bool     `json:"health" yaml:"health"`
	APIVersions []string `json:"apiVersions" yaml:"apiVersions"`
	APIPolicy   string   `json:"apiPolicy" yaml:"apiPolicy"`
}

func apiVersionsToStrings(versions []capabilities.APIVersion) []string {
	ret := make([]string, len(versions))
	for i, version := range versions {
		ret[i] = string(version)
	}
	return ret
}
