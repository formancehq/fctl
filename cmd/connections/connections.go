// Package connections manages named v4 connection profiles.
package connections

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func NewCommand(settings *connection.Settings) *cobra.Command {
	root := &cobra.Command{Use: "connections", Short: "Manage named service connections"}
	root.AddCommand(add(settings), use(settings), show(settings), list(settings), deleteCommand(settings))
	return root
}

func add(s *connection.Settings) *cobra.Command {
	var replace bool
	cmd := &cobra.Command{Use: "add NAME", Short: "Save connection settings (client secrets stay in FCTL_CLIENT_SECRET)", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := connection.ValidateName(args[0]); err != nil {
			return err
		}
		opts := s.Input(cmd)
		if err := connection.Validate(opts); err != nil {
			return err
		}
		dir, err := s.DirectoryPath()
		if err != nil {
			return err
		}
		if err := connection.Update(cmd.Context(), dir, func(store *connection.Store) error {
			if _, exists := store.Connections[args[0]]; exists && !replace {
				return fmt.Errorf("connection exists; use --replace to replace settings and clear its login")
			}
			store.Connections[args[0]] = connection.NewEntry(opts)
			if store.Active == "" {
				store.Active = args[0]
			}
			return nil
		}); err != nil {
			return err
		}
		return print(cmd, opts)
	}}
	cmd.Flags().BoolVar(&replace, "replace", false, "Replace existing settings and clear its Cloud session")
	return cmd
}

func use(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "use NAME", Short: "Select the default connection", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := s.DirectoryPath()
		if err != nil {
			return err
		}
		if err := connection.Update(cmd.Context(), dir, func(store *connection.Store) error {
			if _, exists := store.Connections[args[0]]; !exists {
				return fmt.Errorf("connection %q does not exist", args[0])
			}
			store.Active = args[0]
			return nil
		}); err != nil {
			return err
		}
		return print(cmd, map[string]string{"active": args[0]})
	}}
}

func show(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "show", Short: "Show resolved settings without credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		opts, _, _, _, err := s.Resolve(cmd)
		if err != nil {
			return err
		}
		return print(cmd, opts)
	}}
}

func list(s *connection.Settings) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List saved connections without credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		dir, err := s.DirectoryPath()
		if err != nil {
			return err
		}
		store, err := connection.Load(dir)
		if err != nil {
			return err
		}
		type row struct {
			Name     string             `json:"name"`
			Active   bool               `json:"active"`
			Options  connection.Options `json:"options"`
			LoggedIn bool               `json:"loggedIn"`
		}
		rows := make([]row, 0, len(store.Connections))
		for _, name := range slices.Sorted(maps.Keys(store.Connections)) {
			entry := store.Connections[name]
			rows = append(rows, row{name, name == store.Active, entry.Options, entry.Session != nil})
		}
		return print(cmd, rows)
	}}
}

func deleteCommand(s *connection.Settings) *cobra.Command {
	var confirm bool
	cmd := &cobra.Command{Use: "delete NAME", Short: "Delete a connection and its local login", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !confirm {
			return fmt.Errorf("connection deletion requires --confirm")
		}
		dir, err := s.DirectoryPath()
		if err != nil {
			return err
		}
		if err := connection.Update(cmd.Context(), dir, func(store *connection.Store) error {
			if _, exists := store.Connections[args[0]]; !exists {
				return fmt.Errorf("connection %q does not exist", args[0])
			}
			delete(store.Connections, args[0])
			if store.Active == args[0] {
				store.Active = ""
			}
			return nil
		}); err != nil {
			return err
		}
		return print(cmd, map[string]string{"deleted": args[0]})
	}}
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm deletion")
	return cmd
}

func print(cmd *cobra.Command, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return command.WriteJSON(cmd.OutOrStdout(), data)
}
