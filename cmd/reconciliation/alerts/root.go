package alerts

import (
	"fmt"
	"net/http"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/reconciliation/internal/clarity"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const queryFlag = "query"

func NewCommand() *cobra.Command {
	return fctl.NewCommand("alerts",
		fctl.WithAliases("alert"),
		fctl.WithShortDescription("Manage Ledger Clarity alerts"),
		fctl.WithChildCommands(
			newListCommand(),
			newGetCommand(),
			newEventsCommand(),
			newAckCommand(),
			newResolveCommand(),
			newAcceptCommand(),
			newSnoozeCommand(),
			newUnsnoozeCommand(),
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
		var response clarity.CursorResponse[clarity.Alert]
		if err := client.Do(cmd, http.MethodGet, "/alerts", query, body, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"alerts": response.Cursor.Data, "cursor": clarity.Page(response.Cursor)}, nil
	}, renderAlerts)
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List alerts"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithStringFlag(queryFlag, "", "Query-builder expression as JSON"),
		fctl.WithController[clarity.Store](controller),
	)
}

func newGetCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		var response clarity.DataResponse[clarity.Alert]
		if err := client.Do(cmd, http.MethodGet, alertPath(args[0]), nil, nil, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"alert": response.Data}, nil
	}, nil)
	return fctl.NewCommand("get <alertID>",
		fctl.WithAliases("show", "sh", "s"),
		fctl.WithShortDescription("Get an alert"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	)
}

func newEventsCommand() *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		query, err := clarity.PaginationQuery(cmd)
		if err != nil {
			return nil, err
		}
		var response clarity.CursorResponse[clarity.AlertEvent]
		if err := client.Do(cmd, http.MethodGet, alertPath(args[0])+"/events", query, nil, &response); err != nil {
			return nil, err
		}
		return clarity.Store{"events": response.Cursor.Data, "cursor": clarity.Page(response.Cursor)}, nil
	}, renderEvents)
	return fctl.NewCommand("events <alertID>",
		fctl.WithShortDescription("List an alert's append-only event timeline"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	)
}

func newAckCommand() *cobra.Command {
	return newTransitionCommand("ack <alertID>", "Acknowledge an alert", "ack", []string{"by"}, func(cmd *cobra.Command) map[string]any {
		return optionalNote(map[string]any{"by": fctl.GetString(cmd, "by")}, cmd)
	},
		fctl.WithStringFlag("by", "", "Actor acknowledging the alert"),
		fctl.WithStringFlag("note", "", "Optional acknowledgement note"),
	)
}

func newResolveCommand() *cobra.Command {
	return newTransitionCommand("resolve <alertID>", "Resolve an alert as fixed by booking", "resolve", []string{"by"}, func(cmd *cobra.Command) map[string]any {
		body := optionalNote(map[string]any{"by": fctl.GetString(cmd, "by")}, cmd)
		refs, _ := cmd.Flags().GetStringArray("transaction-ref")
		if len(refs) > 0 {
			body["transactionRefs"] = refs
		}
		return body
	},
		fctl.WithStringFlag("by", "", "Actor resolving the alert"),
		fctl.WithStringFlag("note", "", "Optional resolution note"),
		fctl.WithStringArrayFlag("transaction-ref", nil, "Related transaction reference (repeatable)"),
	)
}

func newAcceptCommand() *cobra.Command {
	return newTransitionCommand("accept <alertID>", "Accept an alert as a business exception", "accept", []string{"by", "note"}, func(cmd *cobra.Command) map[string]any {
		return map[string]any{"by": fctl.GetString(cmd, "by"), "note": fctl.GetString(cmd, "note")}
	},
		fctl.WithStringFlag("by", "", "Actor accepting the alert"),
		fctl.WithStringFlag("note", "", "Required business justification"),
	)
}

func newSnoozeCommand() *cobra.Command {
	return newTransitionCommand("snooze <alertID>", "Snooze an alert's notifications", "snooze", []string{"by", "until"}, func(cmd *cobra.Command) map[string]any {
		return optionalNote(map[string]any{"by": fctl.GetString(cmd, "by"), "until": fctl.GetString(cmd, "until")}, cmd)
	},
		fctl.WithStringFlag("by", "", "Actor snoozing the alert"),
		fctl.WithStringFlag("until", "", "Future snooze deadline (RFC3339)"),
		fctl.WithStringFlag("note", "", "Optional snooze note"),
	)
}

func newUnsnoozeCommand() *cobra.Command {
	return newTransitionCommand("unsnooze <alertID>", "Lift an alert snooze early", "unsnooze", []string{"by"}, func(cmd *cobra.Command) map[string]any {
		return map[string]any{"by": fctl.GetString(cmd, "by")}
	}, fctl.WithStringFlag("by", "", "Actor lifting the snooze"))
}

func newTransitionCommand(use, description, endpoint string, required []string, body func(*cobra.Command) map[string]any, options ...fctl.CommandOption) *cobra.Command {
	controller := clarity.NewController(func(cmd *cobra.Command, args []string, client *clarity.Client) (clarity.Store, error) {
		if !fctl.CheckStackApprobation(cmd, "You are about to %s alert '%s'", endpoint, args[0]) {
			return nil, fctl.ErrMissingApproval
		}
		var response clarity.DataResponse[clarity.Alert]
		if err := client.Do(cmd, http.MethodPost, alertPath(args[0])+"/"+endpoint, nil, body(cmd), &response); err != nil {
			return nil, err
		}
		return clarity.Store{"alert": response.Data}, nil
	}, nil)
	baseOptions := []fctl.CommandOption{
		fctl.WithShortDescription(description),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithConfirmFlag(),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[clarity.Store](controller),
	}
	cmd := fctl.NewCommand(use, append(baseOptions, options...)...)
	for _, flag := range required {
		_ = cmd.MarkFlagRequired(flag)
	}
	return cmd
}

func optionalNote(body map[string]any, cmd *cobra.Command) map[string]any {
	if note := fctl.GetString(cmd, "note"); note != "" {
		body["note"] = note
	}
	return body
}

func alertPath(id string) string { return "/alerts/" + clarity.ResourcePath(id) }

func renderAlerts(cmd *cobra.Command, store clarity.Store) error {
	alerts := store["alerts"].([]clarity.Alert)
	rows := pterm.TableData{{"ID", "RuleID", "Fingerprint", "Period", "Status", "Severity", "Occurrences", "LastSeenAt"}}
	for _, alert := range alerts {
		rows = append(rows, []string{alert.ID, alert.RuleID, alert.Fingerprint, alert.PeriodID, alert.Status, alert.Severity, fmt.Sprint(alert.OccurrenceCount), alert.LastSeenAt})
	}
	if err := pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(rows).Render(); err != nil {
		return err
	}
	cursor := store["cursor"].(clarity.PageCursor)
	return fctl.RenderCursor(cmd.OutOrStdout(), fctl.Cursor{HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous})
}

func renderEvents(cmd *cobra.Command, store clarity.Store) error {
	events := store["events"].([]clarity.AlertEvent)
	rows := pterm.TableData{{"ID", "Type", "Previous", "NewStatus", "At", "Reopen", "Notify", "EvaluationID"}}
	for _, event := range events {
		previous, evaluationID := "", ""
		if event.Previous != nil {
			previous = *event.Previous
		}
		if event.EvaluationID != nil {
			evaluationID = *event.EvaluationID
		}
		rows = append(rows, []string{event.ID, event.Type, previous, event.NewStatus, event.At, fmt.Sprint(event.IsReopen), fmt.Sprint(event.Notify), evaluationID})
	}
	if err := pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(rows).Render(); err != nil {
		return err
	}
	cursor := store["cursor"].(clarity.PageCursor)
	return fctl.RenderCursor(cmd.OutOrStdout(), fctl.Cursor{HasMore: cursor.HasMore, PageSize: cursor.PageSize, Next: cursor.Next, Previous: cursor.Previous})
}
