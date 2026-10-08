package connection

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gofrs/flock"

	"github.com/formancehq/fctl/v4/internal/cloud"
)

// Options is persisted configuration. Client secrets are read only from the environment.
type Options struct {
	StackURL     string `json:"stackURL,omitzero"`
	LedgerURL    string `json:"ledgerURL,omitzero"`
	AuthURL      string `json:"authURL,omitzero"`
	AuthMode     string `json:"authMode,omitzero"`
	TokenURL     string `json:"tokenURL,omitzero"`
	ClientID     string `json:"clientID,omitzero"`
	Scopes       string `json:"scopes,omitzero"`
	Issuer       string `json:"issuer,omitzero"`
	Organization string `json:"organization,omitzero"`
	Stack        string `json:"stack,omitzero"`
}

type Entry struct {
	Revision string         `json:"revision"`
	Options  Options        `json:"options"`
	Session  *cloud.Session `json:"session,omitzero"`
}

// Update serializes read/modify/write across CLI processes. The lock is held
// only for local persistence, never while waiting for an OAuth2 response.
func Update(ctx context.Context, directory string, change func(*Store) error) error {
	return withLock(ctx, directory, "connections.lock", func() error {
		store, err := Load(directory)
		if err != nil {
			return err
		}
		if err := change(&store); err != nil {
			return err
		}
		return Save(directory, store)
	})
}
func withLock(ctx context.Context, directory, name string, work func() error) (err error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	// Check containment before the portable lock library opens its fixed file.
	file, err := openLockFile(root, name)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	lock := flock.New(filepath.Join(directory, name))
	locked, err := lock.TryLockContext(ctx, 20*time.Millisecond)
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("connection store lock unavailable")
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	return work()
}

func openLockFile(root *os.Root, name string) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err == nil {
		return file, nil
	}
	if errors.Is(err, os.ErrExist) || errors.Is(err, os.ErrNotExist) {
		// Another creator may have won between Root's path checks and openat.
		return root.OpenFile(name, os.O_RDWR, 0o600)
	}
	return nil, err
}

func NewEntry(options Options) Entry { return Entry{Options: options, Revision: rand.Text()} }

// SaveSession refuses results from commands whose connection was changed or
// logged out while their authentication request was in flight.
func SaveSession(ctx context.Context, directory, name string, expected *string, session *cloud.Session) error {
	var revision string
	err := Update(ctx, directory, func(store *Store) error {
		entry, exists := store.Connections[name]
		if !exists || entry.Revision != *expected {
			return fmt.Errorf("connection changed during authentication; retry with its current settings")
		}
		entry.Session = session
		entry.Revision = rand.Text()
		revision = entry.Revision
		store.Connections[name] = entry
		return nil
	})
	if err == nil {
		*expected = revision
	}
	return err
}

// SaveLoginSession commits profile settings, identity and active selection only
// after successful authentication. An empty revision means the name was absent.
func SaveLoginSession(ctx context.Context, directory, name string, expected *string, options Options, session *cloud.Session) error {
	var revision string
	err := Update(ctx, directory, func(store *Store) error {
		current, exists := store.Connections[name]
		if (*expected == "" && exists) || (*expected != "" && (!exists || current.Revision != *expected)) {
			return fmt.Errorf("connection changed during login; retry with its current settings")
		}
		entry := NewEntry(options)
		entry.Session = session
		revision = entry.Revision
		store.Connections[name] = entry
		store.Active = name
		return nil
	})
	if err == nil {
		*expected = revision
	}
	return err
}

type Store struct {
	Active      string           `json:"active"`
	Connections map[string]Entry `json:"connections"`
}

func DefaultDirectory() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "formance", "fctl", "v4"), nil
}

func ValidateName(name string) error {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`).MatchString(name) {
		return fmt.Errorf("connection name must contain 1-64 letters, digits, underscores or hyphens")
	}
	return nil
}

func Load(directory string) (result Store, err error) {
	store := Store{Connections: make(map[string]Entry)}
	root, err := os.OpenRoot(directory)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("open connection directory: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	data, err := root.ReadFile("connections.json")
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("read connections: %w", err)
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return store, fmt.Errorf("decode connections: %w", err)
	}
	if store.Connections == nil {
		store.Connections = make(map[string]Entry)
	}
	return store, nil
}

// Save atomically replaces private configuration, including Cloud refresh tokens.
func Save(directory string, store Store) (err error) {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	name := ".connections-" + rand.Text()
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := root.Remove(name); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			err = errors.Join(err, cleanupErr)
		}
	}()
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return root.Rename(name, "connections.json")
}
