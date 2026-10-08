// Package command contains the small boundary shared by embedded modules.
package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func WriteJSON(writer io.Writer, value json.RawMessage) error {
	var out bytes.Buffer
	if err := json.Indent(&out, value, "", "  "); err != nil {
		return err
	}
	if err := out.WriteByte('\n'); err != nil {
		return err
	}
	_, err := writer.Write(out.Bytes())
	return err
}

// ReadBody accepts inline JSON, @file, or - for stdin, with a bounded size.
func ReadBody(cmd *cobra.Command, value string) (result json.RawMessage, err error) {
	var reader io.Reader
	switch {
	case value == "-":
		reader = cmd.InOrStdin()
	case len(value) > 0 && value[0] == '@':
		file, openErr := os.Open(value[1:])
		if openErr != nil {
			return nil, fmt.Errorf("open request body: %w", openErr)
		}
		defer func() { err = errors.Join(err, file.Close()) }()
		reader = file
	default:
		reader = bytes.NewBufferString(value)
	}
	data, err := readCancelable(cmd.Context(), reader)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if len(data) > 4<<20 {
		return nil, fmt.Errorf("request body exceeds 4 MiB")
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("request body must be one valid JSON document")
	}
	return data, nil
}

func readCancelable(ctx context.Context, reader io.Reader) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type result struct {
		data []byte
		err  error
	}
	finished := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(io.LimitReader(reader, 4<<20+1))
		finished <- result{data, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case read := <-finished:
		return read.data, read.err
	}
}
