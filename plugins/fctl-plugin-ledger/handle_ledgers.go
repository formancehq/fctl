package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/formancehq/fctl-plugin-ledger/internal"
	"github.com/formancehq/fctl/pkg/pluginsdk/pluginpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/commonpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/servicepb"
	"github.com/pterm/pterm"
)

func handleLedgersList(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	client, conn, err := internal.NewClient(req.Flags)
	if err != nil {
		return errorResponse(err.Error(), 1), nil
	}
	defer conn.Close()

	ctx = internal.ContextWithAuth(ctx, req)

	stream, err := client.ListLedgers(ctx, &servicepb.ListLedgersRequest{})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to list ledgers: %v", err), 1), nil
	}

	ledgers := make(map[string]*commonpb.LedgerInfo)
	for {
		ledger, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errorResponse(fmt.Sprintf("failed to receive ledger: %v", err), 1), nil
		}
		ledgers[ledger.Name] = ledger
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(ledgers, "", "  ")
		return successResponse(string(data), ""), nil
	}

	names := make([]string, 0, len(ledgers))
	for name := range ledgers {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		return successResponse("", "No ledgers found.\n"), nil
	}

	var buf bytes.Buffer
	tableData := pterm.TableData{{"NAME", "CREATED AT"}}
	for _, name := range names {
		ledger := ledgers[name]
		createdAt := "-"
		if ledger.CreatedAt != nil {
			createdAt = ledger.CreatedAt.AsTime().Format(time.RFC3339)
		}
		tableData = append(tableData, []string{ledger.Name, createdAt})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(tableData).Render()
	return successResponse("", buf.String()), nil
}

func handleLedgersCreate(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	name := req.Flags["name"]
	if name == "" {
		return errorResponse("ledger name is required (use --name flag)", 1), nil
	}

	client, conn, err := internal.NewClient(req.Flags)
	if err != nil {
		return errorResponse(err.Error(), 1), nil
	}
	defer conn.Close()

	ctx = internal.ContextWithAuth(ctx, req)

	requests := []*servicepb.Request{
		{
			Type: &servicepb.Request_CreateLedger{
				CreateLedger: &servicepb.CreateLedgerRequest{
					Name: name,
				},
			},
		},
	}

	resp, err := client.Apply(ctx, &servicepb.ApplyRequest{Requests: requests})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to create ledger: %v", err), 1), nil
	}

	if len(resp.Logs) == 0 {
		return errorResponse("no response received", 1), nil
	}

	log := resp.Logs[0]
	createLog := log.Payload.GetCreateLedger()
	if createLog == nil {
		return errorResponse("unexpected response type", 1), nil
	}

	ledger := createLog.Info

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(ledger, "", "  ")
		return successResponse(string(data), ""), nil
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Ledger created successfully!\n\n")
	fmt.Fprintf(&buf, "Name:       %s\n", ledger.Name)
	if ledger.CreatedAt != nil {
		fmt.Fprintf(&buf, "Created At: %s\n", ledger.CreatedAt.AsTime().Format(time.RFC3339))
	}

	return successResponse("", buf.String()), nil
}

func handleLedgersGet(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	if len(req.Args) == 0 {
		return errorResponse("ledger name argument is required", 1), nil
	}
	ledgerName := req.Args[0]

	client, conn, err := internal.NewClient(req.Flags)
	if err != nil {
		return errorResponse(err.Error(), 1), nil
	}
	defer conn.Close()

	ctx = internal.ContextWithAuth(ctx, req)

	ledger, err := client.GetLedger(ctx, &servicepb.GetLedgerRequest{
		Ledger: ledgerName,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to get ledger: %v", err), 1), nil
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(ledger, "", "  ")
		return successResponse(string(data), ""), nil
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Ledger: %s\n", ledger.Name)
	fmt.Fprintf(&buf, "─────────────────────────────────\n")
	fmt.Fprintf(&buf, "Name:       %s\n", ledger.Name)
	if ledger.CreatedAt != nil {
		fmt.Fprintf(&buf, "Created At: %s\n", ledger.CreatedAt.AsTime().Format(time.RFC3339))
	}

	return successResponse("", buf.String()), nil
}

func handleLedgersDelete(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	if len(req.Args) == 0 {
		return errorResponse("ledger name argument is required", 1), nil
	}
	ledgerName := req.Args[0]

	client, conn, err := internal.NewClient(req.Flags)
	if err != nil {
		return errorResponse(err.Error(), 1), nil
	}
	defer conn.Close()

	ctx = internal.ContextWithAuth(ctx, req)

	requests := []*servicepb.Request{
		{
			Type: &servicepb.Request_DeleteLedger{
				DeleteLedger: &servicepb.DeleteLedgerRequest{
					Name: ledgerName,
				},
			},
		},
	}

	_, err = client.Apply(ctx, &servicepb.ApplyRequest{Requests: requests})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to delete ledger: %v", err), 1), nil
	}

	return successResponse("", fmt.Sprintf("Ledger %s deleted successfully.\n", ledgerName)), nil
}

func handleLedgersStats(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	ledgerName := req.Flags["ledger"]
	if ledgerName == "" {
		return errorResponse("--ledger flag is required", 1), nil
	}

	client, conn, err := internal.NewClient(req.Flags)
	if err != nil {
		return errorResponse(err.Error(), 1), nil
	}
	defer conn.Close()

	ctx = internal.ContextWithAuth(ctx, req)

	stats, err := client.GetLedgerStats(ctx, &servicepb.GetLedgerStatsRequest{
		Ledger: ledgerName,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to get stats: %v", err), 1), nil
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(stats, "", "  ")
		return successResponse(string(data), ""), nil
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Ledger Stats: %s\n", ledgerName)
	fmt.Fprintf(&buf, "─────────────────────────────────\n")
	fmt.Fprintf(&buf, "Transactions: %d\n", stats.TransactionCount)
	fmt.Fprintf(&buf, "Accounts:     %d\n", stats.AccountCount)

	return successResponse("", buf.String()), nil
}
