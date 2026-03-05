package main

import (
	"context"
	"fmt"

	"github.com/formancehq/fctl/pkg/pluginsdk/pluginpb"
)

var version = "dev"

// LedgerPlugin implements the FctlPlugin interface for Ledger v3.
type LedgerPlugin struct{}

func (p *LedgerPlugin) GetManifest(ctx context.Context) (*pluginpb.PluginManifest, error) {
	return &pluginpb.PluginManifest{
		Name:        "ledger",
		Version:     version,
		Description: "Formance Ledger v3 commands (gRPC)",
		RootCommand: buildManifest(),
	}, nil
}

func (p *LedgerPlugin) Execute(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error) {
	handler, ok := handlers[req.CommandPath]
	if !ok {
		return errorResponse(fmt.Sprintf("unknown command: %s", req.CommandPath), 1), nil
	}
	return handler(ctx, req)
}

// Handler is a function that handles a specific command.
type Handler func(ctx context.Context, req *pluginpb.ExecuteRequest) (*pluginpb.ExecuteResponse, error)

// handlers maps command paths to their handler functions.
var handlers = map[string]Handler{
	// Ledger management
	"list":   handleLedgersList,
	"create": handleLedgersCreate,
	"get":    handleLedgersGet,
	"delete": handleLedgersDelete,
	"stats":  handleLedgersStats,

	// Transactions
	"transactions/list":   handleTransactionsList,
	"transactions/get":    handleTransactionsGet,
	"transactions/create": handleTransactionsCreate,
	"transactions/revert": handleTransactionsRevert,

	// Accounts
	"accounts/list": handleAccountsList,
	"accounts/get":  handleAccountsGet,

	// Logs
	"logs/list": handleLogsList,
}

func successResponse(jsonData, renderedText string) *pluginpb.ExecuteResponse {
	return &pluginpb.ExecuteResponse{
		Result: &pluginpb.ExecuteResponse_Success{
			Success: &pluginpb.ExecuteSuccess{
				JsonData:     jsonData,
				RenderedText: renderedText,
			},
		},
	}
}

func errorResponse(message string, code int32) *pluginpb.ExecuteResponse {
	return &pluginpb.ExecuteResponse{
		Result: &pluginpb.ExecuteResponse_Error{
			Error: &pluginpb.ExecuteError{
				Message: message,
				Code:    code,
			},
		},
	}
}
