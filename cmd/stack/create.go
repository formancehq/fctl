package stack

import (
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/pointer"

	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const (
	regionFlag  = "region"
	nowaitFlag  = "no-wait"
	versionFlag = "version"
)

type CreateStore struct {
	Stack    *membershipclient.Stack
	Versions *shared.GetVersionsResponse
}

type CreateController struct {
	store *CreateStore
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultStackCreateStore() *CreateStore {
	return &CreateStore{
		Stack:    &membershipclient.Stack{},
		Versions: &shared.GetVersionsResponse{},
	}
}
func NewStackCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultStackCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewMembershipCommand("create [name]",
		fctl.WithShortDescription("Create a new stack"),
		fctl.WithAliases("c", "cr"),
		fctl.WithArgs(cobra.RangeArgs(0, 1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(regionFlag, "", "Region on which deploy the stack"),
		fctl.WithStringFlag(versionFlag, "", "Version of the stack"),
		fctl.WithBoolFlag(nowaitFlag, false, "Not wait stack availability"),
		fctl.WithController(NewStackCreateController()),
	)
}
func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	var err error
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

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	} else {
		name, err = pterm.DefaultInteractiveTextInput.WithMultiLine(false).Show("Enter a name")
		if err != nil {
			return nil, err
		}
	}

	region := fctl.GetString(cmd, regionFlag)
	if region == "" {
		regions, _, err := apiClient.DefaultAPI.ListRegions(cmd.Context(), organizationID).Execute()
		if err != nil {
			return nil, errors.Wrap(err, "listing regions")
		}

		var options []string
		for _, region := range regions.Data {
			privacy := "Private"
			if region.Public {
				privacy = "Public "
			}
			name := "<noname>"
			if region.Name != "" {
				name = region.Name
			}
			options = append(options, fmt.Sprintf("%s | %s | %s", region.Id, privacy, name))
		}

		if len(options) == 0 {
			return nil, errors.New("no regions available")
		}

		printer := pterm.DefaultInteractiveSelect.WithOptions(options)
		selectedOption, err := printer.Show("Please select a region")
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(options); i++ {
			if selectedOption == options[i] {
				region = regions.Data[i].Id
				break
			}
		}
	}

	req := membershipclient.CreateStackRequest{
		Name:     name,
		RegionID: region,
	}

	availableVersions, httpResponse, err := apiClient.DefaultAPI.GetRegionVersions(cmd.Context(), organizationID, region).Execute()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving available versions")
	}

	if httpResponse.StatusCode > 300 {
		return nil, err
	}

	specifiedVersion := fctl.GetString(cmd, versionFlag)
	if specifiedVersion == "" {
		var options []string
		for _, version := range availableVersions.Data {
			options = append(options, version.Name)
		}

		selectedOption := ""
		if len(options) > 0 {
			printer := pterm.DefaultInteractiveSelect.WithOptions(options)
			selectedOption, err = printer.Show("Please select a version")
			if err != nil {
				return nil, err
			}
		}

		specifiedVersion = selectedOption
	}
	req.Version = pointer.For(specifiedVersion)

	stackResponse, _, err := apiClient.DefaultAPI.
		CreateStack(cmd.Context(), organizationID).
		CreateStackRequest(req).
		Execute()
	if err != nil {
		return nil, errors.Wrap(err, "creating stack")
	}

	if !fctl.GetBool(cmd, nowaitFlag) {
		spinner, err := pterm.DefaultSpinner.Start("Waiting services availability")
		if err != nil {
			return nil, err
		}

		stack, err := waitStackReady(cmd, *profile, apiClient, stackResponse.Data.OrganizationId, stackResponse.Data.Id)
		if err != nil {
			return nil, err
		}
		c.store.Stack = stack

		if err := spinner.Stop(); err != nil {
			return nil, err
		}
	} else {
		c.store.Stack = stackResponse.Data
	}

	portal := fctl.DefaultConsoleURL
	serverInfo, err := fctl.MembershipServerInfo(cmd.Context(), apiClient.DefaultAPI)
	if err != nil {
		return nil, err
	}
	if v := serverInfo.ConsoleURL; v != nil {
		portal = *v
	}

	fctl.BasicTextCyan.WithWriter(cmd.OutOrStdout()).Println("Your portal will be reachable on: " + portal)

	// todo: need a long running client with auto refresh
	stackClient, err := fctl.NewStackClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		cfg.CurrentProfile,
		*profile,
		stackResponse.Data.OrganizationId,
		stackResponse.Data.Id,
	)
	if err != nil {
		return nil, err
	}

	versions, err := stackClient.GetVersions(cmd.Context())
	if err != nil {
		return nil, err
	}
	if versions.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when reading versions", versions.StatusCode)
	}

	c.store.Versions = versions.GetVersionsResponse

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, _ []string) error {
	return internal.PrintStackInformation(cmd.OutOrStdout(), c.store.Stack, c.store.Versions)
}
