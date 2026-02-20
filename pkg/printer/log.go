package printer

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	"github.com/formancehq/go-libs/time"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func LogCursor(writer io.Writer, cursor *components.LogCursorData, withData bool) error {
	header := []string{"Identifier", "User", "Date", "Action"}

	if withData {
		header = append(header, "Data")
	}
	tableData := fctl.Map(cursor.GetData(), func(log components.Log) []string {
		line := []string{
			log.GetSeq(),
			func() string {
				if log.GetUserID() == "" {
					return "SYSTEM"
				}
				return log.GetUserID()
			}(),
			log.GetDate().Format(time.DateFormat),
			log.GetAction(),
		}

		if withData {
			line = append(line, func() string {
				data := log.GetData()
				if data == (components.LogData{}) {
					return ""
				}
				return fmt.Sprintf("%v", data)
			}())

		}

		return line
	})
	tableData = fctl.Prepend(tableData, header)

	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(writer).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return Cursor(writer, &CursorData{
		HasMore:  cursor.GetHasMore(),
		PageSize: cursor.GetPageSize(),
		Next:     cursor.GetNext(),
		Previous: cursor.GetPrevious(),
	})
}
