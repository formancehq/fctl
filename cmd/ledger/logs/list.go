package logs

import (
	"fmt"
	"strconv"
	"time"

	"github.com/formancehq/fctl/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/pointer"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var timeRFC = time.RFC3339Nano
var (
	pageSizeFlag  = "page-size"
	afterFlag     = "after"
	startTimeFlag = "start-time"
	endTimeFLag   = "end-time"
	cursorFlag    = "cursor"
)

type ListStore struct {
	Cursor shared.LogsCursorResponseCursor `json:"cursor"`
}

type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List logs"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithIntFlag(pageSizeFlag, 15, "Page size"),
		fctl.WithStringFlag(cursorFlag, "", "Logs Cursor"),
		fctl.WithStringFlag(afterFlag, "", "Pagination cursor, will return the logs after a given ID. (in descending order)."),
		fctl.WithStringFlag(startTimeFlag, "", fmt.Sprintf("Start time (time.RFC %s)", timeRFC)),
		fctl.WithStringFlag(endTimeFLag, "", fmt.Sprintf("End time (time.RFC %s)", timeRFC)),
		fctl.WithController(c),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	store := fctl.GetStackStore(cmd.Context())
	request := operations.ListLogsRequest{
		Ledger:   fctl.GetString(cmd, internal.LedgerFlag),
		PageSize: pointer.For(int64(fctl.GetInt(cmd, pageSizeFlag))),
	}

	if after := fctl.GetString(cmd, afterFlag); after != "" {
		request.After = &after
	}

	if startTime := fctl.GetString(cmd, startTimeFlag); startTime != "" {
		str, err := time.Parse(timeRFC, startTime)
		if err != nil {
			return nil, fmt.Errorf(`invalid start time parsing with %s: %w`, timeRFC, err)
		}
		request.StartTime = &str
	}

	if endTime := fctl.GetString(cmd, endTimeFLag); endTime != "" {
		str, err := time.Parse(timeRFC, endTime)
		if err != nil {
			return nil, fmt.Errorf(`invalid end time parsing with %s: %w`, timeRFC, err)
		}
		request.EndTime = &str
	}

	rsp, err := store.Client().Ledger.V1.ListLogs(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	c.store.Cursor = rsp.LogsCursorResponse.Cursor

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	fmt.Println("")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("HasMore"), fmt.Sprintf("%v", c.store.Cursor.HasMore)})
	tableData = append(tableData, []string{pterm.LightCyan("PageSize"), fmt.Sprintf("%d", c.store.Cursor.PageSize)})
	tableData = append(tableData, []string{pterm.LightCyan("Next"), func() string {
		if c.store.Cursor.Next == nil {
			return ""
		}
		return *c.store.Cursor.Next
	}()})
	tableData = append(tableData, []string{pterm.LightCyan("Previous"), func() string {
		if c.store.Cursor.Previous == nil {
			return ""
		}
		return *c.store.Cursor.Previous
	}()})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	tableData = fctl.Map(c.store.Cursor.Data, func(log shared.Log) []string {
		return []string{
			log.Date.Format(timeRFC),
			strconv.FormatInt(log.ID, 10),
			string(log.Type),
			log.Hash,
		}
	})
	tableData = fctl.Prepend(tableData, []string{"Date", "ID", "Type", "Hash"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
