package stack

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/v3/internal/membershipclient"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UpgradeStore struct {
	Stack *components.Stack
}

type UpgradeController struct {
	store *UpgradeStore
}

var _ fctl.Controller[*UpgradeStore] = (*UpgradeController)(nil)

func NewDefaultUpgradeStore() *UpgradeStore {
	return &UpgradeStore{
		Stack: &components.Stack{},
	}
}
func NewUpgradeController() *UpgradeController {
	return &UpgradeController{
		store: NewDefaultUpgradeStore(),
	}
}

func NewUpgradeCommand() *cobra.Command {
	return fctl.NewMembershipCommand("upgrade <stack-id> <version>",
		fctl.WithShortDescription("Upgrade a stack to specified version"),
		fctl.WithBoolFlag(nowaitFlag, false, "Wait stack availability"),
		fctl.WithArgs(cobra.RangeArgs(1, 2)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithController(NewUpgradeController()),
	)
}
func (c *UpgradeController) GetStore() *UpgradeStore {
	return c.store
}

func (c *UpgradeController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	getRequest := operations.GetStackRequest{
		OrganizationID: organizationID,
		StackID:        args[0],
	}

	stackResponse, err := apiClient.GetStack(cmd.Context(), getRequest)
	if err != nil {
		return nil, fmt.Errorf("retrieving stack: %w", err)
	}

	if stackResponse.GetHTTPMeta().Response.StatusCode > 300 {
		return nil, fmt.Errorf("unexpected status code: %d", stackResponse.GetHTTPMeta().Response.StatusCode)
	}

	if stackResponse.ReadStackResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	stackData := stackResponse.ReadStackResponse.GetData()

	req := components.StackVersion{
		Version: nil,
	}
	specifiedVersion := fctl.GetString(cmd, versionFlag)
	if specifiedVersion == "" {
		upgradeOpts, err := retrieveUpgradableVersion(cmd.Context(), organizationID, *stackData, apiClient)
		if err != nil {
			return nil, err
		}
		printer := pterm.DefaultInteractiveSelect.WithOptions(upgradeOpts)
		selectedOption, err := printer.Show("Please select a version")
		if err != nil {
			return nil, err
		}

		specifiedVersion = selectedOption
	}

	currentVersion := stackData.GetVersion()
	if currentVersion == nil {
		return nil, fmt.Errorf("stack has no version")
	}

	if specifiedVersion != *currentVersion {
		if !fctl.CheckStackApprobation(cmd, "Disclaimer: You are about to migrate the stack '%s' from '%s' to '%s'. It might take some time to fully migrate", stackData.GetName(), *currentVersion, specifiedVersion) {
			return nil, fctl.ErrMissingApproval
		}
	} else {
		pterm.Warning.WithWriter(cmd.OutOrStdout()).Printfln("Stack is already at version %s", specifiedVersion)
		return nil, nil
	}
	req.Version = pointer.For(specifiedVersion)

	upgradeRequest := operations.UpgradeStackRequest{
		OrganizationID: organizationID,
		StackID:        args[0],
		Body:           &req,
	}

	upgradeResponse, err := apiClient.UpgradeStack(cmd.Context(), upgradeRequest)
	if err != nil {
		return nil, fmt.Errorf("upgrading stack: %w", err)
	}

	if upgradeResponse.GetHTTPMeta().Response.StatusCode > 300 {
		return nil, fmt.Errorf("unexpected status code: %d", upgradeResponse.GetHTTPMeta().Response.StatusCode)
	}

	if !fctl.GetBool(cmd, nowaitFlag) {
		spinner, err := pterm.DefaultSpinner.Start("Waiting services availability")
		if err != nil {
			return nil, err
		}

		readyStack, err := waitStackReady(cmd, apiClient, stackData.GetOrganizationID(), stackData.GetID())
		if err != nil {
			return nil, err
		}
		c.store.Stack = readyStack

		if err := spinner.Stop(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *UpgradeController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Stack upgrade progressing.")
	return nil
}

func retrieveUpgradableVersion(ctx context.Context, organization string, stack components.Stack, apiClient *membershipclient.SDK) ([]string, error) {
	getVersionsRequest := operations.GetRegionVersionsRequest{
		OrganizationID: organization,
		RegionID:       stack.GetRegionID(),
	}

	availableVersionsResponse, err := apiClient.GetRegionVersions(ctx, getVersionsRequest)
	if err != nil {
		return nil, err
	}

	if availableVersionsResponse.GetRegionVersionsResponse == nil {
		return nil, fmt.Errorf("unexpected response: no versions data")
	}

	currentVersion := stack.GetVersion()
	if currentVersion == nil {
		return nil, fmt.Errorf("stack has no version")
	}

	var upgradeOptions []string
	for _, version := range availableVersionsResponse.GetRegionVersionsResponse.GetData() {
		versionName := version.GetName()
		if versionName == *currentVersion {
			continue
		}
		if !semver.IsValid(versionName) || semver.Compare(versionName, *currentVersion) >= 1 {
			upgradeOptions = append(upgradeOptions, versionName)
		}
	}
	return upgradeOptions, nil
}
