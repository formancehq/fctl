package clarity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Store map[string]any

type RunFunc func(cmd *cobra.Command, args []string, client *Client) (Store, error)
type RenderFunc func(cmd *cobra.Command, store Store) error

type Controller struct {
	store  Store
	run    RunFunc
	render RenderFunc
}

var _ fctl.Controller[Store] = (*Controller)(nil)

func NewController(run RunFunc, render RenderFunc) *Controller {
	return &Controller{store: Store{}, run: run, render: render}
}

func (c *Controller) GetStore() Store { return c.store }

func (c *Controller) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	client, err := NewClient(cmd)
	if err != nil {
		return nil, err
	}
	c.store, err = c.run(cmd, args, client)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Controller) Render(cmd *cobra.Command, _ []string) error {
	if c.render != nil {
		return c.render(cmd, c.store)
	}
	return PrintJSON(cmd.OutOrStdout(), c.store)
}

func PrintJSON(writer io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, string(data))
	return err
}

func PaginationQuery(cmd *cobra.Command) (url.Values, error) {
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	if cursor != "" {
		values.Set("cursor", cursor)
		return values, nil
	}
	values.Set("pageSize", fmt.Sprintf("%d", pageSize))
	return values, nil
}

func QueryBody(cmd *cobra.Command, flag string) (json.RawMessage, error) {
	raw, err := cmd.Flags().GetString(flag)
	if err != nil || raw == "" {
		return nil, err
	}
	return parseJSONObject([]byte(raw), fmt.Sprintf("--%s JSON", flag))
}

func ReadJSONObject(cmd *cobra.Command, path string) (json.RawMessage, error) {
	raw, err := fctl.ReadFile(cmd, path)
	if err != nil {
		return nil, err
	}
	return parseJSONObject([]byte(raw), "JSON in "+path)
}

func parseJSONObject(raw []byte, source string) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("invalid %s", source)
	}
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("invalid %s: expected an object", source)
	}
	return json.RawMessage(trimmed), nil
}
