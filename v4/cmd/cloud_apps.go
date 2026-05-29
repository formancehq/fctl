package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	deployserver "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"
	"github.com/spf13/cobra"

	cloudcmd "github.com/formancehq/fctl/v4/internal/commands/cloud"
	"github.com/formancehq/fctl/v4/internal/runtime"
)

const defaultDeployURL = "https://deploy.formance.cloud"

func newCloudAppsCommand() *cobra.Command {
	var deployURL string
	var organizationID string

	command := &cobra.Command{
		Use:     "apps",
		Aliases: []string{"app"},
		Short:   "Manage Cloud apps",
		Hidden:  true,
	}
	command.PersistentFlags().StringVar(&deployURL, "deploy-url", defaultDeployURL, "Cloud apps deploy server URL")
	command.PersistentFlags().StringVar(&organizationID, "organization", "", "Cloud organization ID")
	command.AddCommand(newCloudAppsListCommand(&deployURL, &organizationID))
	command.AddCommand(newCloudAppsCreateCommand(&deployURL, &organizationID))
	command.AddCommand(newCloudAppsShowCommand(&deployURL, &organizationID))
	command.AddCommand(newCloudAppsDeleteCommand(&deployURL, &organizationID))
	command.AddCommand(newCloudAppsDeployCommand(&deployURL))
	command.AddCommand(newCloudAppsRunsCommand(&deployURL))
	command.AddCommand(newCloudAppsVersionsCommand(&deployURL))
	command.AddCommand(newCloudAppsVariablesCommand(&deployURL))
	return command
}

func newCloudAppsListCommand(deployURL *string, organizationID *string) *cobra.Command {
	var page int64
	var pageSize int64

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Cloud apps",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListCloudAppsService{Client: client}.Run(cmd.Context(), cloudcmd.ListCloudAppsInput{
				OrganizationID: resolveCloudOrganizationID(rt, *organizationID),
				Page:           page,
				PageSize:       pageSize,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudApps(cmd, output)
		},
	}
	command.Flags().Int64Var(&page, "page", 1, "Page number")
	command.Flags().Int64Var(&pageSize, "page-size", 100, "Page size")
	return command
}

func newCloudAppsCreateCommand(deployURL *string, organizationID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a Cloud app",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreateCloudAppService{Client: client}.Run(cmd.Context(), resolveCloudOrganizationID(rt, *organizationID))
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud app %s created.\n", output.App.ID)
			return err
		},
	}
}

func newCloudAppsShowCommand(deployURL *string, organizationID *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <app-id>",
		Short: "Show a Cloud app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.GetCloudAppService{Client: client}.Run(cmd.Context(), cloudcmd.CloudAppInput{
				OrganizationID: resolveCloudOrganizationID(rt, *organizationID),
				AppID:          args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudApp(cmd, output)
		},
	}
}

func newCloudAppsDeleteCommand(deployURL *string, organizationID *string) *cobra.Command {
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <app-id>",
		Short: "Delete a Cloud app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud apps delete requires --confirm")
			}
			rt, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteCloudAppService{Client: client}.Run(cmd.Context(), cloudcmd.CloudAppInput{
				OrganizationID: resolveCloudOrganizationID(rt, *organizationID),
				AppID:          args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud app %s deleted.\n", output.AppID)
			return err
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud app deletion")
	return command
}

func newCloudAppsDeployCommand(deployURL *string) *cobra.Command {
	var file string

	command := &cobra.Command{
		Use:   "deploy <app-id>",
		Short: "Deploy a Cloud app manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("cloud apps deploy requires --file")
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeployCloudAppService{Client: client}.Run(cmd.Context(), args[0], data)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud app deployment accepted with run %s.\n", output.Run.ID)
			return err
		},
	}
	command.Flags().StringVar(&file, "file", "", "Path to manifest file")
	command.Flags().StringVar(&file, "path", "", "Deprecated alias for --file")
	_ = command.Flags().MarkDeprecated("path", "use --file instead")
	return command
}

func newCloudAppsRunsCommand(deployURL *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "runs",
		Short: "Manage Cloud app runs",
	}
	command.AddCommand(newCloudAppsRunsListCommand(deployURL))
	command.AddCommand(newCloudAppsRunsShowCommand(deployURL))
	command.AddCommand(newCloudAppsRunsLogsCommand(deployURL))
	return command
}

func newCloudAppsRunsListCommand(deployURL *string) *cobra.Command {
	var page int64
	var pageSize int64

	command := &cobra.Command{
		Use:     "list <app-id>",
		Aliases: []string{"ls"},
		Short:   "List Cloud app runs",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListCloudRunsService{Client: client}.Run(cmd.Context(), args[0], page, pageSize)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudRuns(cmd, output)
		},
	}
	command.Flags().Int64Var(&page, "page", 1, "Page number")
	command.Flags().Int64Var(&pageSize, "page-size", 100, "Page size")
	return command
}

func newCloudAppsRunsShowCommand(deployURL *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <run-id>",
		Short: "Show a Cloud app run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.GetCloudRunService{Client: client}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudRun(cmd, output)
		},
	}
}

func newCloudAppsRunsLogsCommand(deployURL *string) *cobra.Command {
	return &cobra.Command{
		Use:   "logs <run-id>",
		Short: "Show Cloud app run logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.GetCloudRunLogsService{Client: client}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudRunLogs(cmd, output)
		},
	}
}

func newCloudAppsVersionsCommand(deployURL *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "versions",
		Short: "Manage Cloud app versions",
	}
	command.AddCommand(newCloudAppsVersionsListCommand(deployURL))
	command.AddCommand(newCloudAppsVersionsShowCommand(deployURL))
	command.AddCommand(newCloudAppsVersionsManifestCommand(deployURL, "manifest"))
	command.AddCommand(newCloudAppsVersionsManifestCommand(deployURL, "show-manifest"))
	command.AddCommand(newCloudAppsVersionsArchiveCommand(deployURL))
	command.AddCommand(newCloudAppsVersionsArchiveShowCommand(deployURL, "show-archive", true))
	return command
}

func newCloudAppsVersionsListCommand(deployURL *string) *cobra.Command {
	var page int64
	var pageSize int64

	command := &cobra.Command{
		Use:     "list <app-id>",
		Aliases: []string{"ls"},
		Short:   "List Cloud app versions",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListCloudVersionsService{Client: client}.Run(cmd.Context(), args[0], page, pageSize)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudVersions(cmd, output)
		},
	}
	command.Flags().Int64Var(&page, "page", 1, "Page number")
	command.Flags().Int64Var(&pageSize, "page-size", 100, "Page size")
	return command
}

func newCloudAppsVersionsShowCommand(deployURL *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <version-id>",
		Short: "Show a Cloud app version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.GetCloudVersionService{Client: client}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudVersion(cmd, output)
		},
	}
}

func newCloudAppsVersionsManifestCommand(deployURL *string, use string) *cobra.Command {
	command := &cobra.Command{
		Use:   use + " <version-id>",
		Short: "Show a Cloud app version manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if use == "show-manifest" {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command cloud apps versions show-manifest has been deprecated, use cloud apps versions manifest")
			}
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.GetCloudVersionBlobService{
				Client: client,
				Accept: operations.AcceptHeaderEnumApplicationYaml,
			}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(output.Data)
			return err
		},
	}
	if use == "show-manifest" {
		command.Hidden = true
	}
	return command
}

func newCloudAppsVersionsArchiveCommand(deployURL *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "archive",
		Short: "Manage Cloud app version archives",
	}
	command.AddCommand(newCloudAppsVersionsArchiveShowCommand(deployURL, "show", false))
	return command
}

func newCloudAppsVersionsArchiveShowCommand(deployURL *string, use string, deprecated bool) *cobra.Command {
	command := &cobra.Command{
		Use:   use + " <version-id>",
		Short: "Show a Cloud app version archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command cloud apps versions show-archive has been deprecated, use cloud apps versions archive show")
			}
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.GetCloudVersionBlobService{
				Client: client,
				Accept: operations.AcceptHeaderEnumApplicationGzip,
			}.Run(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(output.Data)
			return err
		},
	}
	if deprecated {
		command.Hidden = true
	}
	return command
}

func newCloudAppsVariablesCommand(deployURL *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "variables",
		Short: "Manage Cloud app variables",
	}
	command.AddCommand(newCloudAppsVariablesListCommand(deployURL))
	command.AddCommand(newCloudAppsVariablesCreateCommand(deployURL))
	command.AddCommand(newCloudAppsVariablesDeleteCommand(deployURL))
	return command
}

func newCloudAppsVariablesListCommand(deployURL *string) *cobra.Command {
	var page int64
	var pageSize int64

	command := &cobra.Command{
		Use:     "list <app-id>",
		Aliases: []string{"ls"},
		Short:   "List Cloud app variables",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.ListCloudVariablesService{Client: client}.Run(cmd.Context(), args[0], page, pageSize)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudVariables(cmd, output)
		},
	}
	command.Flags().Int64Var(&page, "page", 1, "Page number")
	command.Flags().Int64Var(&pageSize, "page-size", 100, "Page size")
	return command
}

func newCloudAppsVariablesCreateCommand(deployURL *string) *cobra.Command {
	var key string
	var value string
	var valueStdin bool
	var description string
	var sensitive bool

	command := &cobra.Command{
		Use:   "create <app-id>",
		Short: "Create a Cloud app variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" {
				return fmt.Errorf("cloud apps variables create requires --key")
			}
			if value != "" && valueStdin {
				return fmt.Errorf("cloud apps variables create accepts only one of --value or --value-stdin")
			}
			if valueStdin {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				value = strings.TrimRight(string(data), "\r\n")
			}
			if value == "" {
				return fmt.Errorf("cloud apps variables create requires --value")
			}
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.CreateCloudVariableService{Client: client}.Run(cmd.Context(), cloudcmd.CloudVariableInput{
				AppID:       args[0],
				Key:         key,
				Value:       value,
				Description: description,
				Sensitive:   sensitive,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderCloudVariable(cmd, output)
		},
	}
	command.Flags().StringVar(&key, "key", "", "Variable key")
	command.Flags().StringVar(&value, "value", "", "Variable value")
	command.Flags().BoolVar(&valueStdin, "value-stdin", false, "Read variable value from stdin")
	command.Flags().StringVar(&description, "description", "", "Variable description")
	command.Flags().BoolVar(&sensitive, "sensitive", true, "Mark variable as sensitive")
	return command
}

func newCloudAppsVariablesDeleteCommand(deployURL *string) *cobra.Command {
	var confirm bool

	command := &cobra.Command{
		Use:   "delete <app-id> <variable-id>",
		Short: "Delete a Cloud app variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("cloud apps variables delete requires --confirm")
			}
			_, client, err := deployClientFromCommand(cmd, *deployURL)
			if err != nil {
				return err
			}
			output, err := cloudcmd.DeleteCloudVariableService{Client: client}.Run(cmd.Context(), cloudcmd.CloudVariableInput{
				AppID:      args[0],
				VariableID: args[1],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Cloud app variable %s deleted.\n", output.VariableID)
			return err
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm Cloud app variable deletion")
	return command
}

func cloudRuntimeFromCommand(cmd *cobra.Command) (*runtime.Runtime, *http.Client, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return nil, nil, err
	}
	if rt.Target.Kind != runtime.TargetKindCloud && rt.Target.Kind != runtime.TargetKindCloudStack {
		return nil, nil, fmt.Errorf("cloud commands require a cloud or cloud-stack context")
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return nil, nil, err
	}
	return rt, httpClient, nil
}

func deployClientFromCommand(cmd *cobra.Command, deployURL string) (*runtime.Runtime, *deployserver.DeployServer, error) {
	rt, httpClient, err := cloudRuntimeFromCommand(cmd)
	if err != nil {
		return nil, nil, err
	}
	if deployURL == "" {
		deployURL = defaultDeployURL
	}
	options := []deployserver.SDKOption{deployserver.WithServerURL(deployURL)}
	if httpClient != nil {
		options = append(options, deployserver.WithClient(httpClient))
	}
	return rt, deployserver.New(options...), nil
}

func renderCloudApps(cmd *cobra.Command, output cloudcmd.ListCloudAppsOutput) error {
	if len(output.Apps) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No Cloud apps found."))
		return err
	}
	for _, app := range output.Apps {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", app.ID, app.Name, valueOrNA(app.RunStatus), valueOrNA(app.CurrentConfigurationVersionID)); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudApp(cmd *cobra.Command, output cloudcmd.CloudAppOutput) error {
	app := output.App
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nName\t%s\nRunStatus\t%s\nCurrentConfigurationVersionID\t%s\n", app.ID, app.Name, valueOrNA(app.RunStatus), valueOrNA(app.CurrentConfigurationVersionID)); err != nil {
		return err
	}
	for key, value := range app.StateStack {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "State\t%s\t%v\n", key, value); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudRuns(cmd *cobra.Command, output cloudcmd.ListCloudRunsOutput) error {
	if len(output.Runs) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No Cloud app runs found."))
		return err
	}
	for _, run := range output.Runs {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", run.ID, run.Status, run.ConfigurationVersionID, run.Message); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudRun(cmd *cobra.Command, output cloudcmd.CloudRunOutput) error {
	run := output.Run
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nStatus\t%s\nConfigurationVersionID\t%s\nMessage\t%s\nCreatedAt\t%s\n", run.ID, run.Status, run.ConfigurationVersionID, run.Message, run.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	return err
}

func renderCloudRunLogs(cmd *cobra.Command, output cloudcmd.CloudRunLogsOutput) error {
	for _, log := range output.Logs {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", log.Timestamp, log.Module, log.Message, log.Severity); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudVersions(cmd *cobra.Command, output cloudcmd.ListCloudVersionsOutput) error {
	if len(output.Versions) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No Cloud app versions found."))
		return err
	}
	for _, version := range output.Versions {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%t\t%s\n", version.ID, version.Status, version.AutoQueueRuns, version.ErrorMessage); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudVersion(cmd *cobra.Command, output cloudcmd.CloudVersionOutput) error {
	version := output.Version
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nStatus\t%s\nAutoQueueRuns\t%t\nError\t%s\nErrorMessage\t%s\n", version.ID, version.Status, version.AutoQueueRuns, version.Error, version.ErrorMessage)
	return err
}

func renderCloudVariables(cmd *cobra.Command, output cloudcmd.ListCloudVariablesOutput) error {
	if len(output.Variables) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No Cloud app variables found."))
		return err
	}
	for _, variable := range output.Variables {
		value := variable.Value
		if variable.Sensitive {
			value = "****"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", variable.ID, variable.Key, value, variable.Description); err != nil {
			return err
		}
	}
	return nil
}

func renderCloudVariable(cmd *cobra.Command, output cloudcmd.CloudVariableOutput) error {
	variable := output.Variable
	value := variable.Value
	if variable.Sensitive {
		value = "****"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\nKey\t%s\nValue\t%s\nDescription\t%s\nSensitive\t%t\n", variable.ID, variable.Key, value, variable.Description, variable.Sensitive)
	return err
}

func valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}
