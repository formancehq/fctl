package fctl

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	cursorFlag   = "cursor"
	pageSizeFlag = "page-size"
)

func WithPageSizeFlag() CommandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().Int32(pageSizeFlag, 15, "Number of items per page")
	}
}

func WithCursorFlag() CommandOptionFn {
	return func(cmd *cobra.Command) {
		cmd.Flags().String(cursorFlag, "", "Cursor for pagination")
	}
}

func GetCursor(cmd *cobra.Command) (string, error) {
	cursor, err := cmd.Flags().GetString(cursorFlag)
	if err != nil {
		return "", fmt.Errorf("failed to get cursor: %w", err)
	}
	if cursor == "" {
		return "", nil
	}
	return cursor, nil
}

func GetPageSize(cmd *cobra.Command) (int32, error) {
	pageSize, err := cmd.Flags().GetInt32(pageSizeFlag)
	if err != nil {
		return 0, fmt.Errorf("failed to get page size: %w", err)
	}
	if pageSize <= 0 {
		return 0, fmt.Errorf("page size must be greater than 0")
	}
	return pageSize, nil
}

type Cursor struct {
	HasMore  bool
	PageSize int64
	Next     *string
	Previous *string
}

func RenderCursor(writer io.Writer, cursor Cursor) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("HasMore"), fmt.Sprintf("%v", cursor.HasMore)})
	tableData = append(tableData, []string{pterm.LightCyan("PageSize"), fmt.Sprintf("%d", cursor.PageSize)})
	tableData = append(tableData, []string{pterm.LightCyan("Next"), func() string {
		if cursor.Next == nil {
			return ""
		}
		return *cursor.Next
	}()})
	tableData = append(tableData, []string{pterm.LightCyan("Previous"), func() string {
		if cursor.Previous == nil {
			return ""
		}
		return *cursor.Previous
	}()})

	return pterm.DefaultTable.
		WithWriter(writer).
		WithData(tableData).
		Render()
}
