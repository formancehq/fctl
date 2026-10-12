package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Publish uploads a raw executable, empty config and OCI manifest, in that
// order, and returns the complete catalogue release with an immutable digest.
// reference is an HTTPS registry/repository URL without a tag or digest, for
// example https://ghcr.io/formancehq/fctl-plugin-ledger. Platform and manifest
// come from release; SHA256 and Artifact are filled from the uploaded bytes.
// Anonymous push works against a local registry. For authenticated publishing,
// use the release job's OCI tool or an explicitly authenticated HTTP transport.
// Publish never rewrites an existing target lock or updates a catalogue file.
func (m *Manager) Publish(ctx context.Context, reference string, release Release, binary string) (_ Release, err error) {
	u, err := downloadURL(reference)
	if err != nil {
		return Release{}, err
	}
	if u.RawQuery != "" {
		return Release{}, fmt.Errorf("OCI publish reference must not have a query")
	}
	file, err := os.Open(binary) //nolint:gosec // The release publisher explicitly chooses a local build artifact.
	if err != nil {
		return Release{}, err
	}
	defer func() { err = joinClose(err, file) }()
	if err := regularFile(file); err != nil {
		return Release{}, err
	}
	data, err := readBounded(&contextReader{ctx: ctx, reader: file}, maxBinaryBytes)
	if err != nil {
		return Release{}, err
	}
	if len(data) == 0 {
		return Release{}, fmt.Errorf("cannot publish an empty executable")
	}
	release.SHA256 = checksum(data)
	release.Artifact = Artifact{Registry: u.Scheme + "://" + u.Host, Repository: strings.TrimPrefix(u.Path, "/"), Digest: "sha256:" + checksum(nil)}
	if err := validateRelease(release, false); err != nil {
		return Release{}, err
	}
	config := []byte("{}")
	manifest := imageManifest{
		SchemaVersion: 2, MediaType: ImageManifestMediaType, ArtifactType: ArtifactMediaType,
		Config: descriptor{MediaType: EmptyConfigMediaType, Digest: "sha256:" + checksum(config), Size: int64(len(config))},
		Layers: []descriptor{{MediaType: ExecutableMediaType, Digest: "sha256:" + release.SHA256, Size: int64(len(data)),
			Annotations: map[string]string{"org.opencontainers.image.title": "fctl-plugin-" + release.Manifest.Name}}},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return Release{}, err
	}
	release.Artifact.Digest = "sha256:" + checksum(manifestBytes)
	session := registrySession{client: m.client, registry: release.Artifact.Registry, repository: release.Artifact.Repository, actions: "pull,push"}
	if err := session.pushBlob(ctx, manifest.Config.Digest, config); err != nil {
		return Release{}, err
	}
	if err := session.pushBlob(ctx, manifest.Layers[0].Digest, data); err != nil {
		return Release{}, err
	}
	if err := session.pushManifest(ctx, release.Artifact.Digest, manifestBytes); err != nil {
		return Release{}, err
	}
	return release, nil
}

func (s *registrySession) pushBlob(ctx context.Context, digest string, data []byte) (err error) {
	response, err := s.do(ctx, http.MethodPost, s.path("blobs/uploads", ""), nil, "")
	if err != nil {
		return err
	}
	location, locationErr := uploadLocation(response)
	closeErr := response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("OCI upload initialization returned HTTP %d", response.StatusCode)
	}
	if locationErr != nil || closeErr != nil {
		return errors.Join(locationErr, closeErr)
	}
	u, err := url.Parse(location)
	if err != nil {
		return err
	}
	query := u.Query()
	query.Set("digest", digest)
	u.RawQuery = query.Encode()
	response, err = s.do(ctx, http.MethodPut, u.String(), data, "application/octet-stream")
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, response.Body) }()
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("OCI blob upload returned HTTP %d", response.StatusCode)
	}
	if actual := response.Header.Get("Docker-Content-Digest"); actual != "" && actual != digest {
		return fmt.Errorf("published OCI blob digest mismatch")
	}
	return nil
}

func (s *registrySession) pushManifest(ctx context.Context, digest string, data []byte) (err error) {
	response, err := s.do(ctx, http.MethodPut, s.path("manifests", digest), data, ImageManifestMediaType)
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, response.Body) }()
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("OCI manifest publication returned HTTP %d", response.StatusCode)
	}
	if actual := response.Header.Get("Docker-Content-Digest"); actual != "" && actual != digest {
		return fmt.Errorf("published OCI manifest digest mismatch")
	}
	return nil
}
