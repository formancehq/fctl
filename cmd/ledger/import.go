package ledger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/go-libs/v3/pointer"

	fctl "github.com/formancehq/fctl/pkg"
)

type ImportStore struct{}
type ImportController struct {
	store             *ImportStore
	inputFileFlag     string
	resumeFromLastLog string
}

var _ fctl.Controller[*ImportStore] = (*ImportController)(nil)

func NewDefaultImportStore() *ImportStore {
	return &ImportStore{}
}

func NewImportController() *ImportController {
	return &ImportController{
		store:             NewDefaultImportStore(),
		inputFileFlag:     "file",
		resumeFromLastLog: "resume-from-last-log",
	}
}

func NewImportCommand() *cobra.Command {
	c := NewImportController()
	return fctl.NewCommand("import <ledger name> <file path>",
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithShortDescription("Import a ledger"),
		fctl.WithStringFlag(c.inputFileFlag, "", "Import from stdin or file"),
		fctl.WithBoolFlag(c.resumeFromLastLog, false, "Recover interrupted import"),
		fctl.WithController[*ImportStore](c),
	)
}

func (c *ImportController) GetStore() *ImportStore {
	return c.store
}

func (c *ImportController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	lastID := big.NewInt(-1)
	resumeFromLastLog, err := cmd.Flags().GetBool(c.resumeFromLastLog)
	if err != nil {
		return nil, err
	}
	if resumeFromLastLog {
		logs, err := store.Client().Ledger.V2.ListLogs(cmd.Context(), operations.V2ListLogsRequest{
			Ledger:   args[0],
			PageSize: pointer.For[int64](1),
		})
		if err != nil {
			return nil, err
		}
		if len(logs.V2LogsCursorResponse.Cursor.Data) > 0 {
			lastID = logs.V2LogsCursorResponse.Cursor.Data[0].ID
		}
	}

	var (
		f               *os.File
		positionInBytes int
	)
	if lastID.Cmp(big.NewInt(-1)) == 0 {
		f, err = os.Open(args[1])
	} else {
		f, positionInBytes, err = c.openFileWithOffset(args[1], lastID)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fileInfo.Size()

	const blockSize = 100

	var (
		buffer = new(bytes.Buffer)
		count  = 0
	)

	progressBar, err := pterm.DefaultProgressbar.
		WithTotal(int(fileSize)).
		WithWriter(cmd.OutOrStdout()).
		WithCurrent(positionInBytes).
		WithRemoveWhenDone(true).
		WithShowCount(false).
		Start("Import")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		scannerErr := scanner.Err()
		if scannerErr != nil && !errors.Is(scannerErr, io.EOF) {
			return nil, fmt.Errorf("error reading file: %w", scannerErr)
		}
		bytes := scanner.Bytes()
		buffer.Write(bytes)
		buffer.Write([]byte("\n"))
		count++

		progressBar.Add(len(bytes) + 1) // +1 for the end of line

		if count == blockSize {
			_, err = store.Client().Ledger.V2.ImportLogs(cmd.Context(), operations.V2ImportLogsRequest{
				Ledger:              args[0],
				V2ImportLogsRequest: buffer,
			})
			if err != nil {
				return nil, err
			}
			buffer.Reset()
			count = 0
		}
		if errors.Is(scannerErr, io.EOF) {
			break
		}
	}

	if buffer.Len() > 0 {
		_, err = store.Client().Ledger.V2.ImportLogs(cmd.Context(), operations.V2ImportLogsRequest{
			Ledger:              args[0],
			V2ImportLogsRequest: buffer,
		})
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *ImportController) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Ledger imported!")
	return nil
}

func (c *ImportController) openFileWithOffset(filePath string, id *big.Int) (*os.File, int, error) {
	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)

	type log struct {
		ID *big.Int `json:"id"`
	}
	readBytes := 0
	for scanner.Scan() {
		if scanner.Err() != nil {
			return nil, 0, scanner.Err()
		}
		l := &log{}
		if err := json.Unmarshal(scanner.Bytes(), l); err != nil {
			return nil, 0, err
		}

		readBytes += len(scanner.Bytes()) + 1 // +1 for the end of line

		if l.ID.Cmp(id) == 0 {
			break
		}
	}

	ret, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}

	_, err = ret.Seek(int64(readBytes), 0)
	if err != nil {
		return nil, 0, err
	}

	return ret, readBytes, nil
}
