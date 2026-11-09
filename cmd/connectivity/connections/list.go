package connections

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const connectorFlag = "connector"

type ListStore struct {
	Connections []connectivity.Connection `json:"connections"`
	Cursor      fctl.Cursor               `json:"cursor"`
}

type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController() *ListController {
	return &ListController{
		store: &ListStore{Connections: []connectivity.Connection{}},
	}
}

func NewListCommand() *cobra.Command {
	controller := NewListController()
	return fctl.NewCommand(
		"list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List connections"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(connectorFlag, "", "Filter connections by connector"),
		connectivityinternal.WithListQueryFlags(),
		fctl.WithPageSizeFlag(),
		fctl.WithCursorFlag(),
		fctl.WithController[*ListStore](controller),
		fctl.WithExamples(
			"fctl connectivity connections list --connector ee-stripeconnect",
			"fctl connectivity connections list --filter lifecycle=failed",
		),
		fctl.WithoutFileCompletion(connectivityinternal.QueryFlag, "cursor", "page-size"),
		fctl.WithFlagCompletion(connectorFlag, connectors.CompleteConnectorNames()),
		fctl.WithFlagCompletion(connectivityinternal.FilterFlag, connectivityinternal.CompleteFilterExpressions(connectivityinternal.ResourceConnections)),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	client, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, client.Connectivity.V1)
}

func (c *ListController) run(cmd *cobra.Command, args []string, client *sdk.FormanceV1) (fctl.Renderable, error) {
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	filters, err := cmd.Flags().GetStringArray(connectivityinternal.FilterFlag)
	if err != nil {
		return nil, err
	}
	// --connector is sugar for one more filter clause, so it composes with
	// --filter and conflicts with --query exactly like the other filters.
	if connector := fctl.GetString(cmd, connectorFlag); connector != "" {
		filters = append(filters, connectorFlag+"="+connector)
	}
	query, err := connectivityinternal.BuildListQuery(fctl.GetString(cmd, connectivityinternal.QueryFlag), filters)
	if err != nil {
		return nil, err
	}

	response, err := client.ListConnections(cmd.Context(), operations.ListConnectionsRequest{
		Query:    connectivityinternal.OptionalString(query),
		PageSize: fctl.Ptr(int(pageSize)),
		Cursor:   connectivityinternal.OptionalString(cursor),
	}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil || response.ConnectionCursor == nil || response.ConnectionCursor.Cursor.Data == nil {
		return nil, fmt.Errorf("list connectivity connections: empty response")
	}
	page := response.ConnectionCursor.Cursor
	c.store.Connections = page.Data
	c.store.Cursor = fctl.Cursor{PageSize: int64(page.PageSize), HasMore: page.HasMore, Next: page.Next}
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	rows := fctl.Map(c.store.Connections, func(instance connectivity.Connection) []string {
		return []string{instance.Name, fctl.StringPointerToString(instance.ConnectionConfiguration.Connector), fctl.StringPointerToString(instance.ConnectionConfiguration.Ledger),
			targetValue(&instance.ConnectionConfiguration), appliedChannelFloor(&instance), statusSummary(instance.ConnectionStatus)}
	})
	columns := []fctl.TableColumn{{}, {Fit: fctl.FitCut, Min: 10}, {Fit: fctl.FitCut, Min: 10}, {}, {}, {Style: lifecycleStyle}}
	if err := fctl.RenderTable(out, []string{"Name", "Connector", "Ledger", "Target", "Applied", "Status"}, columns, rows); err != nil {
		return err
	}
	return fctl.RenderNextPage(out, c.store.Cursor)
}

// targetValue is the requested version or channel; the API sets exactly one.
func targetValue(config *connectivity.ConnectionConfiguration) string {
	if version := fctl.StringPointerToString(config.Version); version != "" {
		return version
	}
	return enumStringValue(config.Channel)
}

// expectedIngestion is the ingestion state that adds nothing to a lifecycle.
var expectedIngestion = map[connectivity.Lifecycle]connectivity.State{
	connectivity.LifecycleActive:    connectivity.StateRunning,
	connectivity.LifecycleCompleted: connectivity.StateFinished,
	connectivity.LifecycleSuspended: connectivity.StatePaused,
}

// statusSummary is the lifecycle, followed by the ingestion state only when
// it is not the one the lifecycle implies.
func statusSummary(status *connectivity.ConnectionStatus) string {
	lifecycle := lifecycleValue(status)
	state := ingestionState(status)
	if state == "" || expectedIngestion[status.Lifecycle] == connectivity.State(state) {
		return lifecycle
	}
	return lifecycle + " / " + state
}

func lifecycleStyle(value string) string {
	lifecycle, _, _ := strings.Cut(strings.TrimSpace(value), " ")
	switch {
	case strings.Contains(value, "/"), lifecycle == string(connectivity.LifecycleFailed), lifecycle == string(connectivity.LifecycleUnknown):
		return pterm.LightRed(value)
	case lifecycle == string(connectivity.LifecyclePending), lifecycle == string(connectivity.LifecycleSuspended):
		return pterm.Yellow(value)
	case lifecycle == string(connectivity.LifecycleActive):
		return pterm.LightGreen(value)
	}
	return value
}

func enumStringValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
