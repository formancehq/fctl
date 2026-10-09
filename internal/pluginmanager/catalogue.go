package pluginmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

var (
	servicePattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	platformPattern   = regexp.MustCompile(`^[a-z0-9]{1,32}$`)
	shaPattern        = regexp.MustCompile(`^[a-f0-9]{64}$`)
	repositoryPattern = regexp.MustCompile(`^[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*(/[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*)*$`)
)

func (m *Manager) resolve(ctx context.Context, source, service, version string, revision int) (Release, error) {
	if !servicePattern.MatchString(service) || !validVersion(version) || revision < 0 {
		return Release{}, fmt.Errorf("invalid plugin service, service version or revision")
	}
	data, err := m.catalogueBytes(ctx, source)
	if err != nil {
		return Release{}, err
	}
	var catalogue Catalogue
	if err := json.Unmarshal(data, &catalogue); err != nil {
		return Release{}, fmt.Errorf("decode plugin catalogue: %w", err)
	}
	if catalogue.SchemaVersion != SchemaVersion {
		return Release{}, fmt.Errorf("unsupported catalogue schema %d", catalogue.SchemaVersion)
	}
	return selectRelease(catalogue, m.platform, service, version, revision)
}

func selectRelease(catalogue Catalogue, platform Platform, service, version string, revision int) (Release, error) {
	var selected Release
	seen := make(map[string]bool)
	for _, release := range catalogue.Releases {
		if err := validateRelease(release, false); err != nil {
			return Release{}, fmt.Errorf("invalid catalogue release: %w", err)
		}
		key := fmt.Sprintf("%s/%s/%d/%s", release.Service, release.ServiceVersion, release.Revision, release.Platform)
		if seen[key] {
			return Release{}, fmt.Errorf("duplicate catalogue release %s", key)
		}
		seen[key] = true
		if release.Service == service && release.ServiceVersion == version && release.Platform == platform &&
			(revision == 0 || release.Revision == revision) && release.Revision > selected.Revision {
			selected = release
		}
	}
	if selected.Revision == 0 {
		return Release{}, fmt.Errorf("%w: %s %s (%s), revision %d", ErrNoRelease, service, version, platform, revision)
	}
	if selected.Manifest.ProtocolVersion != pluginsdk.ProtocolVersion {
		return Release{}, fmt.Errorf("unsupported plugin protocol %d (host supports %d)", selected.Manifest.ProtocolVersion, pluginsdk.ProtocolVersion)
	}
	return selected, nil
}

func (m *Manager) catalogueBytes(ctx context.Context, source string) (_ []byte, err error) {
	if source == "" {
		return nil, fmt.Errorf("configure a plugin catalogue URL or file")
	}
	source, err = catalogueSource(source)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		u, err := downloadURL(source)
		if err != nil {
			return nil, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		response, err := m.client.Do(request)
		if err != nil {
			return nil, err
		}
		defer func() { err = joinClose(err, response.Body) }()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("plugin catalogue returned HTTP %d", response.StatusCode)
		}
		return readBounded(response.Body, maxCatalogueBytes)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(source) //nolint:gosec // An explicit local catalogue path is selected by the caller, not by a remote manifest.
	if err != nil {
		return nil, fmt.Errorf("open plugin catalogue: %w", err)
	}
	defer func() { err = joinClose(err, file) }()
	if err := regularFile(file); err != nil {
		return nil, err
	}
	return readBounded(file, maxCatalogueBytes)
}

func catalogueSource(source string) (string, error) {
	if source == "" {
		return "", nil
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		if _, err := downloadURL(source); err != nil {
			return "", err
		}
		return source, nil
	}
	if strings.Contains(source, "://") {
		return "", fmt.Errorf("catalogue must be an HTTPS URL or an explicit file path")
	}
	return filepath.Abs(source)
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("plugin data exceeds %d bytes", limit)
	}
	return data, nil
}

func validVersion(version string) bool {
	return version != "" && len(version) <= 128 && strings.TrimSpace(version) == version && !strings.ContainsAny(version, "/\\\x00\r\n\t")
}

func validateRelease(release Release, local bool) error {
	if !servicePattern.MatchString(release.Service) || !validVersion(release.ServiceVersion) || release.Revision < 1 {
		return fmt.Errorf("invalid service, exact version or revision")
	}
	if !platformPattern.MatchString(release.Platform.OS) || !platformPattern.MatchString(release.Platform.Arch) {
		return fmt.Errorf("invalid plugin platform")
	}
	if !shaPattern.MatchString(release.SHA256) {
		return fmt.Errorf("invalid executable SHA256")
	}
	if err := validateManifest(release.Manifest, release.Service); err != nil {
		return err
	}
	if local {
		return nil
	}
	if err := validateArtifact(release.Artifact); err != nil {
		return err
	}
	return nil
}

func validateManifest(manifest pluginsdk.Manifest, service string) error {
	if manifest.Service != service || !servicePattern.MatchString(manifest.Name) || !validVersion(manifest.Version) ||
		pluginsdk.CommandName(manifest.Root) != manifest.Name || manifest.ProtocolVersion < 1 {
		return fmt.Errorf("manifest must name the release service, plugin version, root command and protocol")
	}
	return nil
}

func validateArtifact(artifact Artifact) error {
	u, err := downloadURL(artifact.Registry)
	if err != nil {
		return err
	}
	if u.Path != "" && u.Path != "/" || u.RawQuery != "" {
		return fmt.Errorf("registry must be an HTTPS origin")
	}
	if len(artifact.Repository) > 255 || !repositoryPattern.MatchString(artifact.Repository) {
		return fmt.Errorf("invalid OCI repository")
	}
	if !validDigest(artifact.Digest) {
		return fmt.Errorf("OCI artifact must use an immutable sha256 digest")
	}
	return nil
}

func validDigest(digest string) bool {
	checksum, ok := strings.CutPrefix(digest, "sha256:")
	return ok && shaPattern.MatchString(checksum)
}

func downloadURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("invalid public plugin URL")
	}
	if u.Scheme == "https" {
		return u, nil
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme == "http" && (u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()) {
		return u, nil
	}
	return nil, fmt.Errorf("public plugin downloads require HTTPS (HTTP is allowed on loopback only)")
}

func validateTarget(target Target) error {
	for _, field := range []string{target.Profile, target.Organization, target.Stack} {
		if len(field) > 256 || strings.ContainsAny(field, "\x00\r\n") {
			return fmt.Errorf("invalid plugin target identity")
		}
	}
	u, err := url.Parse(target.Endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" ||
		(u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("plugin target endpoint must be a stable HTTP(S) URL without credentials, query or fragment")
	}
	return nil
}
