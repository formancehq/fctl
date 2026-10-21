package pluginmanager

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/flock"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

// New opens an explicitly selected cache directory. The HTTP client is copied;
// its transport may customize TLS/proxies, but must not inject service tokens.
func New(directory string, client *http.Client) (*Manager, error) {
	if directory == "" {
		return nil, fmt.Errorf("plugin cache directory is required")
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	if err := root.Close(); err != nil {
		return nil, err
	}
	return &Manager{directory: directory, client: publicClient(client), platform: CurrentPlatform()}, nil
}

func publicClient(original *http.Client) *http.Client {
	client := http.Client{Timeout: 2 * time.Minute}
	if original != nil {
		client = *original
		client.Timeout = cmp.Or(client.Timeout, 2*time.Minute)
	}
	previous := client.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if _, err := downloadURL(request.URL.String()); err != nil {
			return err
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many plugin download redirects")
		}
		if len(via) > 0 && request.URL.Host != via[0].URL.Host {
			request.Header.Del("Authorization")
		}
		if previous != nil {
			return previous(request, via)
		}
		return nil
	}
	return &client
}

func (m *Manager) Sync(ctx context.Context, catalogue string, target Target, service, version string) (Lock, error) {
	return m.SyncRevision(ctx, catalogue, target, service, version, 0)
}

func (m *Manager) SyncRevision(ctx context.Context, catalogue string, target Target, service, version string, revision int) (Lock, error) {
	if err := validateTarget(target); err != nil {
		return Lock{}, err
	}
	release, err := m.Resolve(ctx, catalogue, service, version, revision)
	if err != nil {
		return Lock{}, err
	}
	return m.install(ctx, catalogue, target, release)
}

func (m *Manager) install(ctx context.Context, catalogue string, target Target, release Release) (Lock, error) {
	if err := validateTarget(target); err != nil {
		return Lock{}, err
	}
	if err := validateRelease(release, false); err != nil {
		return Lock{}, err
	}
	if release.Platform != m.platform || release.Manifest.ProtocolVersion != pluginsdk.ProtocolVersion {
		return Lock{}, fmt.Errorf("plugin platform or protocol is incompatible with this host")
	}
	catalogue, err := catalogueSource(catalogue)
	if err != nil {
		return Lock{}, err
	}
	lock := newLock(target, release)
	lock.Catalogue = catalogue
	if err := m.withCacheLock(ctx, lock, func(root *os.Root) error {
		if err := verifyBinary(ctx, root, lock); err == nil {
			return saveLock(root, lock)
		}
		if err := m.pull(ctx, root, lock); err != nil {
			return err
		}
		return saveLock(root, lock)
	}); err != nil {
		return Lock{}, err
	}
	return lock, nil
}

func newLock(target Target, release Release) Lock {
	return Lock{
		SchemaVersion: SchemaVersion, Target: target, Service: release.Service,
		ServiceVersion: release.ServiceVersion, Revision: release.Revision,
		ArtifactDigest: release.Artifact.Digest, Platform: release.Platform,
		SHA256: release.SHA256, Manifest: release.Manifest, Artifact: release.Artifact,
		InstalledAt: time.Now().UTC(),
	}
}

// InstallLocal accepts a manifest inspected by the caller. Its version is used
// as the local service version; no remote catalogue is consulted or retained.
func (m *Manager) InstallLocal(ctx context.Context, binary string, target Target, manifest pluginsdk.Manifest) (_ Lock, err error) {
	if err := validateTarget(target); err != nil {
		return Lock{}, err
	}
	if err := validateManifest(manifest, manifest.Service); err != nil {
		return Lock{}, err
	}
	if manifest.ProtocolVersion != pluginsdk.ProtocolVersion {
		return Lock{}, fmt.Errorf("unsupported plugin protocol %d", manifest.ProtocolVersion)
	}
	file, err := os.Open(binary) //nolint:gosec // Local install explicitly selects a source executable; remote entries cannot supply local paths.
	if err != nil {
		return Lock{}, err
	}
	defer func() { err = joinClose(err, file) }()
	if err := regularFile(file); err != nil {
		return Lock{}, err
	}
	checksum, _, err := hashReader(ctx, file, maxBinaryBytes)
	if err != nil {
		return Lock{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Lock{}, err
	}
	release := Release{Service: manifest.Service, ServiceVersion: manifest.Version, Revision: 1,
		Platform: m.platform, Artifact: Artifact{Digest: "sha256:" + checksum}, SHA256: checksum, Manifest: manifest}
	lock := newLock(target, release)
	lock.Local = true
	if err := m.withCacheLock(ctx, lock, func(root *os.Root) error {
		if err := verifyBinary(ctx, root, lock); err != nil {
			if err := writeBinary(ctx, root, lock, file, -1); err != nil {
				return err
			}
		}
		return saveLock(root, lock)
	}); err != nil {
		return Lock{}, err
	}
	return lock, nil
}

// Load reads only a target's cached manifest and lock, without network access.
func (m *Manager) Load(target Target, service string) (_ Lock, err error) {
	if err := validateTarget(target); err != nil {
		return Lock{}, err
	}
	if !servicePattern.MatchString(service) {
		return Lock{}, fmt.Errorf("invalid plugin service")
	}
	root, err := os.OpenRoot(m.directory)
	if err != nil {
		return Lock{}, err
	}
	defer func() { err = joinClose(err, root) }()
	lock, err := readLock(root, lockPath(target, service))
	if errors.Is(err, os.ErrNotExist) {
		return Lock{}, fmt.Errorf("%w: %s", ErrNotInstalled, service)
	}
	if err != nil {
		return Lock{}, err
	}
	if lock.Target != target || lock.Service != service {
		return Lock{}, fmt.Errorf("cached plugin lock does not match the target")
	}
	return lock, nil
}

// Binary verifies checksum, size, platform and protocol before returning the
// cached executable path. It never downloads or updates a plugin.
func (m *Manager) Binary(lock Lock) (_ string, err error) {
	return m.BinaryContext(context.Background(), lock)
}

// BinaryContext is Binary with cancellation during checksum verification.
func (m *Manager) BinaryContext(ctx context.Context, lock Lock) (_ string, err error) {
	if err := validateLock(lock); err != nil {
		return "", err
	}
	if lock.Platform != m.platform || lock.Manifest.ProtocolVersion != pluginsdk.ProtocolVersion {
		return "", fmt.Errorf("cached plugin platform or protocol is incompatible with this host")
	}
	root, err := os.OpenRoot(m.directory)
	if err != nil {
		return "", err
	}
	defer func() { err = joinClose(err, root) }()
	if err := verifyBinary(ctx, root, lock); err != nil {
		return "", fmt.Errorf("cached plugin is unavailable or corrupt; run plugins sync: %w", err)
	}
	return filepath.Join(m.directory, binaryPath(lock)), nil
}

func (m *Manager) List() (_ []Lock, err error) {
	root, err := os.OpenRoot(m.directory)
	if err != nil {
		return nil, err
	}
	defer func() { err = joinClose(err, root) }()
	directory, err := root.Open("targets")
	if errors.Is(err, os.ErrNotExist) {
		return []Lock{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries, readErr := directory.ReadDir(-1)
	if err := errors.Join(readErr, directory.Close()); err != nil {
		return nil, err
	}
	locks := make([]Lock, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		lock, err := readLock(root, filepath.Join("targets", entry.Name()))
		if err != nil {
			return nil, err
		}
		if entry.Name() != filepath.Base(lockPath(lock.Target, lock.Service)) {
			return nil, fmt.Errorf("cached plugin lock filename does not match its target")
		}
		locks = append(locks, lock)
	}
	slices.SortFunc(locks, func(a, b Lock) int {
		return strings.Compare(lockPath(a.Target, a.Service), lockPath(b.Target, b.Service))
	})
	return locks, nil
}

func validateLock(lock Lock) error {
	if lock.SchemaVersion != SchemaVersion || !validDigest(lock.ArtifactDigest) {
		return fmt.Errorf("invalid cached plugin lock schema or artifact digest")
	}
	if err := validateTarget(lock.Target); err != nil {
		return err
	}
	if lock.Artifact.Digest != lock.ArtifactDigest {
		return fmt.Errorf("cached plugin artifact digest does not match its lock")
	}
	return validateRelease(Release{Service: lock.Service, ServiceVersion: lock.ServiceVersion,
		Revision: lock.Revision, Platform: lock.Platform, Artifact: lock.Artifact,
		SHA256: lock.SHA256, Manifest: lock.Manifest}, lock.Local)
}

func readLock(root *os.Root, name string) (_ Lock, err error) {
	file, err := root.Open(name)
	if err != nil {
		return Lock{}, err
	}
	defer func() { err = joinClose(err, file) }()
	if err := regularFile(file); err != nil {
		return Lock{}, err
	}
	data, err := readBounded(file, maxCatalogueBytes)
	if err != nil {
		return Lock{}, err
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return Lock{}, fmt.Errorf("decode cached plugin lock: %w", err)
	}
	return lock, validateLock(lock)
}

func lockPath(target Target, service string) string {
	// Validated identities cannot contain NUL; separators retain empty fields.
	data := strings.Join([]string{target.Profile, target.Organization, target.Stack, target.Endpoint, service}, "\x00")
	return filepath.Join("targets", checksum([]byte(data))+".json")
}

func binaryPath(lock Lock) string {
	name := "plugin"
	if lock.Platform.OS == "windows" {
		name += ".exe"
	}
	return filepath.Join("artifacts", strings.TrimPrefix(lock.ArtifactDigest, "sha256:"), lock.Platform.OS+"_"+lock.Platform.Arch, name)
}

func (m *Manager) withCacheLock(ctx context.Context, lock Lock, work func(*os.Root) error) (err error) {
	root, err := os.OpenRoot(m.directory)
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, root) }()
	if err := root.MkdirAll("guards", 0o700); err != nil {
		return err
	}
	name := filepath.Join("guards", strings.TrimPrefix(lock.ArtifactDigest, "sha256:")+"-"+lock.Platform.OS+"_"+lock.Platform.Arch+".lock")
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if errors.Is(err, os.ErrExist) || errors.Is(err, os.ErrNotExist) {
		// Root's containment check and another creator may race. Once a creator
		// wins, open the existing file without the create flag.
		file, err = root.OpenFile(name, os.O_RDWR, 0o600)
	}
	if err != nil {
		return err
	}
	if err := regularFile(file); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	guard := flock.New(filepath.Join(m.directory, name))
	defer func() { err = errors.Join(err, guard.Close()) }()
	locked, err := guard.TryLockContext(ctx, 20*time.Millisecond)
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("plugin installation lock unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return work(root)
}

func verifyBinary(ctx context.Context, root *os.Root, lock Lock) (err error) {
	file, err := root.Open(binaryPath(lock))
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, file) }()
	if err := regularFile(file); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if lock.Platform.OS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("cached plugin is not executable")
	}
	hash, size, err := hashReader(ctx, file, maxBinaryBytes)
	if err != nil {
		return err
	}
	if size == 0 || hash != lock.SHA256 {
		return fmt.Errorf("cached executable SHA256 mismatch")
	}
	return nil
}

func regularFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("plugin data must be a regular file")
	}
	return nil
}

func saveLock(root *os.Root, lock Lock) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return atomicFile(root, lockPath(lock.Target, lock.Service), 0o600, func(file *os.File) error {
		_, err := file.Write(append(data, '\n'))
		return err
	})
}

func writeBinary(ctx context.Context, root *os.Root, lock Lock, reader io.Reader, expectedSize int64) error {
	return atomicFile(root, binaryPath(lock), 0o700, func(file *os.File) error {
		hash := sha256.New()
		size, err := io.Copy(io.MultiWriter(file, hash), &contextReader{ctx: ctx, reader: io.LimitReader(reader, maxBinaryBytes+1)})
		if err != nil {
			return err
		}
		if size == 0 || size > maxBinaryBytes || expectedSize >= 0 && size != expectedSize {
			return fmt.Errorf("invalid executable size: received %d, expected %d", size, expectedSize)
		}
		if hex.EncodeToString(hash.Sum(nil)) != lock.SHA256 {
			return fmt.Errorf("downloaded executable SHA256 mismatch")
		}
		return nil
	})
}

func atomicFile(root *os.Root, name string, mode os.FileMode, write func(*os.File) error) (err error) {
	if err := root.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return err
	}
	temporary := filepath.Join(filepath.Dir(name), ".tmp-"+rand.Text())
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer func() {
		if cleanup := root.Remove(temporary); cleanup != nil && !errors.Is(cleanup, os.ErrNotExist) {
			err = errors.Join(err, cleanup)
		}
	}()
	writeErr := write(file)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return root.Rename(temporary, name)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(data)
}

func hashReader(ctx context.Context, reader io.Reader, limit int64) (string, int64, error) {
	hash := sha256.New()
	size, err := io.Copy(hash, &contextReader{ctx: ctx, reader: io.LimitReader(reader, limit+1)})
	if err != nil {
		return "", 0, err
	}
	if size > limit {
		return "", 0, fmt.Errorf("plugin executable exceeds %d bytes", limit)
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func checksum(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func joinClose(err error, closer io.Closer) error { return errors.Join(err, closer.Close()) }
