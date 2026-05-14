// Package credentials abstracts secure and explicit-insecure secret storage for
// authentication providers.
package credentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrNotFound = errors.New("credential not found")

type Store interface {
	Get(ctx context.Context, ref string) (string, error)
	Set(ctx context.Context, ref string, value string) error
	Delete(ctx context.Context, ref string) error
}

type MemoryStore struct {
	mu      sync.RWMutex
	secrets map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{secrets: map[string]string{}}
}

func (s *MemoryStore) Get(_ context.Context, ref string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.secrets[ref]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	return value, nil
}

func (s *MemoryStore) Set(_ context.Context, ref string, value string) error {
	if ref == "" {
		return errors.New("credential ref cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[ref] = value
	return nil
}

func (s *MemoryStore) Delete(_ context.Context, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, ref)
	return nil
}

// InsecureFileStore stores secrets in plain text files. It is intended for
// development and tests only, and must be selected explicitly by callers.
type InsecureFileStore struct {
	dir string
}

func NewInsecureFileStore(dir string) *InsecureFileStore {
	return &InsecureFileStore{dir: dir}
}

func (s *InsecureFileStore) Get(_ context.Context, ref string) (string, error) {
	path, err := s.path(ref)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrNotFound, ref)
		}
		return "", fmt.Errorf("read credential: %w", err)
	}
	return string(data), nil
}

func (s *InsecureFileStore) Set(_ context.Context, ref string, value string) error {
	path, err := s.path(ref)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		return fmt.Errorf("write credential: %w", err)
	}
	return nil
}

func (s *InsecureFileStore) Delete(_ context.Context, ref string) error {
	path, err := s.path(ref)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete credential: %w", err)
	}
	return nil
}

func (s *InsecureFileStore) path(ref string) (string, error) {
	if s.dir == "" {
		return "", errors.New("credential directory cannot be empty")
	}
	if ref == "" {
		return "", errors.New("credential ref cannot be empty")
	}
	clean := filepath.Clean(ref)
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || hasPathTraversal(clean) {
		return "", fmt.Errorf("invalid credential ref %q", ref)
	}
	return filepath.Join(s.dir, clean), nil
}

func hasPathTraversal(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
