package demo

import (
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/formancehq/go-libs/v4/pointer"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

//go:embed recipes/*.yaml
var recipeFiles embed.FS

type scenario interface {
	next(ledgerName, runID string) (operations.V2CreateTransactionRequest, func())
}

type recipeDefinition struct {
	name         string
	description  string
	newScenario  func(string) scenario
	openingFunds bool
}

// Only runnable recipes are listed. The docs' three chart-only illustrations
// have no transaction templates and are intentionally excluded.
var recipeDefinitions = []recipeDefinition{
	plannedRecipe("bnpl-lending", "Buy now, pay later financing and collection", bnplLendingSteps, false),
	plannedRecipe("crypto-custody", "Fiat and crypto custody, trades and withdrawals", cryptoCustodySteps, false),
	plannedRecipe("issuing-financial-host", "Card authorization, capture, refunds and chargebacks", issuingFinancialHostSteps, true),
	plannedRecipe("marketplace-payouts", "Marketplace orders, seller payouts, refunds and FX", marketplacePayoutsSteps, false),
	plannedRecipe("neobank-fbo", "Neobank deposits, cards, withdrawals and advances", neobankFBOSteps, true),
	{name: "omnibus", description: "Omnibus deposits, suspense resolution and payouts", newScenario: func(namespace string) scenario { o := newOmnibus(); o.namespace = namespace; return o }},
	plannedRecipe("payment-acceptance", "Payment authorization, settlement, refunds and chargebacks", paymentAcceptanceSteps, false),
	plannedRecipe("rideshare-tutorial", "Ride booking, payment split and driver payout", rideshareSteps, false),
	plannedRecipe("stablecoin-issuer", "Stablecoin mint, transfer, redemption and reserve management", stablecoinIssuerSteps, false),
	plannedRecipe("stablecoin-operations", "Stablecoin onramp, mint, burn and fiat withdrawal", stablecoinOperationsSteps, false),
}

func plannedRecipe(name, description string, build func(cycleContext) []recipeStep, openingFunds bool) recipeDefinition {
	return recipeDefinition{name: name, description: description, openingFunds: openingFunds, newScenario: func(namespace string) scenario {
		return &plannedScenario{name: name, namespace: namespace, build: build}
	}}
}

func recipeNames() []string {
	names := make([]string, len(recipeDefinitions))
	for i, recipe := range recipeDefinitions {
		names[i] = recipe.name
	}
	return names
}

func recipeDefinitionFor(name string) (recipeDefinition, error) {
	for _, recipe := range recipeDefinitions {
		if recipe.name == name {
			return recipe, nil
		}
	}
	return recipeDefinition{}, fmt.Errorf("unknown demo type %q: use --list to see available recipes", name)
}

func loadRecipeSchema(name string) (ledger.V2SchemaDataInput, error) {
	definition, err := recipeDefinitionFor(name)
	if err != nil {
		return ledger.V2SchemaDataInput{}, err
	}
	path := "recipes/" + name + ".yaml"

	raw, err := recipeFiles.ReadFile(path)
	if err != nil {
		return ledger.V2SchemaDataInput{}, err
	}
	var data any
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return ledger.V2SchemaDataInput{}, err
	}
	normalized, err := json.Marshal(data)
	if err != nil {
		return ledger.V2SchemaDataInput{}, err
	}
	var schema ledger.V2SchemaDataInput
	if err := json.Unmarshal(normalized, &schema); err != nil {
		return schema, err
	}
	for key, template := range schema.V2TransactionTemplates {
		template.Runtime = ledger.RuntimeExperimentalInterpreter.ToPointer()
		schema.V2TransactionTemplates[key] = template
	}
	// An explicit opening-funds transaction is required by the card-issuing and
	// neobank examples. Declare its external source in the effective chart as well.
	if definition.openingFunds {
		schema.V2ChartOfAccounts["world"] = ledger.V2ChartSegment{DotSelf: &ledger.DotSelf{}}
	}

	return schema, nil
}

func omnibusSchema() (ledger.V2SchemaDataInput, error) { return loadRecipeSchema("omnibus") }

func newRecipeScenario(name, namespace string) scenario {
	definition, err := recipeDefinitionFor(name)
	if err != nil {
		panic(err)
	} // Callers select an already validated catalog entry.
	return definition.newScenario(namespace)
}

type recipeStep struct {
	template string
	vars     map[string]string
	script   string // Explicit synthetic opening funds; never disguised as an official template.
}

type cycleContext struct {
	namespace, runID string
	cycle, sequence  uint64
	amount           int64
}

func (c cycleContext) account(label string) string {
	return accountSegment(c.namespace, label)
}
func (c cycleContext) reference(label string) string {
	return fmt.Sprintf("%s_c%d_%s", c.account(c.runID), c.cycle, label)
}
func (c cycleContext) txReference(stepIndex int) string {
	return transactionReference(c.runID, c.namespace, c.sequence+uint64(stepIndex))
}
func transactionReference(runID, namespace string, sequence uint64) string {
	if namespace != "" {
		runID += "-" + namespace
	}
	return fmt.Sprintf("%s-%d", runID, sequence)
}
func monetary(asset string, amount int64) string { return asset + " " + strconv.FormatInt(amount, 10) }

type plannedScenario struct {
	name, namespace string
	cycle, sequence uint64
	index           int
	plan            []recipeStep
	build           func(cycleContext) []recipeStep
}

func (s *plannedScenario) next(ledgerName, runID string) (operations.V2CreateTransactionRequest, func()) {
	if s.plan == nil {
		s.plan = s.build(cycleContext{namespace: s.namespace, runID: runID, cycle: s.cycle, sequence: s.sequence, amount: int64(10000 + rand.IntN(9901))})
	}
	step := s.plan[s.index]
	reference := transactionReference(runID, s.namespace, s.sequence)
	script := &ledger.V2PostTransactionScript{Template: pointer.For(step.template), Vars: maps.Clone(step.vars)}
	if step.script != "" {
		script.Template = nil
		script.Plain = pointer.For(step.script)
	}
	request := operations.V2CreateTransactionRequest{
		Ledger: ledgerName, SchemaVersion: pointer.For(schemaVersion), IdempotencyKey: pointer.For(reference),
		V2PostTransaction: ledger.V2PostTransaction{
			Reference: pointer.For(reference), V2Metadata: map[string]string{"demo.run": runID, "demo.type": s.name, "demo.template": step.template}, Script: script,
		},
	}
	if step.script != "" {
		request.V2PostTransaction.Runtime = ledger.RuntimeExperimentalInterpreter.ToPointer()
	}
	return request, func() {
		s.sequence++
		s.index++
		if s.index == len(s.plan) {
			s.index = 0
			s.plan = nil
			s.cycle++
		}
	}
}

func rideshareSteps(c cycleContext) []recipeStep {
	amount := strconv.FormatInt(c.amount, 10)
	rider, driver, ride := c.account("rider"), c.account("driver"), c.reference("ride")
	return []recipeStep{
		{template: "RIDE_BOOKING", vars: map[string]string{"amount": amount, "rider_id": rider, "ride_id": ride}},
		{template: "RIDE_CONFIRMATION", vars: map[string]string{"amount": amount, "rider_id": rider, "ride_id": ride}},
		{template: "RIDE_COMPLETION", vars: map[string]string{"amount": amount, "ride_id": ride, "driver_id": driver}},
		{template: "DRIVER_PAYOUT", vars: map[string]string{"driver_id": driver}},
	}
}

func accountSegment(namespace, label string) string {
	if namespace == "" {
		return label
	}
	return namespace + "_" + label
}
