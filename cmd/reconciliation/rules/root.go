package rules

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/reconciliation/internal/clarity"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const queryFlag = "query"

func NewCommand() *cobra.Command {
	return fctl.NewCommand("rules",
		fctl.WithAliases("rule"),
		fctl.WithShortDescription("Manage Ledger Clarity rules"),
		fctl.WithChildCommands(
			newListCommand(),
			newGetCommand(),
			newCreateCommand(),
			newUpdateCommand(),
			newDeleteCommand(),
			newEvaluateCommand(),
		),
	)
}

func newListCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, _ []string, client *clarity.Client) (clarity.Store, error) {
		query, err := clarity.PaginationQuery(cmd)
		if err != nil {
			return nil, err
		}
		var body any
		if query.Get("cursor") == "" {
			body, err = clarity.QueryBody(cmd, queryFlag)
			if err != nil {
				return nil, err
			}
		}
		var response clarity.CursorResponse[clarity.Rule]
		if err := client.Do(cmd, http.MethodGet, "/rules", query, body, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"rules": response.Cursor.Data, "cursor": clarity.Page(response.Cursor)}, nil
	}, renderRules)
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List rules"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithStringFlag(queryFlag, "", "Query-builder expression as JSON"),
		fctl.WithController[clarity.Store](controller),
	)
}

func newGetCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		var response clarity.DataResponse[clarity.Rule]
		if err := client.Do(cmd, http.MethodGet, "/rules/"+clarity.ResourcePath(args[0]), nil, nil, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"rule": response.Data}, nil
	}, nil)
	return fctl.NewCommand("get <ruleID>",
		fctl.WithAliases("show", "sh", "s"),
		fctl.WithShortDescription("Get a rule"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	)
}

func newCreateCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		if !fctl.CheckStackApprobation(cmd, "You are about to create a reconciliation rule") {
			return nil, fctl.ErrMissingApproval
		}
		body, err := clarity.ReadJSONObject(cmd, args[0])
		if err != nil {
			return nil, err
		}
		var response clarity.DataResponse[clarity.Rule]
		if err := client.Do(cmd, http.MethodPost, "/rules", nil, body, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"rule": response.Data}, nil
	}, func(cmd *cobra.Command, store clarity.Store) error {
		rule := store["rule"].(clarity.Rule)
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Rule created with ID: %s", rule.ID)
		return nil
	})
	return fctl.NewCommand("create <file>|-",
		fctl.WithAliases("cr", "c"),
		fctl.WithShortDescription("Create a rule from a JSON file"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithController[clarity.Store](controller),
	)
}

func newUpdateCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		if !fctl.CheckStackApprobation(cmd, "You are about to update rule '%s'", args[0]) {
			return nil, fctl.ErrMissingApproval
		}
		body, err := clarity.ReadJSONObject(cmd, args[1])
		if err != nil {
			return nil, err
		}
		var response clarity.DataResponse[clarity.Rule]
		if err := client.Do(cmd, http.MethodPatch, "/rules/"+clarity.ResourcePath(args[0]), nil, body, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"rule": response.Data}, nil
	}, nil)
	return fctl.NewCommand("update <ruleID> <file>|-",
		fctl.WithAliases("patch"),
		fctl.WithShortDescription("Partially update a rule from a JSON file"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithConfirmFlag(),
		fctl.WithController[clarity.Store](controller),
	)
}

func newDeleteCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		if !fctl.CheckStackApprobation(cmd, "You are about to delete rule '%s' and its evaluations and alerts", args[0]) {
			return nil, fctl.ErrMissingApproval
		}
		if err := client.Do(cmd, http.MethodDelete, "/rules/"+clarity.ResourcePath(args[0]), nil, nil, nil); err != nil {
			return nil, err
		}
		return clarity.Store{"ruleID": args[0], "success": true}, nil
	}, func(cmd *cobra.Command, store clarity.Store) error {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Rule %s deleted.", store["ruleID"])
		return nil
	})
	return fctl.NewCommand("delete <ruleID>",
		fctl.WithAliases("d"),
		fctl.WithShortDescription("Delete a rule"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	)
}

func newEvaluateCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		body := map[string]any{}
		if at := fctl.GetString(cmd, "at"); at != "" {
			body["at"] = at
		}
		if margin := fctl.GetString(cmd, "safety-margin"); margin != "" {
			body["safetyMargin"] = margin
		}
		sourcePITValues, err := cmd.Flags().GetStringArray("source-pit")
		if err != nil {
			return nil, err
		}
		if len(sourcePITValues) > 0 {
			sourcePITs := map[string]string{}
			for _, item := range sourcePITValues {
				parts := strings.SplitN(item, "=", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return nil, fmt.Errorf("invalid --source-pit %q: expected key=RFC3339", item)
				}
				sourcePITs[parts[0]] = parts[1]
			}
			body["sourcePITs"] = sourcePITs
		}
		var response clarity.DataResponse[clarity.Evaluation]
		path := "/rules/" + clarity.ResourcePath(args[0]) + "/evaluate"
		if err := client.Do(cmd, http.MethodPost, path, nil, body, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"evaluation": response.Data}, nil
	}, nil)
	return fctl.NewCommand("evaluate <ruleID>",
		fctl.WithAliases("run"),
		fctl.WithShortDescription("Evaluate a rule now"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithStringFlag("at", "", "Canonical point in time (RFC3339; defaults to now)"),
		fctl.WithStringFlag("safety-margin", "", "Safety margin as a Go duration (for example 30s)"),
		fctl.WithStringArrayFlag("source-pit", nil, "Per-source PIT as key=RFC3339 (repeatable)"),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	)
}

func renderRules(cmd *cobra.Command, store clarity.Store) error {
	rules := store["rules"].([]clarity.Rule)
	rows := pterm.TableData{{"ID", "Name", "Template", "Enabled", "Severity", "Cadence", "UpdatedAt"}}
	for _, rule := range rules {
		rows = append(rows, []string{rule.ID, rule.Name, rule.TemplateKind, fmt.Sprint(rule.Enabled), rule.Severity, rule.Cadence, rule.UpdatedAt})
	}
	if err := pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(rows).Render(); err != nil {
		return err
	}
	cursor := store["cursor"].(clarity.PageCursor)
	return fctl.RenderCursor(cmd.OutOrStdout(), fctl.Cursor{HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous})
}
