package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/formancehq/fctl-plugin-ledger/internal"
	"github.com/formancehq/fctl-plugin-ledger/proto/commonpb"
	"github.com/formancehq/fctl-plugin-ledger/proto/servicepb"
	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
	"github.com/pterm/pterm"
)

func handleLogsList(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
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
	if fetchAll {
		pageSize = 0
	}

	stream, err := client.ListLogs(ctx, &servicepb.ListLogsRequest{
		PageSize: pageSize,
	})
	if err != nil {
		return errorResponse(fmt.Sprintf("failed to list logs: %v", err), 1), nil
	}

	var logs []*commonpb.Log
	for {
		log, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errorResponse(fmt.Sprintf("failed to receive log: %v", err), 1), nil
		}
		logs = append(logs, log)
	}

	if req.OutputFormat == "json" {
		data, _ := json.MarshalIndent(logs, "", "  ")
		return successResponse(string(data), ""), nil
	}

	if len(logs) == 0 {
		return successResponse("", "No logs found.\n"), nil
	}

	var buf bytes.Buffer
	tableData := pterm.TableData{{"SEQ", "TYPE"}}
	for _, log := range logs {
		logType := describeLogType(log)
		tableData = append(tableData, []string{
			fmt.Sprintf("%d", log.Sequence),
			logType,
		})
	}

	_ = pterm.DefaultTable.WithHasHeader().WithWriter(&buf).WithData(tableData).Render()
	return successResponse("", buf.String()), nil
}

func describeLogType(log *commonpb.Log) string {
	if log.Payload == nil {
		return "unknown"
	}
	applyLog := log.Payload.GetApply()
	if applyLog != nil && applyLog.Log != nil && applyLog.Log.Data != nil {
		switch applyLog.Log.Data.Payload.(type) {
		case *commonpb.LedgerLogPayload_CreatedTransaction:
			return "CreatedTransaction"
		case *commonpb.LedgerLogPayload_RevertedTransaction:
			return "RevertedTransaction"
		case *commonpb.LedgerLogPayload_SavedMetadata:
			return "SetMetadata"
		case *commonpb.LedgerLogPayload_DeletedMetadata:
			return "DeletedMetadata"
		}
		return "LedgerLog"
	}
	if log.Payload.GetCreateLedger() != nil {
		return "CreateLedger"
	}
	if log.Payload.GetDeleteLedger() != nil {
		return "DeleteLedger"
	}
	return "other"
}
