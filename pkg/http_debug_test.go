package fctl

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintBodyKeepsLargeIntegersExact(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	stdout := os.Stdout
	os.Stdout = writer
	printBody([]byte(`{"seq":1234567890123456789}`))
	os.Stdout = stdout
	require.NoError(t, writer.Close())
	printed, err := io.ReadAll(reader)
	require.NoError(t, err)

	require.Contains(t, string(printed), "1234567890123456789")
}
