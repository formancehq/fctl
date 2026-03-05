package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl-plugin-ledger/internal"
	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/commonpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/servicepb"
	"github.com/pterm/pterm"
)

func handleTransactionsList(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
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

	pageSize := uint32(10)
	if ps := req.Flags["page-size"]; ps != "" {
		if v, err := strconv.ParseUint(ps, 10, 32); err == nil {
			pageSize = uint32(v)
		}
	}

	fetchAll := req.Flags["all"] == "true"
	reverse := req.Flags["reverse"] == "true"

	if fetchAll {
		pageSize = 0
	}

	stream, err := client.ListTransactions(ctx, &servicepb.ListTransactionsRequest{
		Ledger:   ledgerName,
		PageSize: pageSize,
		Reverse:  reverse,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to list transactions: %v", err), 1), nil
	}

	var transactions []*commonpb.Transaction
	for {
		tx, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errorResponse(fmt.Sprintf("failed to receive transaction: %v", err), 1), nil
		}
		transactions = append(transactions, tx)
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(transactions, "", "  ")
		return successResponse(string(data), ""), nil
	}

	if len(transactions) == 0 {
		return successResponse("", "No transactions found.\n"), nil
	}

	var buf bytes.Buffer
	tableData := pterm.TableData{{"ID", "TIMESTAMP", "REFERENCE", "POSTINGS", "STATUS"}}
	for _, tx := range transactions {
		timestamp := "-"
		if tx.Timestamp != nil {
			timestamp = tx.Timestamp.AsTime().Format(time.RFC3339)
		}
		reference := "-"
		if tx.Reference != "" {
			reference = tx.Reference
		}
		status := "OK"
		if tx.Reverted {
			status = "Reverted"
		}
		tableData = append(tableData, []string{
			fmt.Sprintf("%d", tx.Id),
			timestamp,
			reference,
			fmt.Sprintf("%d", len(tx.Postings)),
			status,
		})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(tableData).Render()
	return successResponse("", buf.String()), nil
}

func handleTransactionsGet(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	if len(req.Args) == 0 {
		return errorResponse("transaction ID argument is required", 1), nil
	}

	txID, err := strconv.ParseUint(req.Args[0], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("invalid transaction ID: %s", req.Args[0]), 1), nil
	}

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

	resp, err := client.GetTransaction(ctx, &servicepb.GetTransactionRequest{
		Ledger:        ledgerName,
		TransactionId: txID,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to get transaction: %v", err), 1), nil
	}

	tx := resp.Transaction

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(tx, "", "  ")
		return successResponse(string(data), ""), nil
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Transaction: #%d\n", tx.Id)
	fmt.Fprintf(&buf, "─────────────────────────────────\n")
	if tx.Reference != "" {
		fmt.Fprintf(&buf, "Reference:   %s\n", tx.Reference)
	}
	if tx.Timestamp != nil {
		fmt.Fprintf(&buf, "Timestamp:   %s\n", tx.Timestamp.AsTime().Format(time.RFC3339))
	}
	if tx.Reverted {
		fmt.Fprintf(&buf, "Status:      Reverted\n")
	}

	if len(tx.Postings) > 0 {
		fmt.Fprintf(&buf, "\nPostings:\n")
		postingsTable := pterm.TableData{{"#", "SOURCE", "", "DESTINATION", "AMOUNT", "ASSET"}}
		for i, p := range tx.Postings {
			postingsTable = append(postingsTable, []string{
				fmt.Sprintf("%d", i+1),
				p.Source,
				"→",
				p.Destination,
				p.Amount.Dec(),
				p.Asset,
			})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(postingsTable).Render()
	}

	if tx.Metadata != nil && len(tx.Metadata.Metadata) > 0 {
		fmt.Fprintf(&buf, "\nMetadata:\n")
		mdTable := pterm.TableData{{"KEY", "VALUE"}}
		for _, md := range tx.Metadata.Metadata {
			mdTable = append(mdTable, []string{md.Key, commonpb.MetadataValueToString(md.Value)})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(mdTable).Render()
	}

	return successResponse("", buf.String()), nil
}

func handleTransactionsCreate(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	ledgerName := req.Flags["ledger"]
	if ledgerName == "" {
		return errorResponse("--ledger flag is required", 1), nil
	}

	postingStr := req.Flags["posting"]
	if postingStr == "" {
		return errorResponse("--posting flag is required (format: source,destination,amount,asset)", 1), nil
	}

	// Parse postings
	var postings []*commonpb.Posting
	parts := strings.Split(postingStr, ",")
	if len(parts) != 4 {
		return errorResponse("posting format: source,destination,amount,asset", 1), nil
	}

	amount, ok := new(big.Int).SetString(strings.TrimSpace(parts[2]), 10)
	if !ok {
		return errorResponse(fmt.Sprintf("invalid amount: %s", parts[2]), 1), nil
	}
	postings = append(postings, commonpb.NewPosting(
		strings.TrimSpace(parts[0]),
		strings.TrimSpace(parts[1]),
		strings.TrimSpace(parts[3]),
		amount,
	))

	client, conn, err := internal.NewClient(req.Flags)
	if err != nil {
		return errorResponse(err.Error(), 1), nil
	}
	defer conn.Close()

	ctx = internal.ContextWithAuth(ctx, req)

	force := req.Flags["force"] == "true"

	requests := []*servicepb.Request{
		{
			Type: &servicepb.Request_Apply{
				Apply: &servicepb.LedgerApplyRequest{
					Ledger: ledgerName,
					Data: &servicepb.LedgerApplyRequest_CreateTransaction{
						CreateTransaction: &servicepb.CreateTransactionPayload{
							Postings:  postings,
							Reference: req.Flags["reference"],
							Force:     force,
						},
					},
				},
			},
		},
	}

	resp, err := client.Apply(ctx, &servicepb.ApplyRequest{Requests: requests})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to create transaction: %v", err), 1), nil
	}

	if len(resp.Logs) == 0 {
		return errorResponse("no response received", 1), nil
	}

	log := resp.Logs[0]
	applyLog := log.Payload.GetApply()
	if applyLog == nil {
		return errorResponse("unexpected response type", 1), nil
	}

	createdTx := applyLog.Log.Data.GetCreatedTransaction()
	if createdTx == nil {
		return errorResponse("unexpected log payload type", 1), nil
	}

	tx := createdTx.Transaction

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(createdTx, "", "  ")
		return successResponse(string(data), ""), nil
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Transaction created: #%d\n", tx.Id)
	fmt.Fprintf(&buf, "─────────────────────────────────\n")
	if tx.Reference != "" {
		fmt.Fprintf(&buf, "Reference: %s\n", tx.Reference)
	}
	if tx.Timestamp != nil {
		fmt.Fprintf(&buf, "Timestamp: %s\n", tx.Timestamp.AsTime().Format(time.RFC3339))
	}

	if len(tx.Postings) > 0 {
		fmt.Fprintf(&buf, "\nPostings:\n")
		postingsTable := pterm.TableData{{"#", "SOURCE", "", "DESTINATION", "AMOUNT", "ASSET"}}
		for i, p := range tx.Postings {
			postingsTable = append(postingsTable, []string{
				fmt.Sprintf("%d", i+1),
				p.Source,
				"→",
				p.Destination,
				p.Amount.Dec(),
				p.Asset,
			})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(postingsTable).Render()
	}

	return successResponse("", buf.String()), nil
}

func handleTransactionsRevert(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	if len(req.Args) == 0 {
		return errorResponse("transaction ID argument is required", 1), nil
	}

	txID, err := strconv.ParseUint(req.Args[0], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("invalid transaction ID: %s", req.Args[0]), 1), nil
	}

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

	force := req.Flags["force"] == "true"

	requests := []*servicepb.Request{
		{
			Type: &servicepb.Request_Apply{
				Apply: &servicepb.LedgerApplyRequest{
					Ledger: ledgerName,
					Data: &servicepb.LedgerApplyRequest_RevertTransaction{
						RevertTransaction: &servicepb.RevertTransactionPayload{
							TransactionId: txID,
							Force:         force,
						},
					},
				},
			},
		},
	}

	resp, err := client.Apply(ctx, &servicepb.ApplyRequest{Requests: requests})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to revert transaction: %v", err), 1), nil
	}

	if len(resp.Logs) == 0 {
		return errorResponse("no response received", 1), nil
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(resp.Logs[0], "", "  ")
		return successResponse(string(data), ""), nil
	}

	return successResponse("", fmt.Sprintf("Transaction #%d reverted successfully.\n", txID)), nil
}
