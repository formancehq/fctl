package stack

import (
	"fmt"
	"time"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const (
	allFlag     = "all"
	deletedFlag = "deleted"
)

type Stack struct {
	Id           string  `json:"id"`
	Name         string  `json:"name"`
	Dashboard    string  `json:"dashboard"`
	RegionID     string  `json:"region"`
	DisabledAt   *string `json:"disabledAt"`
	DeletedAt    *string `json:"deletedAt"`
	AuditEnabled string  `json:"auditEnabled"`
	Status       string  `json:"status"`
}
type StackListStore struct {
	Stacks []Stack `json:"stacks"`
}

type StackListController struct {
	store *StackListStore
}

var _ fctl.Controller[*StackListStore] = (*StackListController)(nil)

func NewDefaultStackListStore() *StackListStore {
	return &StackListStore{
		Stacks: []Stack{},
	}
}

func NewStackListController() *StackListController {
	return &StackListController{
		store: NewDefaultStackListStore(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewMembershipCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List stacks"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithBoolFlag(deletedFlag, false, "Display deleted stacks"),
		fctl.WithBoolFlag(allFlag, false, "Display deleted stacks"),
		fctl.WithDeprecatedFlag(deletedFlag, "Use --all instead"),
		fctl.WithController(NewStackListController()),
	)
}
func (c *StackListController) GetStore() *StackListStore {
	return c.store
}

func (c *StackListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	request := operations.ListStacksRequest{
		OrganizationID: organizationID,
		All:            pointer.For(fctl.GetBool(cmd, allFlag)),
		Deleted:        pointer.For(fctl.GetBool(cmd, deletedFlag)),
	}

	rsp, err := apiClient.ListStacks(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("listing stacks: %w", err)
	}

	if rsp.ListStacksResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	if len(rsp.ListStacksResponse.GetData()) == 0 {
		return c, nil
	}

	portal := fctl.DefaultConsoleURL
	serverInfo, err := fctl.MembershipServerInfo(cmd.Context(), apiClient)
	if err != nil {
		return nil, err
	}
	if v := serverInfo.GetConsoleURL(); v != nil {
		portal = *v
	}

	c.store.Stacks = fctl.Map(rsp.ListStacksResponse.GetData(), func(stack components.Stack) Stack {
		return Stack{
			Id:           stack.GetID(),
			Name:         stack.GetName(),
			Dashboard:    portal,
			RegionID:     stack.GetRegionID(),
			Status:       string(stack.GetState()),
			AuditEnabled: fctl.BoolPointerToString(stack.GetAuditEnabled()),
			DisabledAt: func() *string {
				if disabledAt := stack.GetDisabledAt(); disabledAt != nil {
					t := disabledAt.Format(time.RFC3339)
					return &t
				}
				return nil
			}(),
			DeletedAt: func() *string {
				if deletedAt := stack.GetDeletedAt(); deletedAt != nil {
					t := deletedAt.Format(time.RFC3339)
					return &t
				}
				return nil
			}(),
		}
	})

	return c, nil
}

func (c *StackListController) Render(cmd *cobra.Command, args []string) error {
	if len(c.store.Stacks) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No stacks found.")
		return nil
	}

	tableData := fctl.Map(c.store.Stacks, func(stack Stack) []string {
		data := []string{
			stack.Id,
			stack.Name,
			stack.Dashboard,
			stack.RegionID,
			stack.Status,
			stack.AuditEnabled,
		}
		if fctl.GetBool(cmd, allFlag) {
			if stack.DisabledAt != nil {
				data = append(data, *stack.DisabledAt)
			} else {
				data = append(data, "")
			}

			if stack.DeletedAt != nil {
				data = append(data, *stack.DeletedAt)
			} else {
				if stack.Status != "DELETED" {
					data = append(data, "")
				} else {
					data = append(data, "<retention period>")
				}
			}
		}

		return data
	})

	headers := []string{"ID", "Name", "Dashboard", "Region", "Status", "Audit Enabled"}
	if fctl.GetBool(cmd, allFlag) {
		headers = append(headers, "Disabled At", "Deleted At")
	}
	tableData = fctl.Prepend(tableData, headers)

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
