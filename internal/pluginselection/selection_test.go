package pluginselection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func fixture() Selection {
	return Selection{
		Target:  Target{Profile: "sandbox", Organization: "org-1", Stack: "stack-1", Endpoint: "https://stack.example.invalid/api"},
		Service: "ledger", ServiceVersion: "2.2.0", StackVersion: "3.2.0",
		Provider: "legacy", PluginVersion: "1.0.0",
	}
}

func mustSave(t *testing.T, directory string, selection Selection) {
	t.Helper()
	if err := Save(directory, selection); err != nil {
		t.Fatal(err)
	}
}

func assertLoad(t *testing.T, directory string, want Selection) {
	t.Helper()
	got, err := Load(directory, want.Target, want.Service)
	if err != nil || got != want {
		t.Fatalf("Load = %+v, %v; want %+v", got, err, want)
	}
}

func writeRecord(t *testing.T, directory string, selection Selection, data []byte) string {
	t.Helper()
	path := filepath.Join(directory, "selections", filename(selection.Target, selection.Service))
	writeFile(t, path, data)
	return path
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // Paths belong to test-owned temporary directories.
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func makeDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func removePath(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected absent path %s: %v", path, err)
	}
}

func prepareEmptyCache(t *testing.T, directory, state string) {
	t.Helper()
	if state != "missing base" {
		makeDirectory(t, directory)
	}
	if state == "empty selections" {
		makeDirectory(t, filepath.Join(directory, "selections"))
	}
}

func TestMissingSelectionReadsDoNotCreateCache(t *testing.T) {
	for _, state := range []string{"missing base", "missing selections", "empty selections"} {
		t.Run(state, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "plugins")
			prepareEmptyCache(t, directory, state)
			selection := fixture()
			got, err := Load(directory, selection.Target, selection.Service)
			if got != (Selection{}) || !errors.Is(err, ErrNotSelected) {
				t.Fatalf("Load = %+v, %v; want ErrNotSelected", got, err)
			}
			listed, err := List(directory)
			if err != nil || listed == nil || len(listed) != 0 {
				t.Fatalf("List = %+v, %v; want empty list", listed, err)
			}
			if state == "missing base" {
				assertAbsent(t, directory)
			}
			if state == "missing selections" {
				assertAbsent(t, filepath.Join(directory, "selections"))
			}
		})
	}
}

func TestLoadBeforeLoginIsNotSelected(t *testing.T) {
	for _, directory := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		got, err := Load(directory, Target{}, "ledger")
		if got != (Selection{}) || !errors.Is(err, ErrNotSelected) {
			t.Fatalf("Load without target = %+v, %v; want ErrNotSelected", got, err)
		}
	}
	selection := fixture()
	selection.Target.Endpoint = ""
	if err := Save(t.TempDir(), selection); err == nil {
		t.Fatal("Save accepted an empty endpoint")
	}
}

func TestRoundTripAndVersionSwitches(t *testing.T) {
	directory := t.TempDir()
	initial := fixture()
	mustSave(t, directory, initial)
	assertLoad(t, directory, initial)
	modern := initial
	modern.Provider = "modern"
	modern.ServiceVersion = "3.0.0-beta.10"
	modern.StackVersion = "4.0.0"
	modern.PluginVersion = "" // The OCI cache owns the modern plugin revision.
	mustSave(t, directory, modern)
	assertLoad(t, directory, modern)
	legacy := initial
	legacy.ServiceVersion = "2.3.0+build.1"
	legacy.StackVersion = "3.3.0"
	legacy.PluginVersion = "1.1.0"
	mustSave(t, directory, legacy)
	mustSave(t, directory, legacy)
	assertLoad(t, directory, legacy)
	listed, err := List(directory)
	if err != nil || len(listed) != 1 || listed[0] != legacy {
		t.Fatalf("switch retained stale records: %+v, %v", listed, err)
	}
	entries, err := os.ReadDir(filepath.Join(directory, "selections"))
	if err != nil || len(entries) != 1 || !filenamePattern.MatchString(entries[0].Name()) {
		t.Fatalf("expected one deterministic hashed file: %+v, %v", entries, err)
	}
}

func TestStandaloneTargetsHaveNoStackVersion(t *testing.T) {
	for _, provider := range []string{"legacy", "modern"} {
		t.Run(provider, func(t *testing.T) {
			directory := t.TempDir()
			selection := fixture()
			selection.Target = Target{Endpoint: "http://localhost:8080/api/ledger"}
			selection.StackVersion = ""
			selection.Provider = provider
			if provider == "modern" {
				selection.PluginVersion = ""
			}
			mustSave(t, directory, selection)
			assertLoad(t, directory, selection)
			listed, err := List(directory)
			if err != nil || len(listed) != 1 || listed[0] != selection {
				t.Fatalf("standalone selection = %+v, %v", listed, err)
			}
		})
	}
}

func TestOptionalStringFieldsCannotBeNull(t *testing.T) {
	selection := fixture()
	selection.Target.Profile = ""
	selection.StackVersion = ""
	selection.Provider, selection.PluginVersion = "modern", ""
	valid, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"profile", "stackVersion", "pluginVersion"} {
		t.Run(field, func(t *testing.T) {
			directory := t.TempDir()
			mustSave(t, directory, selection)
			data := bytes.Replace(valid, []byte(`"`+field+`":""`), []byte(`"`+field+`":null`), 1)
			writeRecord(t, directory, selection, data)
			if _, err := Load(directory, selection.Target, selection.Service); err == nil {
				t.Fatal("Load accepted a null string field")
			}
			if _, err := List(directory); err == nil {
				t.Fatal("List accepted a null string field")
			}
		})
	}
}

func TestExactTargetAndServiceIsolation(t *testing.T) {
	directory := t.TempDir()
	initial := fixture()
	mustSave(t, directory, initial)
	changes := map[string]func(*Selection){
		"profile":      func(s *Selection) { s.Target.Profile = "other" },
		"organization": func(s *Selection) { s.Target.Organization = "other" },
		"stack":        func(s *Selection) { s.Target.Stack = "other" },
		"endpoint":     func(s *Selection) { s.Target.Endpoint = "https://other.example.invalid/api" },
		"URL spelling": func(s *Selection) { s.Target.Endpoint += "/" },
		"service":      func(s *Selection) { s.Service = "auth" },
		"direct":       func(s *Selection) { s.Target.Profile, s.Target.Organization, s.Target.Stack = "", "", "" },
		"incomplete":   func(s *Selection) { s.Target.Organization, s.Target.Stack = "", "" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			other := initial
			change(&other)
			if _, err := Load(directory, other.Target, other.Service); !errors.Is(err, ErrNotSelected) {
				t.Fatalf("unprepared target found another selection: %v", err)
			}
			mustSave(t, directory, other)
			assertLoad(t, directory, other)
			assertLoad(t, directory, initial)
		})
	}
	listed, err := List(directory)
	if err != nil || len(listed) != len(changes)+1 {
		t.Fatalf("List = %d records, %v", len(listed), err)
	}
	again, err := List(directory)
	if err != nil {
		t.Fatal(err)
	}
	for i := range listed {
		if listed[i] != again[i] {
			t.Fatal("list order is nondeterministic")
		}
	}
}

func TestPrivateFilesAndIndependentCache(t *testing.T) {
	directory := t.TempDir()
	selections := filepath.Join(directory, "selections")
	if err := os.Mkdir(selections, 0o755); err != nil { //nolint:gosec // Verify Save repairs pre-existing public directory permissions.
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o755); err != nil { //nolint:gosec // Verify Save repairs pre-existing public directory permissions.
		t.Fatal(err)
	}
	otherCache := filepath.Join(directory, "locks")
	makeDirectory(t, otherCache)
	modernLock := filepath.Join(otherCache, "modern.json")
	lockData := []byte(`{"access_token":"synthetic-secret","provider":"modern"}`)
	writeFile(t, modernLock, lockData)
	selection := fixture()
	for range 2 {
		mustSave(t, directory, selection)
	}
	for path, want := range map[string]os.FileMode{
		directory: 0o700, selections: 0o700,
		filepath.Join(selections, filename(selection.Target, selection.Service)): 0o600,
	} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != want {
			t.Fatalf("unexpected private permissions for %s: %v, %v", path, info, err)
		}
	}
	gotLock := readFile(t, modernLock)
	if !bytes.Equal(gotLock, lockData) {
		t.Fatalf("selection changed OCI cache: %s", gotLock)
	}
	listed, err := List(directory)
	if err != nil || len(listed) != 1 || listed[0] != selection {
		t.Fatalf("selection list used OCI cache: %+v, %v", listed, err)
	}
	data := readFile(t, filepath.Join(selections, filename(selection.Target, selection.Service)))
	for _, forbidden := range []string{"synthetic-secret", "access_token", "Authorization", "executable", "catalogue"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("selection contains unrelated/private material %q", forbidden)
		}
	}
}

func TestValidationRejectsUnsafeIdentitiesAndVersions(t *testing.T) {
	changes := map[string]func(*Selection){
		"profile traversal": func(s *Selection) { s.Target.Profile = "../outside" },
		"absolute org":      func(s *Selection) { s.Target.Organization = "/outside" },
		"backslash stack":   func(s *Selection) { s.Target.Stack = `..\outside` },
		"dot identity":      func(s *Selection) { s.Target.Stack = ".." },
		"identity control":  func(s *Selection) { s.Target.Profile = "sandbox\x00" },
		"identity newline":  func(s *Selection) { s.Target.Organization = "org\n" },
		"identity space":    func(s *Selection) { s.Target.Stack = "stack other" },
		"identity length":   func(s *Selection) { s.Target.Profile = strings.Repeat("x", 257) },
		"empty service":     func(s *Selection) { s.Service = "" },
		"service traversal": func(s *Selection) { s.Service = "../../outside" },
		"invalid provider":  func(s *Selection) { s.Provider = "automatic" },
		"empty version":     func(s *Selection) { s.ServiceVersion = "" },
		"legacy revision":   func(s *Selection) { s.PluginVersion = "" },
		"version traversal": func(s *Selection) { s.PluginVersion = "../outside" },
		"version newline":   func(s *Selection) { s.StackVersion = "3.2.0\n" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "plugins")
			selection := fixture()
			change(&selection)
			if err := Save(directory, selection); err == nil {
				t.Fatal("invalid selection was saved")
			}
			if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("validation failure created cache: %v", err)
			}
		})
	}
}

func TestEndpointValidationAndSecretFreeErrors(t *testing.T) {
	for _, endpoint := range []string{
		"", "/api", "//stack.example.invalid", "https:stack.example.invalid", "file:///tmp/cache",
		"https://", "https://:443/api", "https://user:synthetic-secret@stack.example.invalid",
		"https://synthetic-secret@stack.example.invalid", "https://stack.example.invalid?token=synthetic-secret",
		"https://stack.example.invalid?", "https://stack.example.invalid#synthetic-secret",
		"https://stack.example.invalid#", "https://stack.example.invalid/%synthetic-secret",
		"https://stack.example.invalid/ synthetic-secret", "https://stack.example.invalid/\nsynthetic-secret",
		"https://stack.example.invalid/\xffsynthetic-secret",
	} {
		t.Run(fmt.Sprintf("URL %x", []byte(endpoint)), func(t *testing.T) {
			selection := fixture()
			selection.Target.Endpoint = endpoint
			err := Save(t.TempDir(), selection)
			if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatalf("unsafe endpoint was saved or exposed: %v", err)
			}
			if endpoint != "" {
				if _, err := Load(t.TempDir(), selection.Target, selection.Service); err == nil || errors.Is(err, ErrNotSelected) {
					t.Fatalf("Load skipped endpoint validation: %v", err)
				}
			}
		})
	}
	for _, endpoint := range []string{"http://localhost:8080/api", "http://stack.example.invalid", "https://[::1]:443/api", "https://stack.example.invalid/path%20name"} {
		selection := fixture()
		selection.Target.Endpoint = endpoint
		directory := t.TempDir()
		mustSave(t, directory, selection)
		assertLoad(t, directory, selection)
	}
}

func TestInvalidJSONFailsClosed(t *testing.T) {
	selection := fixture()
	valid, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	invalid := map[string][]byte{
		"empty":                nil,
		"null":                 []byte("null"),
		"array":                []byte("[]"),
		"truncated":            valid[:len(valid)-1],
		"unknown field":        bytes.Replace(valid, []byte(`"provider":`), []byte(`"synthetic-secret":"value","provider":`), 1),
		"unknown target field": bytes.Replace(valid, []byte(`"profile":`), []byte(`"password":"synthetic-secret","profile":`), 1),
		"extra document":       append(bytes.Clone(valid), valid...),
		"trailing malformed":   append(bytes.Clone(valid), []byte(" {broken")...),
		"trailing scalar":      append(bytes.Clone(valid), []byte(" true")...),
		"invalid provider":     bytes.Replace(valid, []byte(`"legacy"`), []byte(`"wrong"`), 1),
		"missing version":      bytes.Replace(valid, []byte(`"2.2.0"`), []byte(`""`), 1),
		"oversized":            append(bytes.Clone(valid), bytes.Repeat([]byte(" "), maxSelectionBytes)...),
		"invalid UTF-8":        bytes.Replace(valid, []byte("sandbox"), []byte{'s', 0xff}, 1),
		"duplicate provider":   bytes.Replace(valid, []byte(`"provider":`), []byte(`"provider":"modern","provider":`), 1),
		"case duplicate":       bytes.Replace(valid, []byte(`"provider":`), []byte(`"Provider":"modern","provider":`), 1),
		"duplicate target":     bytes.Replace(valid, []byte(`"profile":`), []byte(`"profile":"other","profile":`), 1),
		"null identity":        bytes.Replace(valid, []byte(`"sandbox"`), []byte(`null`), 1),
	}
	for name, data := range invalid {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			mustSave(t, directory, selection)
			writeRecord(t, directory, selection, data)
			got, err := Load(directory, selection.Target, selection.Service)
			if err == nil || errors.Is(err, ErrNotSelected) || got != (Selection{}) {
				t.Fatalf("invalid selection was usable: %+v, %v", got, err)
			}
			if strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatalf("JSON error revealed untrusted input: %v", err)
			}
			listed, err := List(directory)
			if err == nil || listed != nil {
				t.Fatalf("invalid selection was listed: %+v, %v", listed, err)
			}
		})
	}
}

func TestIdentityMismatchIsNotASelection(t *testing.T) {
	directory := t.TempDir()
	selection := fixture()
	mustSave(t, directory, selection)
	other := selection
	other.Target.Stack = "other"
	data, err := json.Marshal(other)
	if err != nil {
		t.Fatal(err)
	}
	writeRecord(t, directory, selection, data)
	if _, err := Load(directory, selection.Target, selection.Service); err == nil || errors.Is(err, ErrNotSelected) {
		t.Fatalf("Load accepted wrong identity: %v", err)
	}
	if _, err := List(directory); err == nil {
		t.Fatal("List accepted content with a mismatched filename")
	}
}

func TestSymlinksCannotEscapeSelections(t *testing.T) {
	for _, attack := range []string{"absolute directory", "relative directory", "absolute file", "relative file", "inside directory", "inside file"} {
		t.Run(attack, func(t *testing.T) {
			directory, protected, secret := installSymlinkAttack(t, attack)
			selection := fixture()
			if _, err := Load(directory, selection.Target, selection.Service); err == nil || errors.Is(err, ErrNotSelected) {
				t.Fatalf("Load followed symlink: %v", err)
			}
			if _, err := List(directory); err == nil {
				t.Fatal("List followed symlink")
			}
			if err := Save(directory, selection); err == nil {
				t.Fatal("Save followed symlink")
			}
			assertFilesUnchanged(t, protected, secret)
		})
	}
}

func assertFilesUnchanged(t *testing.T, paths []string, want []byte) {
	t.Helper()
	for _, path := range paths {
		if got := readFile(t, path); !bytes.Equal(got, want) {
			t.Fatalf("cache write changed symlink destination: %q", got)
		}
	}
}

func installSymlinkAttack(t *testing.T, attack string) (directory string, protected []string, secret []byte) {
	t.Helper()
	parent := t.TempDir()
	directory = filepath.Join(parent, "plugins")
	outside := filepath.Join(parent, "outside")
	makeDirectory(t, outside)
	selection := fixture()
	name := filename(selection.Target, selection.Service)
	secretPath := filepath.Join(outside, name)
	secret = []byte("synthetic-secret must not be read or overwritten")
	writeFile(t, secretPath, secret)
	protected = append(protected, secretPath)
	mustSave(t, directory, selection)
	link := filepath.Join(directory, "selections")
	destination := outside
	if strings.Contains(attack, "file") {
		link = filepath.Join(link, name)
		destination = secretPath
	}
	removePath(t, link)
	if strings.HasPrefix(attack, "inside") {
		destination = filepath.Join(directory, "locks")
		makeDirectory(t, destination)
		insideFile := filepath.Join(destination, name)
		writeFile(t, insideFile, secret)
		protected = append(protected, insideFile)
		if strings.Contains(attack, "file") {
			destination = insideFile
		}
	}
	if !strings.HasPrefix(attack, "absolute") {
		relative, err := filepath.Rel(filepath.Dir(link), destination)
		if err != nil {
			t.Fatal(err)
		}
		destination = relative
	}
	if err := os.Symlink(destination, link); err != nil {
		t.Fatal(err)
	}
	return directory, protected, secret
}

func TestInvalidFilesAndTemporaryFiles(t *testing.T) {
	for _, invalid := range []string{"directory record", "public record", "invalid filename"} {
		t.Run(invalid, func(t *testing.T) {
			directory := t.TempDir()
			selection := fixture()
			mustSave(t, directory, selection)
			makeInvalidFile(t, directory, selection, invalid)
			if _, err := List(directory); err == nil {
				t.Fatal("invalid file was listed")
			}
		})
	}
	directory := t.TempDir()
	selection := fixture()
	mustSave(t, directory, selection)
	writeFile(t, filepath.Join(directory, "selections", ".tmp-interrupted"), []byte("{partial"))
	listed, err := List(directory)
	if err != nil || len(listed) != 1 || listed[0] != selection {
		t.Fatalf("temporary file broke listing: %+v, %v", listed, err)
	}
}

func makeInvalidFile(t *testing.T, directory string, selection Selection, invalid string) {
	t.Helper()
	path := filepath.Join(directory, "selections", filename(selection.Target, selection.Service))
	switch invalid {
	case "directory record":
		removePath(t, path)
		makeDirectory(t, path)
		if err := Save(directory, selection); err == nil {
			t.Fatal("Save replaced a directory")
		}
	case "public record":
		if err := os.Chmod(path, 0o644); err != nil { //nolint:gosec // Fixture deliberately violates selection privacy to verify rejection.
			t.Fatal(err)
		}
	case "invalid filename":
		if err := os.Rename(path, filepath.Join(directory, "selections", "unhashed.json")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConcurrentDistinctTargets(t *testing.T) {
	directory := t.TempDir()
	const count = 48
	selections := make([]Selection, count)
	failures := make(chan error, count)
	var writers sync.WaitGroup
	for i := range count {
		selection := fixture()
		selection.Target.Stack = fmt.Sprintf("stack-%d", i)
		selection.ServiceVersion = fmt.Sprintf("2.2.%d", i)
		selections[i] = selection
		writers.Go(func() {
			for range 3 {
				if err := Save(directory, selection); err != nil {
					failures <- err
					return
				}
			}
		})
	}
	writers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	for _, selection := range selections {
		assertLoad(t, directory, selection)
	}
	listed, err := List(directory)
	if err != nil || len(listed) != count {
		t.Fatalf("concurrent writes lost distinct targets: %d, %v", len(listed), err)
	}
	entries, err := os.ReadDir(filepath.Join(directory, "selections"))
	if err != nil || len(entries) != count {
		t.Fatalf("temporary writes were not cleaned up: %d, %v", len(entries), err)
	}
}

func TestConcurrentVersionSwitchesAreAtomic(t *testing.T) {
	directory := t.TempDir()
	legacy := fixture()
	modern := legacy
	modern.Provider, modern.ServiceVersion, modern.StackVersion, modern.PluginVersion = "modern", "3.0.0", "4.0.0", ""
	mustSave(t, directory, legacy)
	failures := make(chan error, 5)
	var workers sync.WaitGroup
	for _, selection := range []Selection{legacy, modern} {
		workers.Go(func() {
			for range 30 {
				if err := Save(directory, selection); err != nil {
					failures <- err
					return
				}
			}
		})
	}
	for range 3 {
		workers.Go(func() {
			readConcurrentVersions(directory, legacy, modern, failures)
		})
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func readConcurrentVersions(directory string, legacy, modern Selection, failures chan<- error) {
	for range 100 {
		got, err := Load(directory, legacy.Target, legacy.Service)
		if err != nil || got != legacy && got != modern {
			failures <- errors.Join(fmt.Errorf("partial selection observed: %+v", got), err)
			return
		}
		listed, err := List(directory)
		if err != nil || len(listed) != 1 || listed[0] != legacy && listed[0] != modern {
			failures <- errors.Join(fmt.Errorf("partial selection listed: %+v", listed), err)
			return
		}
	}
}
