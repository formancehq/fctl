package stacktools

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func readSSE(body io.Reader, id json.RawMessage) (json.RawMessage, error) {
	reader := bufio.NewReader(io.LimitReader(body, MaxMessageSize+1))
	total := 0
	for {
		data, err := nextEvent(reader, &total)
		if matchesResponse(data, id) {
			return json.RawMessage(data), nil
		}
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("MCP event stream has no matching response")
		}
		if err != nil {
			return nil, err
		}
	}
}

func nextEvent(reader *bufio.Reader, total *int) ([]byte, error) {
	var data []byte
	for {
		line, err := readLine(reader)
		*total += len(line)
		if *total > MaxMessageSize {
			return nil, fmt.Errorf("MCP event stream exceeds size limit")
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("MCP event stream read failed")
		}
		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			return data, err
		}
		data = appendEventData(data, line)
		if err != nil {
			return data, err
		}
	}
}

func appendEventData(data, line []byte) []byte {
	if !bytes.HasPrefix(line, []byte("data:")) {
		return data
	}
	if len(data) > 0 {
		data = append(data, '\n')
	}
	return append(data, bytes.TrimPrefix(bytes.TrimPrefix(line, []byte("data:")), []byte(" "))...)
}
