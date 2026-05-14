package credentials

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	if err := store.Set(ctx, "keyring://local/testing", "secret"); err != nil {
		t.Fatalf("set credential: %v", err)
	}

	value, err := store.Get(ctx, "keyring://local/testing")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if value != "secret" {
		t.Fatalf("expected secret, got %q", value)
	}

	if err := store.Delete(ctx, "keyring://local/testing"); err != nil {
		t.Fatalf("delete credential: %v", err)
	}
	if _, err := store.Get(ctx, "keyring://local/testing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestInsecureFileStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := NewInsecureFileStore(dir)

	if err := store.Set(ctx, "local/testing", "secret"); err != nil {
		t.Fatalf("set credential: %v", err)
	}

	path := filepath.Join(dir, "local", "testing")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credential: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected credential mode 0600, got %o", mode)
	}

	value, err := store.Get(ctx, "local/testing")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if value != "secret" {
		t.Fatalf("expected secret, got %q", value)
	}

	if err := store.Delete(ctx, "local/testing"); err != nil {
		t.Fatalf("delete credential: %v", err)
	}
	if _, err := store.Get(ctx, "local/testing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestInsecureFileStoreRejectsTraversal(t *testing.T) {
	store := NewInsecureFileStore(t.TempDir())
	if err := store.Set(context.Background(), "../secret", "secret"); err == nil {
		t.Fatal("expected traversal error")
	}
}
