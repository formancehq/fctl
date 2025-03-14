package ledger

import (
	"bufio"
	"encoding/json"
	"math/big"
	"os"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/go-libs/pointer"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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

	var f *os.File
	if lastID.Cmp(big.NewInt(-1)) == 0 {
		f, err = os.Open(args[1])
	} else {
		f, err = c.openFileWithOffset(args[1], lastID)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = store.Client().Ledger.V2.ImportLogs(cmd.Context(), operations.V2ImportLogsRequest{
		Ledger:              args[0],
		V2ImportLogsRequest: f,
	})
	if err != nil {
		return nil, err
	}

	return c, err
}

func (c *ImportController) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Ledger imported!")
	return nil
}

func (c *ImportController) openFileWithOffset(filePath string, id *big.Int) (*os.File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
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
			return nil, scanner.Err()
		}
		l := &log{}
		if err := json.Unmarshal(scanner.Bytes(), l); err != nil {
			return nil, err
		}

		readBytes += len(scanner.Bytes()) + 1 // +1 for the end of line

		if l.ID.Cmp(id) == 0 {
			break
		}
	}

	ret, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	_, err = ret.Seek(int64(readBytes), 0)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
