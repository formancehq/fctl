package demo

import (
	"fmt"
	"maps"
	"math/big"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	ns "github.com/formancehq/numscript"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledgerv3"
)

// Run the actual public Numscript evaluator against the state left by earlier
// acknowledged requests. A fake HTTP success would miss insufficient funds,
// invalid variables, broken return paths and charts that reject real postings.
func TestCatalogRecipesExecuteRepeatedCycles(t *testing.T) {
	for _, name := range recipeNames() {
		for _, namespace := range []string{"", "worker_017"} {
			t.Run(name+"/"+namespace, func(t *testing.T) {
				schema, err := loadRecipeSchema(name)
				require.NoError(t, err)
				scenario := newRecipeScenario(name, namespace)
				if random, ok := scenario.(*omnibus); ok {
					random.random = rand.New(rand.NewPCG(42, 99))
				}
				if planned, ok := scenario.(*plannedScenario); ok {
					build := planned.build
					planned.build = func(c cycleContext) []recipeStep {
						c.amount = 10000 + int64((c.cycle*373)%9901)
						return build(c)
					}
				}
				balances := ns.Balances{}
				metadata := ns.AccountsMetadata{}
				parses := make(map[string]ns.ParseResult)
				for template, definition := range schema.V2TransactionTemplates {
					parsed := ns.Parse(definition.Script)
					require.Empty(t, parsed.GetParsingErrors(), "%s/%s must parse", name, template)
					parses[template] = parsed
				}
				accountTypes := v3AccountTypes(schema.V2ChartOfAccounts)
				seenTemplates := make(map[string]int)
				seenReferences := make(map[string]bool)
				marketplaceCaptures := make(map[string]string)
				for index := range 500 {
					request, commit := scenario.next("demo-test", "test-run")
					require.NotNil(t, request.V2PostTransaction.Script)
					require.NotNil(t, request.IdempotencyKey)
					require.NotNil(t, request.V2PostTransaction.Reference)
					require.Equal(t, *request.IdempotencyKey, *request.V2PostTransaction.Reference)
					require.False(t, seenReferences[*request.IdempotencyKey], "reference reused at step %d", index)
					seenReferences[*request.IdempotencyKey] = true
					require.Equal(t, name, request.V2PostTransaction.V2Metadata["demo.type"])
					require.Equal(t, "test-run", request.V2PostTransaction.V2Metadata["demo.run"])
					script := request.V2PostTransaction.Script
					template := request.V2PostTransaction.V2Metadata["demo.template"]
					var parsed ns.ParseResult
					if script.Template != nil {
						require.Equal(t, *script.Template, template)
						var found bool
						parsed, found = parses[*script.Template]
						require.True(t, found, "unknown template %q at step %d", *script.Template, index)
					} else {
						require.NotNil(t, script.Plain, "step %d must declare template or plain script", index)
						require.Equal(t, "DEMO_BOOTSTRAP", template, "only explicit demo setup may be inline")
						cacheKey := "inline:" + *script.Plain
						var found bool
						parsed, found = parses[cacheKey]
						if !found {
							parsed = ns.Parse(*script.Plain)
							require.Empty(t, parsed.GetParsingErrors(), "bootstrap must parse")
							parses[cacheKey] = parsed
						}
					}
					require.ElementsMatch(t, slices.Collect(maps.Keys(parsed.GetNeededVariables())), slices.Collect(maps.Keys(script.Vars)), "%s step %d variables", template, index)
					result, err := parsed.Run(t.Context(), script.Vars, ns.StaticStore{Balances: balances, Meta: metadata})
					require.NoError(t, err, "recipe=%s namespace=%s step=%d template=%s vars=%v balances=%v", name, namespace, index, template, script.Vars, balances)
					require.NotEmpty(t, result.Postings, "recipe=%s step=%d template=%s must produce actual fund movements", name, index, template)
					if name == "marketplace-payouts" {
						if template == "ORDER_PAYMENT_CAPTURED" {
							marketplaceCaptures[script.Vars["order_ref"]] = *request.V2PostTransaction.Reference
						}
						if template == "CHARGEBACK" {
							capture := marketplaceCaptures[script.Vars["order_ref"]]
							require.NotEmpty(t, capture, "chargeback must adjust an existing captured order")
							require.Equal(t, capture, script.Vars["dispute_ref"])
							require.Equal(t, capture, fmt.Sprint(result.Metadata["adjusted_posting_event_id"]))
						}
					}

					touched := make(map[string]bool)
					for _, posting := range result.Postings {
						require.Positive(t, posting.Amount.Sign(), "%s step %d contains zero or negative posting", template, index)
						for _, account := range []string{posting.Source, posting.Destination} {
							touched[account] = true
							require.True(t, catalogAccountMatches(accountTypes, account), "%s step %d %s is outside strict v3 chart: %s", name, index, template, account)
						}
						catalogAddBalance(balances, posting.Source, posting.Asset, new(big.Int).Neg(posting.Amount))
						catalogAddBalance(balances, posting.Destination, posting.Asset, posting.Amount)
					}
					// The v3 adapter derives chart defaults from script literals. Ensure
					// those defaults never target a read-only or skipped account.
					for account, values := range v3RecipeAccountMetadata(schema, parsed.GetSource(), script.Vars) {
						require.True(t, touched[account], "%s step %d v3 default metadata on untouched account %s", template, index, account)
						if metadata[account] == nil {
							metadata[account] = map[string]string{}
						}
						for key, value := range values {
							require.NotNil(t, value.Str, "documented metadata defaults are strings")
							metadata[account][key] = *value.Str
						}
					}
					for account, values := range result.AccountsMetadata {
						if metadata[account] == nil {
							metadata[account] = map[string]string{}
						}
						maps.Copy(metadata[account], values)
					}
					// Metadata supplied by a scenario must obey the v3 restriction to accounts
					// actually involved in this transaction, not just addresses in balance().
					for account, values := range request.V2PostTransaction.AccountMetadata {
						require.True(t, touched[account], "%s step %d metadata on untouched account %s", template, index, account)
						if metadata[account] == nil {
							metadata[account] = map[string]string{}
						}
						maps.Copy(metadata[account], values)
					}
					commit()
					if script.Template != nil {
						seenTemplates[*script.Template]++
					}
				}
				require.ElementsMatch(t, slices.Collect(maps.Keys(schema.V2TransactionTemplates)), slices.Collect(maps.Keys(seenTemplates)), "all official templates must execute positively")
				sums := map[string]*big.Int{}
				for _, accountBalances := range balances {
					for asset, amount := range accountBalances {
						if sums[asset] == nil {
							sums[asset] = new(big.Int)
						}
						sums[asset].Add(sums[asset], amount)
					}
				}
				for asset, total := range sums {
					require.Zero(t, total.Sign(), "funds must balance in %s", asset)
				}
				t.Logf("executed 500 transactions; covered %d official templates", len(seenTemplates))
			})
		}
	}
}

// Check the query's data contract against actual interpreter output. This does
// not replace executing the volumes query against a deployed Ledger v2 server.
func TestIssuingAuthorizationsPopulateActiveHoldQueryFields(t *testing.T) {
	schema, err := loadRecipeSchema("issuing-financial-host")
	require.NoError(t, err)
	query := schema.V2QueryTemplates["ACTIVE_HOLDS_FOR_CARD"].V2VolumeQueryTemplate
	require.NotNil(t, query)
	require.Equal(t, map[string]any{"$and": []any{
		map[string]any{"$match": map[string]any{"address": "cardholder::hold:"}},
		map[string]any{"$match": map[string]any{"metadata[pii_id]": "${pii_id}"}},
		map[string]any{"$gt": map[string]any{"balance[EUR/2]": float64(0)}},
	}}, query.Body)
	require.Contains(t, query.Vars, "pii_id")
	for _, template := range []string{"CARD_AUTHORIZATION_APPROVED", "CARD_AUTHORIZATION_PARTIAL", "CARD_AUTHORIZATION_INCREMENTAL"} {
		t.Run(template, func(t *testing.T) {
			parsed := ns.Parse(schema.V2TransactionTemplates[template].Script)
			require.Empty(t, parsed.GetParsingErrors())
			const hold = "cardholder:alice:hold:auth_1"
			balances := ns.Balances{"cardholder:alice:main": {"EUR/2": big.NewInt(100)}}
			if template == "CARD_AUTHORIZATION_INCREMENTAL" {
				// An incremental authorization must tag an existing hold too.
				balances[hold] = ns.AccountBalance{"EUR/2": big.NewInt(20)}
			}
			vars := map[string]string{
				"asset": "EUR/2", "amount": "150", "overdraft": "100",
				"account_id": "alice", "authorization_id": "auth_1",
				"pii_id": "card_1", "trx_details": "Synthetic authorization",
			}
			if template == "CARD_AUTHORIZATION_PARTIAL" {
				vars["overdraft"] = "0" // Only the available 100 can be held.
			}
			result, err := parsed.Run(t.Context(), vars, ns.StaticStore{Balances: balances})
			require.NoError(t, err)
			require.NotEmpty(t, result.Postings)
			for _, posting := range result.Postings {
				require.Equal(t, hold, posting.Destination)
				catalogAddBalance(balances, posting.Source, posting.Asset, new(big.Int).Neg(posting.Amount))
				catalogAddBalance(balances, posting.Destination, posting.Asset, posting.Amount)
			}
			// All three conjuncts of ACTIVE_HOLDS_FOR_CARD must hold for the
			// account, including account metadata rather than transaction metadata.
			require.Positive(t, balances[hold]["EUR/2"].Sign())
			require.Equal(t, vars["pii_id"], result.AccountsMetadata[hold]["pii_id"])
			require.Len(t, result.AccountsMetadata, 1, "only the funded hold needs the instrument tag")
		})
	}
}

func catalogAddBalance(balances ns.Balances, account, asset string, amount *big.Int) {
	if balances[account] == nil {
		balances[account] = ns.AccountBalance{}
	}
	if balances[account][asset] == nil {
		balances[account][asset] = new(big.Int)
	}
	balances[account][asset].Add(balances[account][asset], amount)
}

func catalogAccountMatches(types map[string]ledgerv3.AddAccountTypeRequest, account string) bool {
	parts := strings.Split(account, ":")
	for _, definition := range types {
		pattern := strings.Split(definition.Pattern, ":")
		if len(pattern) != len(parts) {
			continue
		}
		matches := true
		for index, segment := range pattern {
			if variable, ok := strings.CutPrefix(segment, "{"); ok && strings.HasSuffix(variable, "}") {
				variable = strings.TrimSuffix(variable, "}")
				if parts[index] == "" {
					matches = false
					break
				}
				constraint := definition.SegmentTypes[variable]
				if constraint != nil && constraint.Regex != nil {
					matcher, err := regexp.Compile(*constraint.Regex)
					if err != nil {
						panic(fmt.Sprintf("invalid official chart regex %q: %v", *constraint.Regex, err))
					}
					if !matcher.MatchString(parts[index]) {
						matches = false
						break
					}
				}
			} else if segment != parts[index] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

// Concurrent HTTP requests commit atomically in some order. Interleave two
// independent scenarios against one ledger store to exercise shared fixed
// platform accounts as well as the actor addresses isolated by namespaces.
func TestCatalogParallelScenariosShareLedgerSafely(t *testing.T) {
	for _, name := range recipeNames() {
		t.Run(name, func(t *testing.T) {
			schema, err := loadRecipeSchema(name)
			require.NoError(t, err)
			scenarios := []scenario{newRecipeScenario(name, "worker_000"), newRecipeScenario(name, "worker_001")}
			for lane, instance := range scenarios {
				if random, ok := instance.(*omnibus); ok {
					random.random = rand.New(rand.NewPCG(uint64(42+lane), 99))
				}
				if planned, ok := instance.(*plannedScenario); ok {
					build := planned.build
					planned.build = func(c cycleContext) []recipeStep { c.amount = 10000 + int64((c.cycle*373)%9901); return build(c) }
				}
			}
			order := rand.New(rand.NewPCG(41, 91))
			parses := map[string]ns.ParseResult{}
			balances := ns.Balances{}
			metadata := ns.AccountsMetadata{}
			references := map[string]bool{}
			for index := range 1000 {
				lane := order.IntN(len(scenarios))
				request, commit := scenarios[lane].next("demo-test", "shared-run")
				require.False(t, references[*request.IdempotencyKey], "cross-stream reference collision")
				references[*request.IdempotencyKey] = true
				script := request.V2PostTransaction.Script
				source := ""
				if script.Template != nil {
					source = schema.V2TransactionTemplates[*script.Template].Script
				} else {
					source = *script.Plain
				}
				parsed, exists := parses[source]
				if !exists {
					parsed = ns.Parse(source)
					require.Empty(t, parsed.GetParsingErrors())
					parses[source] = parsed
				}
				result, err := parsed.Run(t.Context(), script.Vars, ns.StaticStore{Balances: balances, Meta: metadata})
				require.NoError(t, err, "shared ledger recipe=%s step=%d lane=%d template=%s vars=%v", name, index, lane, request.V2PostTransaction.V2Metadata["demo.template"], script.Vars)
				require.NotEmpty(t, result.Postings)
				for _, posting := range result.Postings {
					catalogAddBalance(balances, posting.Source, posting.Asset, new(big.Int).Neg(posting.Amount))
					catalogAddBalance(balances, posting.Destination, posting.Asset, posting.Amount)
				}
				for account, values := range result.AccountsMetadata {
					if metadata[account] == nil {
						metadata[account] = map[string]string{}
					}
					maps.Copy(metadata[account], values)
				}
				commit()
			}
		})
	}
}
