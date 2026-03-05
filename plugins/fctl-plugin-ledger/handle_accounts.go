package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/formancehq/fctl-plugin-ledger/internal"
	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/commonpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/servicepb"
	"github.com/pterm/pterm"
)

func handleAccountsList(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
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

	stream, err := client.ListAccounts(ctx, &servicepb.ListAccountsRequest{
		Ledger:   ledgerName,
		PageSize: pageSize,
		Reverse:  reverse,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to list accounts: %v", err), 1), nil
	}

	var accounts []*commonpb.Account
	for {
		account, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errorResponse(fmt.Sprintf("failed to receive account: %v", err), 1), nil
		}
		accounts = append(accounts, account)
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(accounts, "", "  ")
		return successResponse(string(data), ""), nil
	}

	if len(accounts) == 0 {
		return successResponse("", "No accounts found.\n"), nil
	}

	var buf bytes.Buffer
	tableData := pterm.TableData{{"ADDRESS", "METADATA"}}
	for _, account := range accounts {
		metadataCount := "0"
		if account.Metadata != nil {
			metadataCount = fmt.Sprintf("%d", len(account.Metadata.Metadata))
		}
		tableData = append(tableData, []string{account.Address, metadataCount})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(tableData).Render()
	return successResponse("", buf.String()), nil
}

func handleAccountsGet(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	if len(req.Args) == 0 {
		return errorResponse("account address argument is required", 1), nil
	}
	address := req.Args[0]

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

	account, err := client.GetAccount(ctx, &servicepb.GetAccountRequest{
		Ledger:  ledgerName,
		Address: address,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to get account: %v", err), 1), nil
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(account, "", "  ")
		return successResponse(string(data), ""), nil
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Account: %s\n", account.Address)
	fmt.Fprintf(&buf, "─────────────────────────────────\n")

	if account.Metadata != nil && len(account.Metadata.Metadata) > 0 {
		fmt.Fprintf(&buf, "\nMetadata:\n")
		mdTable := pterm.TableData{{"KEY", "VALUE"}}
		for _, md := range account.Metadata.Metadata {
			mdTable = append(mdTable, []string{md.Key, commonpb.MetadataValueToString(md.Value)})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(mdTable).Render()
	}

	fmt.Fprintf(&buf, "\nVolumes:\n")
	if len(account.Volumes) > 0 {
		volTable := pterm.TableData{{"ASSET", "INPUT", "OUTPUT", "BALANCE"}}
		assets := make([]string, 0, len(account.Volumes))
		for asset := range account.Volumes {
			assets = append(assets, asset)
		}
		sort.Strings(assets)

		for _, asset := range assets {
			vol := account.Volumes[asset]
			volTable = append(volTable, []string{asset, vol.Input, vol.Output, vol.Balance})
		}
		_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(volTable).Render()
	} else {
		fmt.Fprintf(&buf, "(no volumes)\n")
	}

	return successResponse("", buf.String()), nil
}
