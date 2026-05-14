package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

func newTargetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "target",
		Short: "Inspect the active fctl v4 target",
	}
	command.AddCommand(newTargetInspectCommand())
	return command
}

func newTargetInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the current target and inferred capabilities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := runtimeFromCommand(cmd)
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
		},
	}
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
