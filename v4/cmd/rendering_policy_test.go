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
		"newContextListCommand":                                    true,
		"newLedgerExportCommand":                                   true,
		"newPaymentsVersionsCommand":                               true,
		"newSessionLoginClientCredentialsCommand":                  true,
		"newSessionLoginNoneCommand":                               true,
		"newSessionLoginTokenCommand":                              true,
		"newSessionLogoutCommand":                                  true,
		"newSessionStatusCommand":                                  true,
		"newSessionTokenCommand":                                   true,
		"newTargetInspectCommand":                                  true,
		"newVersionCommand":                                        true,
		"promptValue":                                              true,
		"renderCloudApp":                                           true,
		"renderCloudApps":                                          true,
		"renderCloudRun":                                           true,
		"renderCloudRunLogs":                                       true,
		"renderCloudRuns":                                          true,
		"renderCloudVariable":                                      true,
		"renderCloudVariables":                                     true,
		"renderCloudVersion":                                       true,
		"renderCloudVersions":                                      true,
		"renderMigrationPlan":                                      true,
		"renderPaymentConnectorConfig":                             true,
		"renderSetupGuidance":                                      true,
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
