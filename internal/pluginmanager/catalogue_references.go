package pluginmanager

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

func validateReferences(catalogue Catalogue) error {
	if catalogue.Releases != nil || catalogue.Plugins == nil {
		return fmt.Errorf("registry schema 2 requires plugins and must not embed releases")
	}
	for service, product := range catalogue.Plugins {
		if !servicePattern.MatchString(service) || len(product.Releases) == 0 {
			return fmt.Errorf("registry requires a valid service and at least one catalogue reference")
		}
		seen := make(map[string]bool)
		for _, reference := range product.Releases {
			if err := validateReference(reference); err != nil {
				return fmt.Errorf("invalid %s catalogue reference: %w", service, err)
			}
			if seen[reference.ServiceVersion] {
				return fmt.Errorf("duplicate catalogue reference for %s %s", service, reference.ServiceVersion)
			}
			seen[reference.ServiceVersion] = true
		}
	}
	return nil
}

func validateReference(reference CatalogueReference) error {
	if !validVersion(reference.ServiceVersion) || strings.HasPrefix(reference.ServiceVersion, "v") {
		return fmt.Errorf("exact service version without v prefix is required")
	}
	if !shaPattern.MatchString(reference.SHA256) {
		return fmt.Errorf("invalid catalogue SHA256")
	}
	_, err := downloadURL(reference.Catalogue)
	return err
}

// HasService checks advertised services without downloading product catalogues.
// A schema-2 index defers platform selection until its selected catalogue is read.
func (c Catalogue) HasService(service string, platform Platform) bool {
	if c.SchemaVersion == RegistrySchemaVersion {
		return len(c.Plugins[service].Releases) > 0
	}
	for _, release := range c.Releases {
		if release.Service == service && release.Platform == platform {
			return true
		}
	}
	return false
}

// ResolveCatalogue resolves one exact service version from an inspected index
// or a schema-1 catalogue. Only the selected reference is downloaded.
func (m *Manager) ResolveCatalogue(ctx context.Context, catalogue Catalogue, service, version string, revision int) (Release, error) {
	if !servicePattern.MatchString(service) || !validVersion(version) || revision < 0 {
		return Release{}, fmt.Errorf("invalid plugin service, service version or revision")
	}
	if err := validateCatalogue(catalogue); err != nil {
		return Release{}, err
	}
	if catalogue.SchemaVersion == SchemaVersion {
		return catalogue.Resolve(m.platform, service, version, revision)
	}
	reference, err := findReference(catalogue, service, version)
	if err != nil {
		return Release{}, err
	}
	product, err := m.readReferencedCatalogue(ctx, reference, service)
	if err != nil {
		return Release{}, err
	}
	return product.Resolve(m.platform, service, version, revision)
}

func findReference(catalogue Catalogue, service, version string) (CatalogueReference, error) {
	for _, reference := range catalogue.Plugins[service].Releases {
		if reference.ServiceVersion == version {
			return reference, nil
		}
	}
	return CatalogueReference{}, fmt.Errorf("%w: no catalogue for %s %s", ErrNoRelease, service, version)
}

func (m *Manager) readReferencedCatalogue(ctx context.Context, reference CatalogueReference, service string) (Catalogue, error) {
	data, err := m.catalogueBytes(ctx, reference.Catalogue)
	if err != nil {
		return Catalogue{}, fmt.Errorf("read %s %s catalogue: %w", service, reference.ServiceVersion, err)
	}
	if checksum(data) != reference.SHA256 {
		return Catalogue{}, fmt.Errorf("referenced catalogue SHA256 mismatch for %s %s", service, reference.ServiceVersion)
	}
	var product Catalogue
	if err := yaml.UnmarshalStrict(data, &product); err != nil {
		return Catalogue{}, fmt.Errorf("decode referenced catalogue: %w", err)
	}
	if product.SchemaVersion != SchemaVersion {
		return Catalogue{}, fmt.Errorf("referenced catalogue must use schema 1; nested references are unsupported")
	}
	if err := validateCatalogue(product); err != nil {
		return Catalogue{}, err
	}
	for _, release := range product.Releases {
		if release.Service != service || release.ServiceVersion != reference.ServiceVersion {
			return Catalogue{}, fmt.Errorf("referenced catalogue contains a different service or version")
		}
	}
	return product, nil
}
