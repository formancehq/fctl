package evaluations

import (
	"net/http"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/reconciliation/internal/clarity"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const queryFlag = "query"

func NewCommand() *cobra.Command {
	return fctl.NewCommand("evaluations",
		fctl.WithAliases("evaluation", "evals"),
		fctl.WithShortDescription("Inspect Ledger Clarity evaluations"),
		fctl.WithChildCommands(newListCommand(), newGetCommand()),
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
		var response clarity.CursorResponse[clarity.Evaluation]
		if err := client.Do(cmd, http.MethodGet, "/evaluations", query, body, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"evaluations": response.Cursor.Data, "cursor": clarity.Page(response.Cursor)}, nil
	}, renderEvaluations)
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List evaluations"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithStringFlag(queryFlag, "", "Query-builder expression as JSON"),
		fctl.WithController[clarity.Store](controller),
	)
}

func newGetCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		var response clarity.DataResponse[clarity.Evaluation]
		path := "/evaluations/" + clarity.ResourcePath(args[0])
		if err := client.Do(cmd, http.MethodGet, path, nil, nil, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"evaluation": response.Data}, nil
	}, nil)
	return fctl.NewCommand("get <evaluationID>",
		fctl.WithAliases("show", "sh", "s"),
		fctl.WithShortDescription("Get an evaluation"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	)
}

func renderEvaluations(cmd *cobra.Command, store clarity.Store) error {
	evaluations := store["evaluations"].([]clarity.Evaluation)
	rows := pterm.TableData{{"ID", "RuleID", "Result", "StartedAt", "EndedAt", "CostUnits"}}
	for _, evaluation := range evaluations {
		rows = append(rows, []string{evaluation.ID, evaluation.RuleID, evaluation.Result, evaluation.StartedAt, evaluation.EndedAt, pterm.Sprintf("%d", evaluation.CostUnits)})
	}
	if err := pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(rows).Render(); err != nil {
		return err
	}
	cursor := store["cursor"].(clarity.PageCursor)
	return fctl.RenderCursor(cmd.OutOrStdout(), fctl.Cursor{HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous})
}
