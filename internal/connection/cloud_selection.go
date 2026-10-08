package connection

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/interactive"
)

// cloudCommandTarget follows the manifest's inherited target metadata. Cloud
// identity commands do not need a selected organization or stack.
func cloudCommandTarget(cmd *cobra.Command) string {
	for current := cmd; current != nil; current = current.Parent() {
		if target := current.Annotations["fctl.target"]; target != "" {
			return target
		}
	}
	return ""
}

// rootMembershipClient uses the existing coordinated authentication path. The
// context contains only IDs obtained from verified current identity claims.
func (s *Settings) rootMembershipClient(ctx context.Context, cmd *cobra.Command, base *http.Client, options Options, entry Entry, name, dir string) (*api.Client, cloud.IdentityInfo, error) {
	client, err := s.membershipClient(ctx, cmd, base, options, entry, name, dir)
	if err != nil {
		return nil, cloud.IdentityInfo{}, err
	}
	metadata := client.Context()
	var identity cloud.IdentityInfo
	if err := json.Unmarshal([]byte(metadata["organizations"]), &identity.Organizations); err != nil {
		return nil, cloud.IdentityInfo{}, fmt.Errorf("read verified Cloud organizations: %w", err)
	}
	if err := json.Unmarshal([]byte(metadata["stacks"]), &identity.Stacks); err != nil {
		return nil, cloud.IdentityInfo{}, fmt.Errorf("read verified Cloud stacks: %w", err)
	}
	return client, identity, nil
}

// cloudOrganizationClient also serves application grants, which require an
// organization but do not require a stack selection.
func (s *Settings) cloudOrganizationClient(ctx context.Context, cmd *cobra.Command, base *http.Client, options Options, entry Entry, name, dir string) (*api.Client, error) {
	client, identity, err := s.rootMembershipClient(ctx, cmd, base, options, entry, name, dir)
	if err != nil {
		return nil, err
	}
	metadata := client.Context()
	if metadata["organization"] != "" || !interactive.Enabled(cmd) {
		return client, nil
	}
	organization, err := chooseCloudOrganization(ctx, cmd, client, identity.Organizations)
	if err != nil {
		return nil, err
	}
	if organization == "" {
		return client, nil // The command retains its existing missing-target error.
	}
	if err := setCloudTargetFlag(cmd, "organization", organization); err != nil {
		return nil, err
	}
	metadata["organization"] = organization
	return client.WithContext(metadata), nil
}

func (s *Settings) selectCloudTarget(ctx context.Context, cmd *cobra.Command, base *http.Client, options Options, entry Entry, name, dir string) (Options, error) {
	if !interactive.Enabled(cmd) || (options.Organization != "" && options.Stack != "") || entry.Session == nil || entry.Session.Options.Stack != "" {
		return options, nil
	}
	client, identity, err := s.rootMembershipClient(ctx, cmd, base, options, entry, name, dir)
	if err != nil {
		return options, err
	}
	organization := options.Organization
	if organization == "" {
		organizations := organizationsForStack(identity, options.Stack)
		organization, err = chooseCloudOrganization(ctx, cmd, client, organizations)
		if err != nil || organization == "" {
			return options, err
		}
	}
	stack := options.Stack
	if stack == "" {
		stack, err = chooseCloudStack(ctx, cmd, client, organization, identity.Stacks[organization])
		if err != nil {
			return options, err
		}
	}
	return applyCloudTarget(cmd, options, organization, stack)
}

func applyCloudTarget(cmd *cobra.Command, options Options, organization, stack string) (Options, error) {
	if options.Organization == "" {
		if err := setCloudTargetFlag(cmd, "organization", organization); err != nil {
			return options, err
		}
		options.Organization = organization
	}
	if options.Stack == "" {
		if err := setCloudTargetFlag(cmd, "stack", stack); err != nil {
			return options, err
		}
		options.Stack = stack
	}
	return options, nil
}

func chooseCloudStack(ctx context.Context, cmd *cobra.Command, client *api.Client, organization string, allowed []string) (string, error) {
	resources, err := readCloudChoices(ctx, client, api.Path("organizations", organization, "stacks"))
	if err != nil {
		return "", err
	}
	choices := readyStackChoices(organization, allowed, resources)
	if len(choices) == 0 {
		return "", fmt.Errorf("no ready Cloud stacks are available; inspect fctl cloud stack list or set --stack STACK_ID")
	}
	return chooseCloudOption(ctx, cmd, "Choose a Cloud stack", choices)
}

func organizationsForStack(identity cloud.IdentityInfo, stack string) []string {
	var organizations []string
	for _, organization := range identity.Organizations {
		stacks := identity.Stacks[organization]
		if (stack == "" && len(stacks) > 0) || slices.Contains(stacks, stack) {
			organizations = append(organizations, organization)
		}
	}
	return organizations
}

func chooseCloudOrganization(ctx context.Context, cmd *cobra.Command, client *api.Client, organizations []string) (string, error) {
	if len(organizations) < 2 {
		return chooseCloudOption(ctx, cmd, "Choose a Cloud organization", organizationChoices(organizations, nil))
	}
	resources, err := readCloudChoices(ctx, client, "/organizations")
	if err != nil {
		return "", err
	}
	return chooseCloudOption(ctx, cmd, "Choose a Cloud organization", organizationChoices(organizations, resources))
}

type cloudChoiceResource struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	State          string `json:"state"`
	OrganizationID string `json:"organizationId"`
}

func readCloudChoices(ctx context.Context, client *api.Client, path string) ([]cloudChoiceResource, error) {
	data, err := client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []cloudChoiceResource `json:"data"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("read Cloud choices: %w", err)
	}
	return response.Data, nil
}

func organizationChoices(organizations []string, resources []cloudChoiceResource) []interactive.Option {
	names := make(map[string]string, len(resources))
	for _, resource := range resources {
		names[resource.ID] = resource.Name
	}
	choices := make([]interactive.Option, 0, len(organizations))
	for _, id := range organizations {
		label := id
		if name := names[id]; name != "" {
			label = fmt.Sprintf("%s (%s)", name, id)
		}
		choices = append(choices, interactive.Option{Label: label, Value: id})
	}
	sortCloudChoices(choices)
	return choices
}

func readyStackChoices(organization string, allowed []string, resources []cloudChoiceResource) []interactive.Option {
	var choices []interactive.Option
	seen := make(map[string]bool)
	for _, resource := range resources {
		if resource.Status != "READY" || (resource.State != "" && resource.State != "ACTIVE") ||
			(resource.OrganizationID != "" && resource.OrganizationID != organization) || !slices.Contains(allowed, resource.ID) || seen[resource.ID] {
			continue
		}
		seen[resource.ID] = true
		choices = append(choices, interactive.Option{
			Label: fmt.Sprintf("%s (%s) · %s", cmp.Or(resource.Name, resource.ID), resource.ID, resource.Status),
			Value: resource.ID,
		})
	}
	sortCloudChoices(choices)
	return choices
}

func sortCloudChoices(choices []interactive.Option) {
	slices.SortFunc(choices, func(a, b interactive.Option) int {
		return cmp.Or(cmp.Compare(a.Label, b.Label), cmp.Compare(a.Value, b.Value))
	})
}

func chooseCloudOption(ctx context.Context, cmd *cobra.Command, title string, choices []interactive.Option) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	switch len(choices) {
	case 0:
		return "", nil
	case 1:
		return choices[0].Value, nil
	default:
		return interactive.Select(ctx, cmd, title, choices)
	}
}

func setCloudTargetFlag(cmd *cobra.Command, name, value string) error {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Flags().Lookup(name) != nil {
			return current.Flags().Set(name, value)
		}
		if current.PersistentFlags().Lookup(name) != nil {
			return current.PersistentFlags().Set(name, value)
		}
	}
	return fmt.Errorf("cloud target flag --%s is unavailable", name)
}
