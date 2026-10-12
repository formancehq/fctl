package pluginmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type descriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitzero"`
}

type imageManifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	ArtifactType  string       `json:"artifactType"`
	Config        descriptor   `json:"config"`
	Layers        []descriptor `json:"layers"`
}

type registrySession struct {
	client     *http.Client
	registry   string
	repository string
	actions    string
	token      string
}

func (m *Manager) pull(ctx context.Context, root *os.Root, lock Lock) error {
	session := registrySession{client: m.client, registry: strings.TrimRight(lock.Artifact.Registry, "/"), repository: lock.Artifact.Repository, actions: "pull"}
	data, err := session.bytes(ctx, session.path("manifests", lock.ArtifactDigest), maxManifestBytes)
	if err != nil {
		return fmt.Errorf("download OCI manifest: %w", err)
	}
	if "sha256:"+checksum(data) != lock.ArtifactDigest {
		return fmt.Errorf("OCI manifest digest mismatch")
	}
	var manifest imageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("decode OCI manifest: %w", err)
	}
	if err := validateImage(manifest, lock.SHA256); err != nil {
		return err
	}
	config, err := session.bytes(ctx, session.path("blobs", manifest.Config.Digest), maxManifestBytes)
	if err != nil {
		return fmt.Errorf("download OCI config: %w", err)
	}
	if int64(len(config)) != manifest.Config.Size || "sha256:"+checksum(config) != manifest.Config.Digest {
		return fmt.Errorf("OCI config digest or size mismatch")
	}
	return session.binary(ctx, root, lock, manifest.Layers[0])
}

func validateImage(manifest imageManifest, binarySHA string) error {
	if manifest.SchemaVersion != 2 || manifest.MediaType != ImageManifestMediaType || manifest.ArtifactType != ArtifactMediaType {
		return fmt.Errorf("unsupported OCI plugin artifact format")
	}
	if len(manifest.Layers) != 1 || manifest.Layers[0].MediaType != ExecutableMediaType {
		return fmt.Errorf("OCI plugin requires exactly one raw executable layer")
	}
	if !validDigest(manifest.Config.Digest) || manifest.Config.Size < 0 || manifest.Config.Size > maxManifestBytes {
		return fmt.Errorf("invalid OCI config descriptor")
	}
	layer := manifest.Layers[0]
	if !validDigest(layer.Digest) || layer.Size <= 0 || layer.Size > maxBinaryBytes {
		return fmt.Errorf("invalid OCI executable descriptor")
	}
	if layer.Digest != "sha256:"+binarySHA {
		return fmt.Errorf("OCI executable digest does not match catalogue SHA256")
	}
	return nil
}

func (s *registrySession) path(kind, digest string) string {
	return s.registry + "/v2/" + s.repository + "/" + kind + "/" + digest
}

func (s *registrySession) bytes(ctx context.Context, address string, limit int64) (_ []byte, err error) {
	response, err := s.do(ctx, http.MethodGet, address, nil, "")
	if err != nil {
		return nil, err
	}
	defer func() { err = joinClose(err, response.Body) }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OCI registry returned HTTP %d", response.StatusCode)
	}
	return readBounded(response.Body, limit)
}

func (s *registrySession) binary(ctx context.Context, root *os.Root, lock Lock, layer descriptor) (err error) {
	response, err := s.do(ctx, http.MethodGet, s.path("blobs", layer.Digest), nil, "")
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, response.Body) }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("OCI executable download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength >= 0 && response.ContentLength != layer.Size {
		return fmt.Errorf("OCI executable content length does not match descriptor")
	}
	return writeBinary(ctx, root, lock, response.Body, layer.Size)
}

func (s *registrySession) do(ctx context.Context, method, address string, data []byte, contentType string) (*http.Response, error) {
	response, err := s.request(ctx, method, address, data, contentType)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusUnauthorized {
		return response, nil
	}
	challenge := response.Header.Get("WWW-Authenticate")
	if err := response.Body.Close(); err != nil {
		return nil, err
	}
	if err := s.authenticate(ctx, challenge); err != nil {
		return nil, err
	}
	return s.request(ctx, method, address, data, contentType)
}

func (s *registrySession) request(ctx context.Context, method, address string, data []byte, contentType string) (*http.Response, error) {
	u, err := downloadURL(address)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if s.token != "" && u.Scheme+"://"+u.Host == s.registry {
		request.Header.Set("Authorization", "Bearer "+s.token)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Accept", ImageManifestMediaType+", application/octet-stream")
	return s.client.Do(request)
}

func (s *registrySession) authenticate(ctx context.Context, challenge string) (err error) {
	realm, service, err := bearerChallenge(challenge)
	if err != nil {
		return err
	}
	u, err := downloadURL(realm)
	if err != nil {
		return fmt.Errorf("invalid OCI token realm: %w", err)
	}
	query := u.Query()
	query.Set("service", service)
	// Never use a registry-provided scope to request write privileges on pulls.
	query.Set("scope", "repository:"+s.repository+":"+s.actions)
	u.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, response.Body) }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("OCI anonymous token request returned HTTP %d", response.StatusCode)
	}
	data, err := readBounded(response.Body, 64<<10)
	if err != nil {
		return err
	}
	var result struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("invalid OCI token response")
	}
	s.token = result.Token
	if s.token == "" {
		s.token = result.AccessToken
	}
	if s.token == "" || strings.ContainsAny(s.token, "\r\n") {
		return fmt.Errorf("OCI token response has no valid token")
	}
	return nil
}

func bearerChallenge(challenge string) (string, string, error) {
	scheme, parameters, ok := strings.Cut(challenge, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", "", fmt.Errorf("OCI registry requires unsupported authentication")
	}
	// MIME parameters have the same quoted-string escaping as this challenge.
	// Translate only commas outside quoted strings; scope itself contains commas.
	translated := make([]rune, 0, len(parameters))
	quoted, escaped := false, false
	for _, char := range parameters {
		if char == '"' && !escaped {
			quoted = !quoted
		}
		if char == ',' && !quoted {
			char = ';'
		}
		translated = append(translated, char)
		escaped = char == '\\' && !escaped
	}
	_, values, err := mime.ParseMediaType("bearer;" + string(translated))
	if err != nil || values["realm"] == "" || values["service"] == "" {
		return "", "", fmt.Errorf("invalid OCI bearer challenge")
	}
	return values["realm"], values["service"], nil
}

// OCI registries may return relative upload locations or signed HTTPS blob
// locations. Resolve them against the responding request, preserving its query.
func uploadLocation(response *http.Response) (string, error) {
	location := response.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("OCI upload response has no location")
	}
	u, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("invalid OCI upload location")
	}
	u = response.Request.URL.ResolveReference(u)
	if _, err := downloadURL(u.String()); err != nil {
		return "", err
	}
	return u.String(), nil
}
