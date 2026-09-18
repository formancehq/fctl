package fctl

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type jsonTestController struct {
	Amount int64 `json:"amount"`
}

func (c *jsonTestController) GetStore() *jsonTestController { return c }

func (c *jsonTestController) Run(*cobra.Command, []string) (Renderable, error) {
	return nil, nil
}

func TestWithRenderJSON(t *testing.T) {
	cmd := NewCommand("json", WithStringFlag(OutputFlag, "json", ""))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	controller := &jsonTestController{Amount: 9007199254740993}
	require.NoError(t, WithRender(cmd, nil, controller, nil))

	var document struct {
		Data jsonTestController `json:"data"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &document), "stdout: %q", stdout.String())
	require.Equal(t, controller.Amount, document.Data.Amount, "JSON must preserve integers beyond float64 precision")
}
