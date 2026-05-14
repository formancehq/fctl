package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestHumanOutputUsesStyledRendering(t *testing.T) {
	allowedRawOutputFunctions := map[string]bool{
		"choosePlatformAuth":                                       true,
		"chooseTarget":                                             true,
		"input":                                                    true,
		"newCloudAppsCreateCommand":                                true,
		"newCloudAppsDeleteCommand":                                true,
		"newCloudAppsDeployCommand":                                true,
		"newCloudAppsVariablesDeleteCommand":                       true,
		"newCloudAppsVersionsArchiveShowCommand":                   true,
		"newCloudAppsVersionsManifestCommand":                      true,
		"newCloudOrganizationsAuthenticationProviderDeleteCommand": true,
		"newCloudOrganizationsDeleteCommand":                       true,
		"newCloudOrganizationsInvitationsDeleteCommand":            true,
		"newCloudOrganizationsInvitationsSendCommand":              true,
		"newCloudOrganizationsOAuthClientsDeleteCommand":           true,
		"newCloudRegionsDeleteCommand":                             true,
		"newConfigMigrateV3Command":                                true,
		"newContextCreateCloudCommand":                             true,
		"newContextCreateCloudStackCommand":                        true,
		"newContextCreateStackCommand":                             true,
		"newContextDeleteCommand":                                  true,
		"newContextListCommand":                                    true,
		"newContextRenameCommand":                                  true,
		"newContextSetCommand":                                     true,
		"newContextShowCommand":                                    true,
		"newContextUnsetDefaultsCommand":                           true,
		"newContextUseCommand":                                     true,
		"newLedgerExportCommand":                                   true,
		"newLoginCommand":                                          true,
		"newLogoutCommand":                                         true,
		"newPaymentsVersionsCommand":                               true,
		"newProfilesSetDefaultOrganizationCommand":                 true,
		"newProfilesSetDefaultStackCommand":                        true,
		"newSessionLoginClientCredentialsCommand":                  true,
		"newSessionLoginNoneCommand":                               true,
		"newSessionLoginTokenCommand":                              true,
		"newSessionLogoutCommand":                                  true,
		"newSessionStatusCommand":                                  true,
		"newSessionTokenCommand":                                   true,
		"newTargetInspectCommand":                                  true,
		"newTargetProxyCommand":                                    true,
		"newUICommand":                                             true,
		"newVersionCommand":                                        true,
		"newWhoamiCommand":                                         true,
		"promptValue":                                              true,
		"renderAuthClient":                                         true,
		"renderAuthClientDeleted":                                  true,
		"renderAuthClientMutated":                                  true,
		"renderAuthClients":                                        true,
		"renderAuthSecretCreated":                                  true,
		"renderAuthSecretDeleted":                                  true,
		"renderAuthUser":                                           true,
		"renderAuthUsers":                                          true,
		"renderCloudApp":                                           true,
		"renderCloudApps":                                          true,
		"renderCloudRun":                                           true,
		"renderCloudRunLogs":                                       true,
		"renderCloudRuns":                                          true,
		"renderCloudVariable":                                      true,
		"renderCloudVariables":                                     true,
		"renderCloudVersion":                                       true,
		"renderCloudVersions":                                      true,
		"renderFlowsInstance":                                      true,
		"renderFlowsInstanceEventSent":                             true,
		"renderFlowsInstanceStopped":                               true,
		"renderFlowsInstances":                                     true,
		"renderFlowsTrigger":                                       true,
		"renderFlowsTriggerCreated":                                true,
		"renderFlowsTriggerDeleted":                                true,
		"renderFlowsTriggerOccurrences":                            true,
		"renderFlowsTriggerTest":                                   true,
		"renderFlowsTriggers":                                      true,
		"renderFlowsWorkflow":                                      true,
		"renderFlowsWorkflowCreated":                               true,
		"renderFlowsWorkflowDeleted":                               true,
		"renderFlowsWorkflowRun":                                   true,
		"renderFlowsWorkflows":                                     true,
		"renderLedgerAccount":                                      true,
		"renderLedgerAccountMetadataDeleted":                       true,
		"renderLedgerAccountMetadataSet":                           true,
		"renderLedgerAccountQuery":                                 true,
		"renderLedgerAccounts":                                     true,
		"renderLedgerCreated":                                      true,
		"renderLedgerExported":                                     true,
		"renderLedgerImported":                                     true,
		"renderLedgerInfo":                                         true,
		"renderLedgerMetadataDeleted":                              true,
		"renderLedgerMetadataSet":                                  true,
		"renderLedgerRevertedTransaction":                          true,
		"renderLedgerSchema":                                       true,
		"renderLedgerSchemaInserted":                               true,
		"renderLedgerSchemas":                                      true,
		"renderLedgerSentTransaction":                              true,
		"renderLedgerStats":                                        true,
		"renderLedgerTransaction":                                  true,
		"renderLedgerTransactionMetadataDeleted":                   true,
		"renderLedgerTransactionMetadataSet":                       true,
		"renderLedgerTransactions":                                 true,
		"renderLedgerTransactionsCount":                            true,
		"renderLedgerVolumes":                                      true,
		"renderLedgers":                                            true,
		"renderMigrationPlan":                                      true,
		"renderPayment":                                            true,
		"renderPaymentAccount":                                     true,
		"renderPaymentAccountBalances":                             true,
		"renderPaymentAccountCreated":                              true,
		"renderPaymentAccounts":                                    true,
		"renderPaymentBankAccount":                                 true,
		"renderPaymentBankAccountCreated":                          true,
		"renderPaymentBankAccountForwarded":                        true,
		"renderPaymentBankAccountMetadataSet":                      true,
		"renderPaymentBankAccounts":                                true,
		"renderPaymentConnectorConfig":                             true,
		"renderPaymentConnectorConfigUpdated":                      true,
		"renderPaymentConnectorInstalled":                          true,
		"renderPaymentConnectorUninstalled":                        true,
		"renderPaymentConnectors":                                  true,
		"renderPaymentCreated":                                     true,
		"renderPaymentMetadataSet":                                 true,
		"renderPaymentPool":                                        true,
		"renderPaymentPoolAccountAdded":                            true,
		"renderPaymentPoolAccountRemoved":                          true,
		"renderPaymentPoolBalances":                                true,
		"renderPaymentPoolCreated":                                 true,
		"renderPaymentPoolDeleted":                                 true,
		"renderPaymentPoolQueryUpdated":                            true,
		"renderPaymentPools":                                       true,
		"renderPaymentTask":                                        true,
		"renderPaymentTransferInitiation":                          true,
		"renderPaymentTransferInitiationAction":                    true,
		"renderPaymentTransferInitiationCreated":                   true,
		"renderPaymentTransferInitiationReversed":                  true,
		"renderPaymentTransferInitiationStatusUpdated":             true,
		"renderPaymentTransferInitiations":                         true,
		"renderPayments":                                           true,
		"renderReconciliation":                                     true,
		"renderReconciliationPolicies":                             true,
		"renderReconciliationPolicy":                               true,
		"renderReconciliationPolicyCreated":                        true,
		"renderReconciliationPolicyDeleted":                        true,
		"renderReconciliationStarted":                              true,
		"renderReconciliations":                                    true,
		"renderSetupGuidance":                                      true,
		"renderWallet":                                             true,
		"renderWalletBalance":                                      true,
		"renderWalletBalanceCreated":                               true,
		"renderWalletBalances":                                     true,
		"renderWalletCreated":                                      true,
		"renderWalletCredited":                                     true,
		"renderWalletDebited":                                      true,
		"renderWalletHold":                                         true,
		"renderWalletHoldConfirmed":                                true,
		"renderWalletHoldVoided":                                   true,
		"renderWalletHolds":                                        true,
		"renderWalletTransactions":                                 true,
		"renderWalletUpdated":                                      true,
		"renderWallets":                                            true,
		"renderWebhookConfigDeleted":                               true,
		"renderWebhookConfigMutated":                               true,
		"renderWebhookConfigs":                                     true,
		"report":                                                   true,
		"selectValue":                                              true,
	}

	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list cmd files: %v", err)
	}

	var violations []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "terminal_ui.go" || file == "output.go" {
			continue
		}
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		parsed, err := parser.ParseFile(fset, file, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}
			ast.Inspect(fn.Body, func(inner ast.Node) bool {
				call, ok := inner.(*ast.CallExpr)
				if !ok || !isRawCommandOutputCall(call) || allowedRawOutputFunctions[fn.Name.Name] {
					return true
				}
				pos := fset.Position(call.Pos())
				violations = append(violations, fmt.Sprintf("%s:%d in %s", pos.Filename, pos.Line, fn.Name.Name))
				return true
			})
			return false
		})
	}

	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("human plain output must use styled rendering helpers; found raw cmd.OutOrStdout writes:\n%s", strings.Join(violations, "\n"))
	}
}

func isRawCommandOutputCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if isFmtPrintSelector(selector) {
		return len(call.Args) > 0 && isCommandOutputCall(call.Args[0]) && !usesStyledOutputHelper(call.Args[1:])
	}
	return selector.Sel.Name == "Write" && isCommandOutputCall(selector.X)
}

func isFmtPrintSelector(selector *ast.SelectorExpr) bool {
	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != "fmt" {
		return false
	}
	switch selector.Sel.Name {
	case "Fprint", "Fprintf", "Fprintln":
		return true
	default:
		return false
	}
}

func isCommandOutputCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "OutOrStdout"
}

func usesStyledOutputHelper(args []ast.Expr) bool {
	for _, arg := range args {
		call, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		switch ident.Name {
		case "styledEmptyLine", "styledInfoLine", "styledKeyValueLine", "styledSuccessLine":
			return true
		}
	}
	return false
}
