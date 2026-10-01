package ledger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/go-libs/v4/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	lastID := big.NewInt(-1)
	resumeFromLastLog, err := cmd.Flags().GetBool(c.resumeFromLastLog)
	if err != nil {
		return nil, err
	}
	if resumeFromLastLog {
		logs, err := stackClient.Ledger.V2.ListLogs(cmd.Context(), operations.V2ListLogsRequest{
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

	err = importLogs(f, importBlockSize, func(n int) {
		progressBar.Add(n)
	}, func(buffer *bytes.Buffer) error {
		_, err := stackClient.Ledger.V2.ImportLogs(cmd.Context(), operations.V2ImportLogsRequest{
			Ledger:              args[0],
			V2ImportLogsRequest: buffer,
		})
		return err
	})
	if err != nil {
		return nil, err
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

	reader := bufio.NewReader(f)

	type log struct {
		ID *big.Int `json:"id"`
	}
	readBytes := 0
	for {
		line, err := readLine(reader)
		if errors.Is(err, io.EOF) {
			return nil, 0, fmt.Errorf("log %s not found in %s", id, filePath)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("error reading file: %w", err)
		}
		l := &log{}
		if err := json.Unmarshal(line, l); err != nil {
			return nil, 0, err
		}

		readBytes += len(line)

		if l.ID.Cmp(id) == 0 {
			break
		}
	}

	ret, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, 0, err
	}

	_, err = ret.Seek(int64(readBytes), 0)
	if err != nil {
		return nil, 0, err
	}

	return ret, readBytes, nil
}

const importBlockSize = 100

// importLogs reads newline-delimited logs from r and passes them to send in
// blocks of blockSize lines. onRead receives the number of bytes consumed.
func importLogs(r io.Reader, blockSize int, onRead func(int), send func(*bytes.Buffer) error) error {
	var (
		reader = bufio.NewReader(r)
		buffer = new(bytes.Buffer)
		count  = 0
	)
	for {
		line, err := readLine(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
		onRead(len(line))

		buffer.Write(line)
		if line[len(line)-1] != '\n' {
			buffer.WriteByte('\n')
		}
		count++

		if count == blockSize {
			if err := send(buffer); err != nil {
				return err
			}
			buffer.Reset()
			count = 0
		}
	}

	if buffer.Len() > 0 {
		return send(buffer)
	}
	return nil
}

// readLine returns the next line of r with its trailing newline, if any.
// Unlike bufio.Scanner it has no line length limit. It returns io.EOF only
// once no bytes are left.
func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if errors.Is(err, io.EOF) && len(line) > 0 {
		return line, nil
	}
	return line, err
}
