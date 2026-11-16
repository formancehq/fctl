package demo

import (
	"context"
	"fmt"
	"io"
	"math"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

// ledgerClient is the Ledger V2 surface needed by a demo run.
type ledgerClient interface {
	CreateLedger(context.Context, operations.V2CreateLedgerRequest, ...operations.Option) (*operations.V2CreateLedgerResponse, error)
	InsertSchema(context.Context, operations.V2InsertSchemaRequest, ...operations.Option) (*operations.V2InsertSchemaResponse, error)
	CreateTransaction(context.Context, operations.V2CreateTransactionRequest, ...operations.Option) (*operations.V2CreateTransactionResponse, error)
}

type recipeInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type runSummary struct {
	Ledger         string  `json:"ledger"`
	RunID          string  `json:"runID"`
	Transactions   uint64  `json:"transactions"`
	Errors         uint64  `json:"errors"`
	Interrupted    uint64  `json:"interrupted"`
	ElapsedSeconds float64 `json:"elapsedSeconds"`
	Rate           float64 `json:"rate"`
}

type store struct {
	Recipes []recipeInfo `json:"recipes,omitempty"`
	*runSummary
}

func newRunStore(ledgerName, runID string) *store {
	return &store{runSummary: &runSummary{Ledger: ledgerName, RunID: runID}}
}

type controller struct {
	recipeName   string
	concurrency  int
	store        *store
	client       func(*cobra.Command, ledger.V2SchemaDataInput) (ledgerClient, error)
	summaryReady bool
}

func NewCommand() *cobra.Command {
	return newCommand(&controller{client: authenticatedLedger})
}

func newCommand(c *controller) *cobra.Command {
	cmd := fctl.NewStackCommand("demo",
		fctl.WithShortDescription("Continuously generate demo transactions in a dedicated ledger"),
		fctl.WithDescription("Run a built-in demo recipe until Ctrl+C or --duration. Each run creates a new ledger and keeps its data after stopping. Supports Ledger v2 and v3, detected before creating the ledger."),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithBoolFlag("list", false, "List available recipes without connecting to a stack"),
		fctl.WithStringFlag("type", "", "Demo recipe (see --list)"),
		fctl.WithConfirmFlag(),
		fctl.WithController[*store](c),
		fctl.WithFlagCompletion("type", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return recipeNames(), cobra.ShellCompDirectiveNoFileComp
		}),
	)
	cmd.Flags().Float64("rate", 1, "Maximum transaction submissions per second (0 < rate <= 10000)")
	cmd.Flags().Duration("duration", 0, "Run duration after setup (0 runs until Ctrl+C)")
	cmd.Flags().Int("concurrency", 0, "Maximum in-flight requests (0 selects automatically from rate, maximum 1024)")
	return cmd
}

func authenticatedLedger(cmd *cobra.Command, schema ledger.V2SchemaDataInput) (ledgerClient, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return demoLedgerClient(cmd.Context(), clients, schema)
}

func (c *controller) GetStore() *store { return c.store }

func (c *controller) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	c.store = &store{}
	c.summaryReady = false
	if fctl.GetBool(cmd, "list") {
		for _, recipe := range recipeDefinitions {
			c.store.Recipes = append(c.store.Recipes, recipeInfo{Name: recipe.name, Description: recipe.description})
		}
		return c, nil
	}
	name := fctl.GetString(cmd, "type")
	if name == "" {
		return nil, fmt.Errorf("--type is required: use --list to see available recipes")
	}
	if _, err := recipeDefinitionFor(name); err != nil {
		return nil, err
	}
	c.recipeName = name

	rate, err := cmd.Flags().GetFloat64("rate")
	if err != nil {
		return nil, err
	}
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 || rate > 10000 || float64(time.Second)/rate >= float64(math.MaxInt64) {
		return nil, fmt.Errorf("--rate must be finite, greater than 0 and at most 10000 transactions per second")
	}
	duration, err := cmd.Flags().GetDuration("duration")
	if err != nil {
		return nil, err
	}
	if duration < 0 {
		return nil, fmt.Errorf("--duration must not be negative")
	}
	concurrency, err := cmd.Flags().GetInt("concurrency")
	if err != nil {
		return nil, err
	}
	if concurrency < 0 || concurrency > 1024 {
		return nil, fmt.Errorf("--concurrency must be between 0 and 1024")
	}
	c.concurrency = concurrency
	schema, err := loadRecipeSchema(c.recipeName)
	if err != nil {
		return nil, fmt.Errorf("load %s schema: %w", c.recipeName, err)
	}
	client, err := c.client(cmd, schema)
	if err != nil {
		return nil, err
	}
	runID := ksuid.New().String()
	c.store = newRunStore("demo-"+c.recipeName+"-"+runID, runID)
	if !fctl.CheckStackApprobation(cmd, "You are about to create ledger %s and continuously generate %s demo transactions at up to %g transactions/s", c.store.Ledger, c.recipeName, rate) {
		return nil, fctl.ErrMissingApproval
	}
	// The root command already handles Ctrl+C; also handle SIGTERM for scripted runs.
	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGTERM)
	defer stop()
	if err := c.generate(ctx, client, schema, rate, duration, cmd.ErrOrStderr()); err != nil {
		if c.summaryReady {
			cmd.SilenceUsage = true
			if renderErr := fctl.WithRender(cmd, nil, c, c); renderErr != nil {
				return nil, fmt.Errorf("%w (could not render final summary: %v)", err, renderErr)
			}
		}
		return nil, err
	}
	return c, nil
}

func (c *controller) generate(ctx context.Context, client ledgerClient, schema ledger.V2SchemaDataInput, rate float64, duration time.Duration, progress io.Writer) error {
	fmt.Fprintf(progress, "Demo %s: ledger=%s run=%s\n", c.recipeName, c.store.Ledger, c.store.RunID)
	response, err := client.CreateLedger(ctx, operations.V2CreateLedgerRequest{
		Ledger:                c.store.Ledger,
		V2CreateLedgerRequest: ledger.V2CreateLedgerRequest{V2Metadata: map[string]string{"demo.run": c.store.RunID, "demo.type": c.recipeName}},
	}, operations.WithOperationTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("create demo ledger %s: %w", c.store.Ledger, err)
	}
	if response.StatusCode != 204 {
		return fmt.Errorf("create demo ledger %s: unexpected HTTP %d", c.store.Ledger, response.StatusCode)
	}
	inserted, err := client.InsertSchema(ctx, operations.V2InsertSchemaRequest{Ledger: c.store.Ledger, Version: schemaVersion, V2SchemaData: schema}, operations.WithOperationTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("install %s recipe on %s (ledger retained): %w", c.recipeName, c.store.Ledger, err)
	}
	if inserted.StatusCode != 204 {
		return fmt.Errorf("install %s recipe on %s: unexpected HTTP %d (ledger retained)", c.recipeName, c.store.Ledger, inserted.StatusCode)
	}
	if duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}
	concurrency := c.concurrency
	if concurrency == 0 {
		concurrency = min(512, max(1, int(math.Ceil(rate/10))))
	}
	fmt.Fprintf(progress, "Target rate=%g/s concurrency=%d\n", rate, concurrency)
	c.summaryReady = true
	return c.runStream(ctx, client, rate, concurrency, progress)
}

func (c *controller) Render(cmd *cobra.Command, _ []string) error {
	if len(c.store.Recipes) > 0 {
		for _, recipe := range recipeDefinitions {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", recipe.name, recipe.description); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s: %d transactions, %d errors, %d interrupted, %.2f transactions/s. Data retained.\n", c.store.Ledger, c.store.Transactions, c.store.Errors, c.store.Interrupted, c.store.Rate)
	return err
}
