package stacktools

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func readMessage(reader *bufio.Reader) ([]byte, error) {
	line, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	for len(bytes.TrimSpace(line)) == 0 {
		line, err = readLine(reader)
		if err != nil {
			return nil, err
		}
	}
	if !strings.HasPrefix(strings.ToLower(string(line)), "content-length:") {
		return bytes.TrimSpace(line), nil
	}
	_, value, _ := strings.Cut(string(line), ":")
	size, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || size < 0 || size > MaxMessageSize {
		return nil, fmt.Errorf("invalid MCP Content-Length (maximum %d bytes)", MaxMessageSize)
	}
	headers := len(line)
	for {
		line, err = readLine(reader)
		if err != nil {
			return nil, frameError(err)
		}
		headers += len(line)
		if headers > MaxMessageSize {
			return nil, fmt.Errorf("MCP headers exceed size limit")
		}
		if len(bytes.TrimSpace(line)) == 0 {
			break
		}
	}
	data := make([]byte, size)
	_, err = io.ReadFull(reader, data)
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	return data, err
}

func readLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(line)+len(part) > MaxMessageSize {
			return nil, fmt.Errorf("MCP message exceeds size limit")
		}
		line = append(line, part...)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) && len(line) > 0 {
			return line, nil
		}
		return line, err
	}
}

func frameError(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
