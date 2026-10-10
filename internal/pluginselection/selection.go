// Package pluginselection records the prepared provider for an exact service
// target. It is independent of provider binaries and the modern OCI cache.
package pluginselection

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxSelectionBytes = 16 << 10

var (
	// ErrNotSelected means no provider was prepared for this exact target/service.
	ErrNotSelected  = errors.New("plugin provider is not selected for this target")
	identityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,255}$`)
	servicePattern  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	versionPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,255}$`)
	filenamePattern = regexp.MustCompile(`^[a-f0-9]{64}\.json$`)
)

// Target contains stable, secret-free identity. Profile, Organization and Stack
// may be empty for direct connections; no field is treated as a wildcard.
type Target struct {
	Profile      string `json:"profile"`
	Organization string `json:"organization,omitzero"`
	Stack        string `json:"stack,omitzero"`
	Endpoint     string `json:"endpoint"`
}

// Selection records exact versions, without credentials or executable paths.
type Selection struct {
	Target         Target `json:"target"`
	Service        string `json:"service"`
	ServiceVersion string `json:"serviceVersion"`
	StackVersion   string `json:"stackVersion"`
	Provider       string `json:"provider"`
	PluginVersion  string `json:"pluginVersion"`
}

// Load reads only the selection for target and service. An empty endpoint before
// login, or missing directories/files, returns ErrNotSelected. Corrupt or unsafe
// records return another error.
func Load(directory string, target Target, service string) (selection Selection, err error) {
	if target.Endpoint == "" {
		return Selection{}, ErrNotSelected
	}
	if err := validateKey(target, service); err != nil {
		return Selection{}, err
	}
	root, err := openSelections(directory, false)
	if errors.Is(err, fs.ErrNotExist) {
		return Selection{}, ErrNotSelected
	}
	if err != nil {
		return Selection{}, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	selection, err = readSelection(root, filename(target, service))
	if errors.Is(err, fs.ErrNotExist) {
		return Selection{}, ErrNotSelected
	}
	if err != nil {
		return Selection{}, err
	}
	if selection.Target != target || selection.Service != service {
		return Selection{}, errors.New("plugin selection identity does not match requested target and service")
	}
	return selection, nil
}

// Save atomically replaces one target/service selection. Versions and provider
// are values, not keys, so switching a provider cannot retain a stale selection.
// ServiceVersion is required. StackVersion is optional for standalone targets;
// PluginVersion is optional for modern providers, whose revision is owned by
// the OCI cache.
func Save(directory string, selection Selection) (err error) {
	if err := validateSelection(selection); err != nil {
		return err
	}
	data, err := json.Marshal(selection)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > maxSelectionBytes {
		return errors.New("plugin selection exceeds size limit")
	}
	root, err := openSelections(directory, true)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	name := filename(selection.Target, selection.Service)
	if err := regularFile(root, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return atomicWrite(root, name, data)
}

// List returns validated selections in filename order. Missing caches are empty;
// any invalid record fails the list rather than exposing partial routing data.
func List(directory string) (selections []Selection, err error) {
	root, err := openSelections(directory, false)
	if errors.Is(err, fs.ErrNotExist) {
		return []Selection{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, err
	}
	selections = make([]Selection, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue // Includes temporary files from interrupted writers.
		}
		if !filenamePattern.MatchString(name) {
			return nil, errors.New("invalid plugin selection filename")
		}
		selection, err := readSelection(root, name)
		if err != nil {
			return nil, err
		}
		if name != filename(selection.Target, selection.Service) {
			return nil, errors.New("plugin selection identity does not match filename")
		}
		selections = append(selections, selection)
	}
	return selections, nil
}

func validateKey(target Target, service string) error {
	for _, identity := range []string{target.Profile, target.Organization, target.Stack} {
		if identity != "" && !identityPattern.MatchString(identity) {
			return errors.New("invalid plugin selection target identity")
		}
	}
	if !servicePattern.MatchString(service) {
		return errors.New("invalid plugin selection service")
	}
	endpoint := target.Endpoint
	if len(endpoint) > 4096 || !utf8.ValidString(endpoint) || strings.ContainsAny(endpoint, "?#") ||
		strings.ContainsFunc(endpoint, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return errors.New("plugin selection endpoint must be a secret-free absolute HTTP(S) URL")
	}
	u, err := url.Parse(endpoint)
	if err != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || u.Opaque != "" ||
		(u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("plugin selection endpoint must be a secret-free absolute HTTP(S) URL")
	}
	return nil
}

func validateSelection(selection Selection) error {
	if err := validateKey(selection.Target, selection.Service); err != nil {
		return err
	}
	if selection.Provider != "legacy" && selection.Provider != "modern" {
		return errors.New("plugin selection provider must be legacy or modern")
	}
	if !versionPattern.MatchString(selection.ServiceVersion) {
		return errors.New("plugin selection service version must be a nonempty version identifier")
	}
	if selection.StackVersion != "" && !versionPattern.MatchString(selection.StackVersion) {
		return errors.New("invalid plugin selection stack version")
	}
	if (selection.Provider == "legacy" || selection.PluginVersion != "") && !versionPattern.MatchString(selection.PluginVersion) {
		return errors.New("invalid plugin selection plugin version")
	}
	return nil
}

func filename(target Target, service string) string {
	// A fixed JSON tuple prevents separator collisions and includes every field.
	data, _ := json.Marshal([5]string{target.Profile, target.Organization, target.Stack, target.Endpoint, service}) //nolint:errcheck // A string array cannot fail JSON marshaling.
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) + ".json"
}

func openSelections(directory string, create bool) (*os.Root, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("plugin selection directory is required")
	}
	if create {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, err
		}
	}
	base, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	root, err := openSelectionDirectory(base, create)
	if closeErr := base.Close(); closeErr != nil {
		if root != nil {
			closeErr = errors.Join(closeErr, root.Close())
		}
		return nil, errors.Join(err, closeErr)
	}
	return root, err
}

func openSelectionDirectory(base *os.Root, create bool) (*os.Root, error) {
	if create {
		if err := privateDirectory(base); err != nil {
			return nil, err
		}
		if err := base.Mkdir("selections", 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
	}
	info, err := base.Lstat("selections")
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("plugin selections must be a directory, not a symlink")
	}
	root, err := base.OpenRoot("selections")
	if err != nil {
		return nil, err
	}
	if create {
		if err := privateDirectory(root); err != nil {
			return nil, errors.Join(err, root.Close())
		}
	}
	return root, nil
}

func privateDirectory(root *os.Root) error {
	file, err := root.Open(".")
	if err != nil {
		return err
	}
	// Chmod the open descriptor, avoiding Root.Chmod's Unix symlink race.
	return errors.Join(file.Chmod(0o700), file.Close())
}

func regularFile(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("plugin selection must be a regular file, not a symlink")
	}
	return nil
}

func readSelection(root *os.Root, name string) (_ Selection, err error) {
	if err := regularFile(root, name); err != nil {
		return Selection{}, err
	}
	file, err := root.Open(name)
	if err != nil {
		return Selection{}, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return Selection{}, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > maxSelectionBytes {
		return Selection{}, errors.New("plugin selection must be a private, bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSelectionBytes+1))
	if err != nil {
		return Selection{}, err
	}
	if len(data) > maxSelectionBytes || !utf8.Valid(data) {
		return Selection{}, errors.New("plugin selection exceeds size limit or contains invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var selection Selection
	if err := decoder.Decode(&selection); err != nil {
		return Selection{}, errors.New("invalid plugin selection JSON")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Selection{}, errors.New("invalid trailing plugin selection JSON")
	}
	// encoding/json accepts duplicate keys and null string fields by default.
	// Neither is an unambiguous provider selection.
	if err := strictJSONValue(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return Selection{}, err
	}
	if err := validateSelection(selection); err != nil {
		return Selection{}, err
	}
	return selection, nil
}

func strictJSONValue(decoder *json.Decoder) error {
	value, err := decoder.Token()
	if err != nil {
		return errors.New("invalid plugin selection JSON")
	}
	if _, ok := value.(string); ok {
		return nil
	}
	if value != json.Delim('{') {
		return errors.New("plugin selection JSON must contain only objects and strings")
	}
	return strictJSONObject(decoder)
}

func strictJSONObject(decoder *json.Decoder) error {
	seen := make(map[string]bool)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return errors.New("invalid plugin selection JSON")
		}
		name, ok := key.(string)
		// Match encoding/json's case-insensitive handling of struct field names.
		name = strings.ToLower(name)
		if !ok || seen[name] {
			return errors.New("ambiguous plugin selection JSON field")
		}
		seen[name] = true
		if err := strictJSONValue(decoder); err != nil {
			return err
		}
	}
	if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
		return errors.New("invalid plugin selection JSON")
	}
	return nil
}

func atomicWrite(root *os.Root, name string, data []byte) (err error) {
	temporary := ".tmp-" + rand.Text()
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cleanup := root.Remove(temporary); cleanup != nil && !errors.Is(cleanup, fs.ErrNotExist) {
			err = errors.Join(err, cleanup)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return errors.Join(err, file.Close())
	}
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	return root.Rename(temporary, name)
}
