package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	webhookscmd "github.com/formancehq/fctl/v4/internal/commands/webhooks"
)

func newWebhooksCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "webhooks",
		Short: "Manage webhooks",
	}
	command.AddCommand(newWebhooksCreateCommand())
	command.AddCommand(newWebhooksListCommand())
	command.AddCommand(newWebhooksActivateCommand())
	command.AddCommand(newWebhooksDeactivateCommand())
	command.AddCommand(newWebhooksDeleteCommand())
	command.AddCommand(newWebhooksSecretCommand())
	command.AddCommand(newWebhooksChangeSecretCommand())
	return command
}

func newWebhooksSecretCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "secret",
		Short: "Manage webhook secrets",
	}
	command.AddCommand(newWebhooksSecretRotateCommand("rotate", nil, false))
	return command
}

func newWebhooksCreateCommand() *cobra.Command {
	var name string
	var secret string
	var secretStdin bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "create <endpoint> <event-type>...",
		Short: "Create a webhook config",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretValue, err := webhookSecretValue(cmd, secret, secretStdin, false)
			if err != nil {
				return err
			}
			var namePtr *string
			if name != "" {
				namePtr = &name
			}
			var secretPtr *string
			if secretValue != "" {
				secretPtr = &secretValue
			}
			service, err := newInsertWebhookConfigService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), webhookscmd.ConfigInput{
				Endpoint:   args[0],
				EventTypes: args[1:],
				Name:       namePtr,
				Secret:     secretPtr,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWebhookConfigMutated(cmd, output, "created")
		},
	}
	command.Flags().StringVar(&name, "name", "", "Webhook config name")
	command.Flags().StringVar(&secret, "secret", "", "Webhook signing secret")
	command.Flags().BoolVar(&secretStdin, "secret-stdin", false, "Read webhook signing secret from stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin webhooks API version")
	return command
}

func newWebhooksListCommand() *cobra.Command {
	var configID string
	var endpoint string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List webhook configs",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := newListWebhookConfigsService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), webhookscmd.ListConfigsInput{
				ConfigID: configID,
				Endpoint: endpoint,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWebhookConfigs(cmd, output)
		},
	}
	command.Flags().StringVar(&configID, "id", "", "Filter by config ID")
	command.Flags().StringVar(&endpoint, "endpoint", "", "Filter by endpoint URL")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin webhooks API version")
	return command
}

func newWebhooksActivateCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "activate <config-id>",
		Short: "Activate a webhook config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newActivateWebhookConfigService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), webhookscmd.ConfigIDInput{ConfigID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWebhookConfigMutated(cmd, output, "activated")
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin webhooks API version")
	return command
}

func newWebhooksDeactivateCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "deactivate <config-id>",
		Short: "Deactivate a webhook config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("webhooks deactivate requires --confirm")
			}
			service, err := newDeactivateWebhookConfigService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), webhookscmd.ConfigIDInput{ConfigID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWebhookConfigMutated(cmd, output, "deactivated")
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm webhook deactivation")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin webhooks API version")
	return command
}

func newWebhooksDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <config-id>",
		Short: "Delete a webhook config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("webhooks delete requires --confirm")
			}
			service, err := newDeleteWebhookConfigService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), webhookscmd.ConfigIDInput{ConfigID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWebhookConfigDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm webhook deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin webhooks API version")
	return command
}

func newWebhooksSecretRotateCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var secret string
	var secretStdin bool
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <config-id>",
		Aliases: aliases,
		Short:   "Rotate a webhook signing secret",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command webhooks change-secret has been deprecated, use webhooks secret rotate <config-id> --secret-stdin")
			}
			positionalSecret := ""
			if len(args) == 2 {
				positionalSecret = args[1]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional secret has been deprecated, use --secret or --secret-stdin")
			}
			secretValue, err := webhookSecretValue(cmd, firstNonEmpty(secret, positionalSecret), secretStdin, true)
			if err != nil {
				return err
			}
			service, err := newChangeWebhookSecretService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), webhookscmd.ChangeSecretInput{ConfigID: args[0], Secret: secretValue})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWebhookConfigMutated(cmd, output, "secret rotated")
		},
	}
	if deprecated {
		command.Deprecated = "use webhooks secret rotate"
	}
	command.Flags().StringVar(&secret, "secret", "", "Webhook signing secret")
	command.Flags().BoolVar(&secretStdin, "secret-stdin", false, "Read webhook signing secret from stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin webhooks API version")
	return command
}

func newWebhooksChangeSecretCommand() *cobra.Command {
	command := newWebhooksSecretRotateCommand("change-secret", nil, true)
	command.Use = "change-secret <config-id>"
	return command
}

func newListWebhookConfigsService(cmd *cobra.Command, apiVersion string) (webhookscmd.ListConfigsService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return webhookscmd.ListConfigsService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return webhookscmd.ListConfigsService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return webhookscmd.ListConfigsService{
		Handlers: webhookscmd.SDKListConfigsHandlers(sdk),
		Resolve:  webhooksVersionResolver(rt, webhookscmd.FeatureGetManyConfigs, apiVersion),
	}, nil
}

func newInsertWebhookConfigService(cmd *cobra.Command, apiVersion string) (webhookscmd.InsertConfigService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return webhookscmd.InsertConfigService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return webhookscmd.InsertConfigService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return webhookscmd.InsertConfigService{
		Handlers: webhookscmd.SDKInsertConfigHandlers(sdk),
		Resolve:  webhooksVersionResolver(rt, webhookscmd.FeatureInsertConfig, apiVersion),
	}, nil
}

func newActivateWebhookConfigService(cmd *cobra.Command, apiVersion string) (webhookscmd.ActivateConfigService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return webhookscmd.ActivateConfigService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return webhookscmd.ActivateConfigService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return webhookscmd.ActivateConfigService{
		Handlers: webhookscmd.SDKActivateConfigHandlers(sdk),
		Resolve:  webhooksVersionResolver(rt, webhookscmd.FeatureActivateConfig, apiVersion),
	}, nil
}

func newDeactivateWebhookConfigService(cmd *cobra.Command, apiVersion string) (webhookscmd.DeactivateConfigService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return webhookscmd.DeactivateConfigService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return webhookscmd.DeactivateConfigService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return webhookscmd.DeactivateConfigService{
		Handlers: webhookscmd.SDKDeactivateConfigHandlers(sdk),
		Resolve:  webhooksVersionResolver(rt, webhookscmd.FeatureDeactivateConfig, apiVersion),
	}, nil
}

func newDeleteWebhookConfigService(cmd *cobra.Command, apiVersion string) (webhookscmd.DeleteConfigService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return webhookscmd.DeleteConfigService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return webhookscmd.DeleteConfigService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return webhookscmd.DeleteConfigService{
		Handlers: webhookscmd.SDKDeleteConfigHandlers(sdk),
		Resolve:  webhooksVersionResolver(rt, webhookscmd.FeatureDeleteConfig, apiVersion),
	}, nil
}

func newChangeWebhookSecretService(cmd *cobra.Command, apiVersion string) (webhookscmd.ChangeSecretService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return webhookscmd.ChangeSecretService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return webhookscmd.ChangeSecretService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return webhookscmd.ChangeSecretService{
		Handlers: webhookscmd.SDKChangeSecretHandlers(sdk),
		Resolve:  webhooksVersionResolver(rt, webhookscmd.FeatureChangeConfigSecret, apiVersion),
	}, nil
}

func webhooksVersionResolver(rt interface {
	ResolveAPIVersion(context.Context, capabilities.VersionResolutionRequest) (capabilities.APIVersion, error)
}, feature capabilities.Feature, apiVersion string) func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
	return func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
		request := capabilities.VersionResolutionRequest{
			Product:         webhookscmd.ProductWebhooks,
			Feature:         feature,
			HandlerVersions: handlerVersions,
		}
		if apiVersion != "" {
			request.Policy = capabilities.VersionPolicyPinned
			request.PinnedVersion = capabilities.APIVersion(apiVersion)
		}
		return rt.ResolveAPIVersion(ctx, request)
	}
}

func webhookSecretValue(cmd *cobra.Command, secret string, secretStdin bool, required bool) (string, error) {
	if secret != "" && secretStdin {
		return "", fmt.Errorf("use either --secret or --secret-stdin, not both")
	}
	if secretStdin {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("read secret from stdin: %w", err)
		}
		secret = strings.TrimSpace(string(data))
	}
	if required && secret == "" {
		return "", fmt.Errorf("webhook secret is required")
	}
	return secret, nil
}

func renderWebhookConfigs(cmd *cobra.Command, output webhookscmd.ListConfigsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Configs) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No webhook configs found."))
		return err
	}
	for _, config := range output.Configs {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%t\t%s\n", config.ID, config.Endpoint, config.Active, strings.Join(config.EventTypes, ",")); err != nil {
			return err
		}
	}
	if output.HasMore {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "More webhook configs are available.")
		return err
	}
	return nil
}

func renderWebhookConfigMutated(cmd *cobra.Command, output webhookscmd.ConfigOutput, action string) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Webhook config %s %s.\n", output.Config.ID, action)
	return err
}

func renderWebhookConfigDeleted(cmd *cobra.Command, output webhookscmd.DeleteConfigOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Webhook config %s deleted.\n", output.ConfigID)
	return err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func formatWebhookTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
