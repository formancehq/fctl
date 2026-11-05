package ledger

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportLogs(t *testing.T) {
	t.Parallel()

	// Larger than bufio.Scanner's 64 KB default token size.
	longLog := fmt.Sprintf(`{"id":1,"data":"%s"}`, strings.Repeat("a", 200*1024))
	input := `{"id":0}` + "\n" + longLog + "\n" + `{"id":2}`

	var (
		blocks []string
		read   int
	)
	err := importLogs(strings.NewReader(input), 2, func(n int) {
		read += n
	}, func(buffer *bytes.Buffer) error {
		blocks = append(blocks, buffer.String())
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, len(input), read)
	require.Equal(t, []string{
		`{"id":0}` + "\n" + longLog + "\n",
		`{"id":2}` + "\n",
	}, blocks)
}

func TestImportLogsSendError(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("boom")
	err := importLogs(strings.NewReader(`{"id":0}`+"\n"), 100, func(int) {}, func(*bytes.Buffer) error {
		return sendErr
	})
	require.ErrorIs(t, err, sendErr)
}

func TestOpenFileWithOffset(t *testing.T) {
	t.Parallel()

	longLog := fmt.Sprintf(`{"id":1,"data":"%s"}`, strings.Repeat("a", 200*1024))
	path := filepath.Join(t.TempDir(), "logs.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"id":0}`+"\n"+longLog+"\n"+`{"id":2}`+"\n"), 0o600))

	c := NewImportController()

	f, offset, err := c.openFileWithOffset(path, big.NewInt(1))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	require.Equal(t, len(`{"id":0}`)+1+len(longLog)+1, offset)

	_, _, err = c.openFileWithOffset(path, big.NewInt(42))
	require.ErrorContains(t, err, "log 42 not found")
}
