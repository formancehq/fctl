package stack

import (
	"fmt"
	"net/http"

	"errors"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/v3/pointer"
)

const (
	regionFlag  = "region"
	nowaitFlag  = "no-wait"
	versionFlag = "version"
)

type CreateStore struct {
	Stack    *components.Stack
	Versions *shared.GetVersionsResponse
}

type CreateController struct {
	store *CreateStore
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultStackCreateStore() *CreateStore {
	return &CreateStore{
		Stack:    &components.Stack{},
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
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
		listRegionsRequest := operations.ListRegionsRequest{
			OrganizationID: organizationID,
		}

		regionsResponse, err := apiClient.ListRegions(cmd.Context(), listRegionsRequest)
		if err != nil {
			return nil, fmt.Errorf("listing regions: %w", err)
		}

		if regionsResponse.ListRegionsResponse == nil {
			return nil, fmt.Errorf("unexpected response: no data")
		}

		var options []string
		for _, regionItem := range regionsResponse.ListRegionsResponse.GetData() {
			privacy := "Private"
			if regionItem.GetPublic() {
				privacy = "Public "
			}
			name := "<noname>"
			if regionName := regionItem.GetName(); regionName != "" {
				name = regionName
			}
			options = append(options, fmt.Sprintf("%s | %s | %s", regionItem.GetID(), privacy, name))
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
				region = regionsResponse.ListRegionsResponse.GetData()[i].GetID()
				break
			}
		}
	}

	req := components.CreateStackRequest{
		Name:     name,
		RegionID: region,
	}

	getVersionsRequest := operations.GetRegionVersionsRequest{
		OrganizationID: organizationID,
		RegionID:       region,
	}

	availableVersionsResponse, err := apiClient.GetRegionVersions(cmd.Context(), getVersionsRequest)
	if err != nil {
		return nil, fmt.Errorf("retrieving available versions: %w", err)
	}

	if availableVersionsResponse.GetRegionVersionsResponse == nil {
		return nil, fmt.Errorf("unexpected response: no versions data")
	}

	specifiedVersion := fctl.GetString(cmd, versionFlag)
	if specifiedVersion == "" {
		var options []string
		for _, version := range availableVersionsResponse.GetRegionVersionsResponse.GetData() {
			options = append(options, version.GetName())
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

	createStackRequest := operations.CreateStackRequest{
		OrganizationID: organizationID,
		Body:           &req,
	}

	stackResponse, err := apiClient.CreateStack(cmd.Context(), createStackRequest)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	if stackResponse.CreateStackResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	stackData := stackResponse.CreateStackResponse.GetData()

	if stackData == nil {
		return nil, fmt.Errorf("unexpected response: stack data is nil")
	}

	if !fctl.GetBool(cmd, nowaitFlag) {
		spinner, err := pterm.DefaultSpinner.Start("Waiting services availability")
		if err != nil {
			return nil, err
		}

		stack, err := waitStackReady(cmd, apiClient, stackData.GetOrganizationID(), stackData.GetID())
		if err != nil {
			return nil, err
		}
		c.store.Stack = stack

		if err := spinner.Stop(); err != nil {
			return nil, err
		}
	} else {
		c.store.Stack = stackData
	}

	portal := fctl.DefaultConsoleURL
	serverInfo, err := fctl.MembershipServerInfo(cmd.Context(), apiClient)
	if err != nil {
		return nil, err
	}
	if v := serverInfo.GetConsoleURL(); v != nil {
		portal = *v
	}

	fctl.BasicTextCyan.WithWriter(cmd.OutOrStdout()).Println("Your portal will be reachable on: " + portal)

	// todo: need a long running client with auto refresh
	stackClient, err := fctl.NewStackClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		organizationID,
		stackData.GetID(),
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
