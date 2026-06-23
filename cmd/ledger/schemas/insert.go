package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/ledger"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"

	internal "github.com/formancehq/fctl/v3/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const schemaFetchTimeout = 30 * time.Second

type InsertStore struct {
	Success bool `json:"success"`
}
type InsertController struct {
	store *InsertStore
}

var _ fctl.Controller[*InsertStore] = (*InsertController)(nil)

func NewDefaultInsertStore() *InsertStore {
	return &InsertStore{}
}

func NewInsertController() *InsertController {
	return &InsertController{
		store: NewDefaultInsertStore(),
	}
}

func NewInsertCommand() *cobra.Command {
	return fctl.NewCommand("insert <version> <source>",
		fctl.WithShortDescription("Insert a schema for a ledger from a JSON/YAML file or URL"),
		fctl.WithAliases("i", "create"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*InsertStore](NewInsertController()),
	)
}

func (c *InsertController) GetStore() *InsertStore {
	return c.store
}

func (c *InsertController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	ledgerName := fctl.GetString(cmd, internal.LedgerFlag)
	version := args[0]

	schemaData, err := loadSchemaData(cmd, args[1])
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to insert schema version %s on ledger %s", version, ledgerName) {
		return nil, fctl.ErrMissingApproval
	}

	response, err := stackClient.Ledger.V2.InsertSchema(cmd.Context(), operations.V2InsertSchemaRequest{
		Ledger:       ledgerName,
		Version:      version,
		V2SchemaData: *schemaData,
	})
	if err != nil {
		return nil, err
	}

	c.store.Success = response.StatusCode == 204
	if !c.store.Success {
		return nil, fmt.Errorf("unexpected status code %d while inserting schema", response.StatusCode)
	}
	return c, nil
}

func (c *InsertController) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Schema inserted!")
	return nil
}

func loadSchemaData(cmd *cobra.Command, source string) (*ledger.V2SchemaDataInput, error) {
	raw, err := readSource(cmd, source)
	if err != nil {
		return nil, err
	}

	var intermediate any
	if err := yaml.Unmarshal(raw, &intermediate); err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	normalized, err := json.Marshal(intermediate)
	if err != nil {
		return nil, err
	}

	schemaData := &ledger.V2SchemaDataInput{}
	if err := json.Unmarshal(normalized, schemaData); err != nil {
		return nil, err
	}

	return schemaData, nil
}

func readSource(cmd *cobra.Command, source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}
		client := fctl.GetHttpClient(cmd)
		client.Timeout = schemaFetchTimeout
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("fetching schema from %s: unexpected status %s", source, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}

	return os.ReadFile(filepath.Clean(source))
}
