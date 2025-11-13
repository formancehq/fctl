package stack

import (
	"errors"
	"fmt"
	"strings"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/fctl/pkg/printer"
	"github.com/formancehq/go-libs/v3/pointer"
)

const (
	pageSizeFlag = "page-size"
	cursorFlag   = "cursor"

	actionFlag      = "action"
	dataFlag        = "data"
	userIdFlag      = "user-id"
	displayDataFlag = "display-data"
)

type HistoryStore struct {
	Cursor *components.LogCursorData `json:"cursor"`
}

type HistoryController struct {
	store *HistoryStore
}

var _ fctl.Controller[*HistoryStore] = (*HistoryController)(nil)

func NewDefaultHistoryStore() *HistoryStore {
	return &HistoryStore{
		Cursor: &components.LogCursorData{},
	}
}
func NewHistoryController() *HistoryController {
	return &HistoryController{
		store: NewDefaultHistoryStore(),
	}
}

func NewHistoryCommand() *cobra.Command {
	return fctl.NewMembershipCommand("history [id]",
		fctl.WithShortDescription("Query stack history"),
		fctl.WithAliases("hist"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(fctl.StackCompletion),
		fctl.WithStringFlag(actionFlag, "", "Filter on Action"),
		fctl.WithStringFlag(userIdFlag, "", "Filter on UserId, use SYSTEM to filter on system logs"),
		fctl.WithStringFlag(dataFlag, "", "Filter on modified Data with --data key=value, key is a jsonb text path"),

		fctl.WithBoolFlag(displayDataFlag, false, "Display data"),

		fctl.WithStringFlag(cursorFlag, "", "Cursor"),
		fctl.WithIntFlag(pageSizeFlag, 10, "Page size"),
		fctl.WithController(NewHistoryController()),
	)
}
func (c *HistoryController) GetStore() *HistoryStore {
	return c.store
}

func (c *HistoryController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	pageSize := fctl.GetInt(cmd, pageSizeFlag)
	stackID := args[0]

	cursor := fctl.GetString(cmd, cursorFlag)
	userID := fctl.GetString(cmd, userIdFlag)
	action := fctl.GetString(cmd, actionFlag)
	data := fctl.GetString(cmd, dataFlag)

	if cursor != "" {
		if userID != "" || action != "" || data != "" {
			return nil, errors.New("cursor can't be used with other flags")
		}
	}

	if stackID == "" && cursor == "" {
		return nil, errors.New("stack-id or cursor is required")
	}

	request := operations.ListLogsRequest{
		OrganizationID: organizationID,
		StackID:        &stackID,
		PageSize:       pointer.For(int64(pageSize)),
	}

	if cursor != "" {
		request.Cursor = &cursor
	}

	if userID != "" {
		request.UserID = &userID
	}

	if action != "" {
		if !strings.Contains(action, "stacks") {
			return nil, errors.New("stacks history are scoped to 'stacks.*' actions")
		}
		actionEnum := components.Action(action)
		request.Action = &actionEnum
	}

	if data != "" {
		keyVal := strings.Split(data, "=")
		if len(keyVal) != 2 {
			return nil, errors.New("data filter must be in the form key=value")
		}
		request.Key = &keyVal[0]
		request.Value = &keyVal[1]
	}

	response, err := apiClient.ListLogs(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.LogCursor == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	cursorData := response.LogCursor.GetData()
	c.store.Cursor = &cursorData
	return c, nil
}

func (c *HistoryController) Render(cmd *cobra.Command, args []string) error {
	return printer.LogCursor(cmd.OutOrStdout(), c.store.Cursor, fctl.GetBool(cmd, displayDataFlag))
}
