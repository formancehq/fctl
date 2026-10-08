package connection_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func TestUpdateSerializesWriters(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var workers sync.WaitGroup
	for i := range 20 {
		workers.Go(func() {
			if err := connection.Update(t.Context(), dir, func(store *connection.Store) error {
				store.Connections[fmt.Sprintf("profile-%d", i)] = connection.NewEntry(connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"})
				return nil
			}); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	store, err := connection.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Connections) != 20 {
		t.Fatalf("lost concurrent updates: %d profiles", len(store.Connections))
	}
}

func TestStaleSessionCannotUndoLogoutOrReplacement(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"logout", "replace", "delete"} {
		t.Run(operation, func(t *testing.T) {
			assertStaleSessionRejected(t, operation)
		})
	}
}

func assertStaleSessionRejected(t *testing.T, operation string) {
	t.Helper()
	dir := t.TempDir()
	entry := connection.NewEntry(connection.Options{AuthMode: "cloud"})
	entry.Session = &cloud.Session{}
	if err := connection.Save(dir, connection.Store{Connections: map[string]connection.Entry{"cloud": entry}}); err != nil {
		t.Fatal(err)
	}
	expected := entry.Revision
	if err := connection.Update(t.Context(), dir, func(store *connection.Store) error {
		if operation == "delete" {
			delete(store.Connections, "cloud")
			return nil
		}
		store.Connections["cloud"] = connection.NewEntry(entry.Options)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := connection.SaveSession(t.Context(), dir, "cloud", &expected, &cloud.Session{}); err == nil {
		t.Fatal("accepted stale authentication result")
	}
	store, err := connection.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if store.Connections["cloud"].Session != nil {
		t.Fatal("resurrected logged-out session")
	}
}
