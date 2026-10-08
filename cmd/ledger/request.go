package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
)

type module struct {
	runtime     command.Runtime
	ledger      string
	consistency string
}

type bodyMode uint8

const (
	bodyNone bodyMode = iota
	bodyRequired
	bodyOptional
	bodyDefault
)

type operation struct {
	use, short, method string
	args               int
	global, ledgerArg  bool
	transactionID      bool
	path               func([]string) []string
	body               bodyMode
	idempotency        bool
	page, reverse      bool
	filter, dates      bool
	inspect, bulk      bool
	boolQuery          map[string]string
	stringQuery        map[string]string
}

type inputs struct {
	data, key, cursor, filter, start, end, mode string
	confirm, reverse                            bool
	size                                        uint32
}

func (m *module) endpoint(op operation) *cobra.Command {
	if op.method == "" {
		op.method = http.MethodGet
	}
	cmd := &cobra.Command{Use: op.use, Short: op.short, Args: cobra.ExactArgs(op.args)}
	if op.ledgerArg {
		cmd.Args = cobra.MaximumNArgs(1)
	}
	in := inputs{}
	in.bindBody(cmd, op)
	in.bindQuery(cmd, op)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := validateArgs(op, args); err != nil {
			return err
		}
		path, err := m.path(op, args)
		if err != nil {
			return err
		}
		headers, err := m.headers(op, in)
		if err != nil {
			return err
		}
		query, err := in.query(cmd, op)
		if err != nil {
			return err
		}
		body, err := in.readBody(cmd, op)
		if err != nil {
			return err
		}
		if op.bulk {
			return m.runBulk(cmd, path, query, body, headers)
		}
		return m.runtime.Run(cmd, "ledger", op.method, path, query, body, headers)
	}
	return cmd
}

func (in *inputs) bindBody(cmd *cobra.Command, op operation) {
	if op.body != bodyNone {
		if op.body == bodyDefault {
			in.data = "{}"
		}
		cmd.Flags().StringVar(&in.data, "data", in.data, "JSON request body: inline JSON, @file or - for stdin")
	}
	if op.idempotency {
		cmd.Flags().StringVar(&in.key, "idempotency-key", "", "Stable request identity (max 256 characters); never generated automatically")
	}
	if op.method == http.MethodDelete {
		cmd.Flags().BoolVar(&in.confirm, "confirm", false, "Confirm this destructive deletion")
	}
}

func (in *inputs) bindQuery(cmd *cobra.Command, op operation) {
	if op.page || op.inspect {
		usage := "Results per page (0 means 100; server caps at 1000)"
		if op.inspect {
			usage = "Results per index page (1 through 10000)"
		}
		cmd.Flags().Uint32Var(&in.size, "page-size", 100, usage)
		cmd.Flags().StringVar(&in.cursor, "cursor", "", "Opaque next or previous page token from the server")
	}
	if op.reverse {
		cmd.Flags().BoolVar(&in.reverse, "reverse", false, "Reverse the endpoint's default ordering")
	}
	if op.filter {
		cmd.Flags().StringVar(&in.filter, "filter", "", "Exact v3 filter expression or structured JSON filter")
	}
	if op.dates {
		cmd.Flags().StringVar(&in.start, "start-date", "", "Inclusive start date (RFC3339)")
		cmd.Flags().StringVar(&in.end, "end-date", "", "Exclusive end date (RFC3339)")
	}
	if op.inspect {
		cmd.Flags().StringVar(&in.mode, "mode", "summary", "Index inspection: summary, distinctValues or facets")
	}
	bindOptions(cmd, op.boolQuery, op.stringQuery)
}

func bindOptions(cmd *cobra.Command, bools, texts map[string]string) {
	for flag, query := range bools {
		cmd.Flags().Bool(flag, false, "Set the v3 "+query+" query option")
	}
	for flag, query := range texts {
		cmd.Flags().String(flag, "", "Set the v3 "+query+" query option")
	}
}

func validateArgs(op operation, args []string) error {
	if slices.Contains(args, "") {
		return fmt.Errorf("resource identifiers must not be empty")
	}
	if op.transactionID {
		if _, err := strconv.ParseUint(args[0], 10, 64); err != nil {
			return fmt.Errorf("transaction id must be an unsigned 64-bit integer: %w", err)
		}
	}
	return nil
}

func (m *module) headers(op operation, in inputs) (http.Header, error) {
	if op.method == http.MethodDelete && !in.confirm {
		return nil, fmt.Errorf("deletion requires --confirm")
	}
	if len(in.key) > 256 || strings.ContainsAny(in.key, "\r\n") {
		return nil, fmt.Errorf("idempotency-key must be at most 256 bytes without newlines")
	}
	headers := make(http.Header)
	if in.key != "" {
		headers.Set("Idempotency-Key", in.key)
	}
	if m.consistency != "" {
		consistency := strings.ToLower(strings.TrimSpace(m.consistency))
		if consistency != "linearizable" && consistency != "stale" {
			return nil, fmt.Errorf("consistency must be linearizable or stale")
		}
		if !op.global || op.page {
			headers.Set("X-Consistency", consistency)
		}
	}
	return headers, nil
}

func (in inputs) query(cmd *cobra.Command, op operation) (url.Values, error) {
	query := make(url.Values)
	if op.page || op.inspect {
		query.Set("pageSize", strconv.FormatUint(uint64(in.size), 10))
		if in.cursor != "" {
			query.Set("cursor", in.cursor)
		}
	}
	if op.reverse && cmd.Flags().Changed("reverse") {
		query.Set("reverse", strconv.FormatBool(in.reverse))
	}
	if in.filter != "" {
		query.Set("filter", in.filter)
	}
	if err := in.dateQuery(query); err != nil {
		return nil, err
	}
	if op.inspect {
		if err := in.inspectQuery(query); err != nil {
			return nil, err
		}
	}
	copyOptions(cmd, query, op.boolQuery)
	copyOptions(cmd, query, op.stringQuery)
	return query, nil
}

func (in inputs) dateQuery(query url.Values) error {
	for name, value := range map[string]string{"startDate": in.start, "endDate": in.end} {
		if value == "" {
			continue
		}
		date, err := time.Parse(time.RFC3339Nano, value)
		if err != nil || date.Before(time.Unix(0, 0)) {
			return fmt.Errorf("%s must be an RFC3339 date on or after the Unix epoch", name)
		}
		query.Set(name, value)
	}
	return nil
}

func (in inputs) inspectQuery(query url.Values) error {
	if in.mode != "summary" && in.mode != "distinctValues" && in.mode != "facets" {
		return fmt.Errorf("mode must be summary, distinctValues or facets")
	}
	if in.size < 1 || in.size > 10000 {
		return fmt.Errorf("index page-size must be between 1 and 10000")
	}
	query.Set("mode", in.mode)
	return nil
}

func copyOptions(cmd *cobra.Command, query url.Values, options map[string]string) {
	for flag, param := range options {
		if cmd.Flags().Changed(flag) {
			query.Set(param, cmd.Flags().Lookup(flag).Value.String())
		}
	}
}

func (in inputs) readBody(cmd *cobra.Command, op operation) (json.RawMessage, error) {
	if op.body == bodyNone {
		return nil, nil
	}
	if op.body == bodyRequired && in.data == "" {
		return nil, fmt.Errorf("--data is required (inline JSON, @file or -)")
	}
	if in.data != "" || cmd.Flags().Changed("data") {
		return command.ReadBody(cmd, in.data)
	}
	return nil, nil
}

// runBulk preserves the complete successful HTTP envelope and still returns an
// error for per-element business failures suppressed by continueOnFailure.
func (m *module) runBulk(cmd *cobra.Command, path string, query url.Values, body json.RawMessage, headers http.Header) error {
	client, err := m.runtime.Client(cmd.Context(), "ledger")
	if err != nil {
		return err
	}
	result, err := client.Do(cmd.Context(), http.MethodPost, path, query, body, headers)
	if err != nil {
		if failure, ok := errors.AsType[*api.Error](err); ok && len(failure.Body) > 0 {
			if writeErr := command.WriteJSON(cmd.OutOrStdout(), failure.Body); writeErr != nil {
				return errors.Join(err, writeErr)
			}
		}
		return err
	}
	if err := command.WriteJSON(cmd.OutOrStdout(), result); err != nil {
		return err
	}
	var response struct {
		ErrorCode string `json:"errorCode"`
		Data      []struct {
			ErrorCode    string `json:"errorCode"`
			ResponseType string `json:"responseType"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("invalid bulk response: %w", err)
	}
	if response.ErrorCode != "" {
		return fmt.Errorf("bulk failed (%s); inspect the JSON response", response.ErrorCode)
	}
	for i, item := range response.Data {
		if item.ErrorCode != "" || item.ResponseType == "ERROR" {
			return fmt.Errorf("bulk element %d failed (%s); inspect the JSON response", i, item.ErrorCode)
		}
	}
	return nil
}
