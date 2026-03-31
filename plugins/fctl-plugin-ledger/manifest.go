package main

import (
	"github.com/formancehq/fctl/v3/pkg/pluginsdk/pluginpb"
)

func buildManifest() *pluginpb.CommandSpec {
	return &pluginpb.CommandSpec{
		Use:         "ledger",
		Aliases:     []string{"lg"},
		Short:       "Ledger v3 management",
		CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
		PersistentFlags: []*pluginpb.FlagSpec{
			stringFlag("server", "localhost:8888", "gRPC server address"),
			boolFlag("insecure", "true", "Use insecure connection (no TLS)"),
			stringFlag("tls-ca-cert", "", "Path to CA certificate file (PEM)"),
			stringFlag("auth-token", "", "Bearer token for authentication"),
		},
		Subcommands: []*pluginpb.CommandSpec{
			// Ledger CRUD
			{
				Use:         "list",
				Aliases:     []string{"ls", "l"},
				Short:       "List all ledgers",
				Runnable:    true,
				CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
				Flags: []*pluginpb.FlagSpec{
					boolFlag("json", "false", "Output as JSON"),
				},
			},
			{
				Use:            "create",
				Aliases:        []string{"new", "add"},
				Short:          "Create a new ledger",
				Runnable:       true,
				CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
				ArgsConstraint: "none",
				Flags: []*pluginpb.FlagSpec{
					stringFlag("name", "", "Name of the ledger to create"),
					boolFlag("json", "false", "Output as JSON"),
				},
			},
			{
				Use:            "get",
				Aliases:        []string{"show", "describe"},
				Short:          "Get a ledger by name",
				Runnable:       true,
				CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
				ArgsConstraint: "exact:1",
				Flags: []*pluginpb.FlagSpec{
					boolFlag("json", "false", "Output as JSON"),
				},
			},
			{
				Use:            "delete",
				Aliases:        []string{"rm"},
				Short:          "Delete a ledger",
				Runnable:       true,
				CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
				ArgsConstraint: "exact:1",
				Confirm:        true,
			},
			{
				Use:         "stats",
				Short:       "Get ledger statistics",
				Runnable:    true,
				CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
				Flags: []*pluginpb.FlagSpec{
					stringFlag("ledger", "", "Name of the ledger"),
					boolFlag("json", "false", "Output as JSON"),
				},
			},

			// Transactions
			{
				Use:         "transactions",
				Aliases:     []string{"tx", "t"},
				Short:       "Manage transactions",
				CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
				Subcommands: []*pluginpb.CommandSpec{
					{
						Use:         "list",
						Aliases:     []string{"ls", "l"},
						Short:       "List transactions in a ledger",
						Runnable:    true,
						CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							intFlag("page-size", "10", "Number of transactions per page"),
							stringFlag("filter", "", "Filter expression"),
							boolFlag("reverse", "false", "Reverse iteration order"),
							boolFlag("all", "false", "Fetch all transactions at once"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
					{
						Use:            "get",
						Aliases:        []string{"show"},
						Short:          "Get a transaction by ID",
						Runnable:       true,
						CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
						ArgsConstraint: "exact:1",
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
					{
						Use:            "create",
						Aliases:        []string{"new", "add"},
						Short:          "Create a new transaction",
						Runnable:       true,
						CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
						ArgsConstraint: "none",
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							stringFlag("posting", "", "Posting: source,destination,amount,asset"),
							stringFlag("reference", "", "Transaction reference"),
							boolFlag("force", "false", "Bypass balance checks"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
					{
						Use:            "revert",
						Short:          "Revert a transaction",
						Runnable:       true,
						CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
						ArgsConstraint: "exact:1",
						Confirm:        true,
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							boolFlag("force", "false", "Force revert even if already reverted"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
				},
			},

			// Accounts
			{
				Use:         "accounts",
				Aliases:     []string{"acc", "a"},
				Short:       "Manage accounts",
				CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
				Subcommands: []*pluginpb.CommandSpec{
					{
						Use:         "list",
						Aliases:     []string{"ls", "l"},
						Short:       "List accounts in a ledger",
						Runnable:    true,
						CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							intFlag("page-size", "10", "Number of accounts per page"),
							stringFlag("prefix", "", "Filter by address prefix"),
							stringFlag("filter", "", "Filter expression"),
							boolFlag("reverse", "false", "Reverse order"),
							boolFlag("all", "false", "Fetch all accounts at once"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
					{
						Use:            "get",
						Aliases:        []string{"show"},
						Short:          "Get an account by address",
						Runnable:       true,
						CommandType:    pluginpb.CommandType_COMMAND_TYPE_BASIC,
						ArgsConstraint: "exact:1",
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
				},
			},

			// Logs
			{
				Use:         "logs",
				Short:       "View logs",
				CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
				Subcommands: []*pluginpb.CommandSpec{
					{
						Use:         "list",
						Aliases:     []string{"ls", "l"},
						Short:       "List logs",
						Runnable:    true,
						CommandType: pluginpb.CommandType_COMMAND_TYPE_BASIC,
						Flags: []*pluginpb.FlagSpec{
							stringFlag("ledger", "", "Name of the ledger"),
							intFlag("page-size", "10", "Number of logs per page"),
							boolFlag("all", "false", "Fetch all logs at once"),
							boolFlag("json", "false", "Output as JSON"),
						},
					},
				},
			},
		},
	}
}

func stringFlag(name, defaultValue, description string) *pluginpb.FlagSpec {
	return &pluginpb.FlagSpec{
		Name:         name,
		DefaultValue: defaultValue,
		Description:  description,
		Type:         pluginpb.FlagType_FLAG_TYPE_STRING,
	}
}

func boolFlag(name, defaultValue, description string) *pluginpb.FlagSpec {
	return &pluginpb.FlagSpec{
		Name:         name,
		DefaultValue: defaultValue,
		Description:  description,
		Type:         pluginpb.FlagType_FLAG_TYPE_BOOL,
	}
}

func intFlag(name, defaultValue, description string) *pluginpb.FlagSpec {
	return &pluginpb.FlagSpec{
		Name:         name,
		DefaultValue: defaultValue,
		Description:  description,
		Type:         pluginpb.FlagType_FLAG_TYPE_INT,
	}
}
