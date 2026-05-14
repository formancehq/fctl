package ledger

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type ImportLogsInput struct {
	Ledger            string
	FilePath          string
	Data              []byte
	ResumeFromLastLog bool
}

type ImportLogsOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Ledger     string                  `json:"ledger" yaml:"ledger"`
	Imported   bool                    `json:"imported" yaml:"imported"`
}

type ImportLogsHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context, ImportLogsInput) (ImportLogsOutput, error)
}

type ImportLogsService struct {
	Handlers []ImportLogsHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ImportLogsService) Run(ctx context.Context, input ImportLogsInput) (ImportLogsOutput, error) {
	if input.Ledger == "" {
		return ImportLogsOutput{}, fmt.Errorf("ledger is required")
	}
	if input.FilePath == "" && len(input.Data) == 0 {
		return ImportLogsOutput{}, fmt.Errorf("file is required")
	}
	if input.FilePath != "" && len(input.Data) > 0 {
		return ImportLogsOutput{}, fmt.Errorf("file and data are mutually exclusive")
	}
	if input.ResumeFromLastLog && (input.FilePath == "" || input.FilePath == "-") {
		return ImportLogsOutput{}, fmt.Errorf("resume requires a file path")
	}

	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ImportLogsHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ImportLogsOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ImportLogsOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx, input)
	if err != nil {
		return ImportLogsOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKImportLogsHandlers(sdk *formance.Formance) []ImportLogsHandler {
	return []ImportLogsHandler{
		{
			APIVersion: "v2",
			Run: func(ctx context.Context, input ImportLogsInput) (ImportLogsOutput, error) {
				body, closeBody, err := v2ImportLogsBody(ctx, sdk, input)
				if err != nil {
					return ImportLogsOutput{}, err
				}
				if closeBody != nil {
					defer closeBody()
				}

				_, err = sdk.Ledger.V2.ImportLogs(ctx, operations.V2ImportLogsRequest{
					Ledger:              input.Ledger,
					V2ImportLogsRequest: body,
				})
				if err != nil {
					return ImportLogsOutput{}, err
				}
				return ImportLogsOutput{Ledger: input.Ledger, Imported: true}, nil
			},
		},
	}
}

func v2ImportLogsBody(ctx context.Context, sdk *formance.Formance, input ImportLogsInput) (any, func(), error) {
	if input.ResumeFromLastLog {
		file, err := openImportFileAfterLastLog(ctx, sdk, input)
		if err != nil {
			return nil, nil, err
		}
		return file, func() { _ = file.Close() }, nil
	}
	if input.FilePath != "" {
		return []byte("file:" + filepath.Clean(input.FilePath)), nil, nil
	}
	return bytes.NewReader(input.Data), nil, nil
}

func openImportFileAfterLastLog(ctx context.Context, sdk *formance.Formance, input ImportLogsInput) (*os.File, error) {
	pageSize := int64(1)
	response, err := sdk.Ledger.V2.ListLogs(ctx, operations.V2ListLogsRequest{
		Ledger:   input.Ledger,
		PageSize: &pageSize,
	})
	if err != nil {
		return nil, err
	}
	if response.V2LogsCursorResponse == nil || len(response.V2LogsCursorResponse.Cursor.Data) == 0 {
		return os.Open(filepath.Clean(input.FilePath))
	}
	lastID := response.V2LogsCursorResponse.Cursor.Data[0].ID
	if lastID == nil {
		return nil, fmt.Errorf("last ledger log has no id")
	}
	return openImportFileAfterLogID(input.FilePath, lastID)
}

func openImportFileAfterLogID(filePath string, id *big.Int) (*os.File, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	offset, err := importOffsetAfterLogID(file, id)
	if err != nil {
		return nil, err
	}

	resumed, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := resumed.Seek(offset, 0); err != nil {
		_ = resumed.Close()
		return nil, err
	}
	return resumed, nil
}

func importOffsetAfterLogID(file *os.File, id *big.Int) (int64, error) {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var offset int64
	for scanner.Scan() {
		line := scanner.Bytes()
		var log struct {
			ID *big.Int `json:"id"`
		}
		if err := json.Unmarshal(line, &log); err != nil {
			return 0, err
		}
		offset += int64(len(line) + 1)
		if log.ID != nil && log.ID.Cmp(id) == 0 {
			return offset, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("log id %s was not found in import file", id.String())
}
