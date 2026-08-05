package reconciliation

import "testing"

func TestLedgerClarityCommandSurface(t *testing.T) {
	t.Parallel()

	root := NewCommand()
	commands := [][]string{
		{"rules", "list"},
		{"rules", "get"},
		{"rules", "create"},
		{"rules", "update"},
		{"rules", "delete"},
		{"rules", "evaluate"},
		{"evaluations", "list"},
		{"evaluations", "get"},
		{"alerts", "list"},
		{"alerts", "get"},
		{"alerts", "events"},
		{"alerts", "ack"},
		{"alerts", "resolve"},
		{"alerts", "accept"},
		{"alerts", "snooze"},
		{"alerts", "unsnooze"},
	}
	for _, path := range commands {
		command, remaining, err := root.Find(path)
		if err != nil {
			t.Errorf("find %v: %v", path, err)
			continue
		}
		if len(remaining) != 0 || command.Name() != path[len(path)-1] {
			t.Errorf("find %v returned %q with remaining %v", path, command.Name(), remaining)
		}
	}
}

func TestEvaluateCommandRequiresConfirmation(t *testing.T) {
	t.Parallel()

	command, _, err := NewCommand().Find([]string{"rules", "evaluate"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Flags().Lookup("confirm") == nil {
		t.Fatal("evaluate command does not expose --confirm")
	}
}
