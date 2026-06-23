package schemas

import (
	"encoding/json"
	"fmt"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/ledger"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"

	internal "github.com/formancehq/fctl/v3/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type GetStore struct {
	Schema ledger.V2SchemaData `json:"schema"`
}
type GetController struct {
	store      *GetStore
	formatFlag string
}

var _ fctl.Controller[*GetStore] = (*GetController)(nil)

func NewDefaultGetStore() *GetStore {
	return &GetStore{}
}

func NewGetController() *GetController {
	return &GetController{
		store:      NewDefaultGetStore(),
		formatFlag: "format",
	}
}

func NewGetCommand() *cobra.Command {
	c := NewGetController()
	return fctl.NewCommand("get <version>",
		fctl.WithShortDescription("Get a schema for a ledger by version"),
		fctl.WithAliases("g", "show"),
		fctl.WithStringFlag(c.formatFlag, "json", "Output format of the schema (json, yaml)"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*GetStore](c),
	)
}

func (c *GetController) GetStore() *GetStore {
	return c.store
}

func (c *GetController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	response, err := stackClient.Ledger.V2.GetSchema(cmd.Context(), operations.V2GetSchemaRequest{
		Ledger:  fctl.GetString(cmd, internal.LedgerFlag),
		Version: args[0],
	})
	if err != nil {
		return nil, err
	}

	c.store.Schema = response.V2SchemaResponse.V2SchemaData
	return c, nil
}

func (c *GetController) Render(cmd *cobra.Command, _ []string) error {
	out, err := json.Marshal(c.store.Schema)
	if err != nil {
		return err
	}

	raw := make(map[string]any)
	if err := json.Unmarshal(out, &raw); err != nil {
		_, err = cmd.OutOrStdout().Write(out)
		return err
	}

	switch format := fctl.GetString(cmd, c.formatFlag); format {
	case "yaml", "yml":
		yamlOut, err := yaml.Marshal(raw)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(yamlOut)
		return err
	case "json":
		f := colorjson.NewFormatter()
		f.Indent = 2
		colorized, err := f.Marshal(raw)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(append(colorized, '\n'))
		return err
	default:
		return fmt.Errorf("unsupported format %q (expected json or yaml)", format)
	}
}
