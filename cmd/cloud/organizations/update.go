package organizations

import (
	"github.com/formancehq/fctl/cmd/cloud/organizations/internal"
	"github.com/formancehq/go-libs/pointer"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

type UpdateController struct {
	store *DescribeStore
}

var _ fctl.Controller[*DescribeStore] = (*UpdateController)(nil)

func NewDefaultUpdateStore() *DescribeStore {
	return &DescribeStore{}
}

func NewUpdateController() *UpdateController {
	return &UpdateController{
		store: NewDefaultUpdateStore(),
	}
}

func NewUpdateCommand() *cobra.Command {
	return fctl.NewCommand("update <organizationId> --name <name> --default-policy-id <defaultPolicyID...>",
		fctl.WithAliases("update"),
		fctl.WithShortDescription("Update organization"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(fctl.OrganizationCompletion),
		fctl.WithConfirmFlag(),
		fctl.WithStringFlag("name", "", "Organization Name"),
		fctl.WithIntFlag("default-policy-id", 0, "Default policy id"),
		fctl.WithStringFlag("domain", "", "Organization Domain"),
		fctl.WithController(NewUpdateController()),
	)
}

func (c *UpdateController) GetStore() *DescribeStore {
	return c.store
}

func (c *UpdateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	store, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to update an organization") {
		return nil, fctl.ErrMissingApproval
	}

	org, _, err := store.DefaultAPI.ReadOrganization(cmd.Context(), args[0]).Execute()
	if err != nil {
		return nil, err
	}

	preparedData := membershipclient.OrganizationData{
		Name: func() string {
			if cmd.Flags().Changed("name") {
				return cmd.Flag("name").Value.String()
			}
			return org.Data.Name
		}(),
		DefaultPolicyID: func() membershipclient.NullableInt32 {
			if cmd.Flags().Changed("default-policy-id") {
				return *membershipclient.NewNullableInt32(
					pointer.For(int32(fctl.GetInt(cmd, "default-policy-id"))),
				)
			}
			return org.Data.DefaultPolicyID
		}(),
		Domain: func() *string {
			str := fctl.GetString(cmd, "domain")
			if str != "" {
				return &str
			}
			return org.Data.Domain
		}(),
	}

	response, _, err := store.DefaultAPI.
		UpdateOrganization(cmd.Context(), args[0]).
		OrganizationData(preparedData).
		Execute()

	if err != nil {
		return nil, err
	}

	c.store.OrganizationExpanded = response.Data

	return c, nil
}

func (c *UpdateController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintOrganization(c.store.OrganizationExpanded)
}
