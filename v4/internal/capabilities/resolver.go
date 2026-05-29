package capabilities

import (
	"errors"
	"fmt"
	"strings"
)

type VersionPolicy string

const (
	VersionPolicyLatestCompatible VersionPolicy = "latest-compatible"
	VersionPolicyPinned           VersionPolicy = "pinned"
	VersionPolicyLatest           VersionPolicy = "latest"
)

type VersionResolutionRequest struct {
	Product          Product
	Feature          Feature
	ComponentVersion string
	Compatibility    ComponentCompatibility
	HandlerVersions  []APIVersion
	Policy           VersionPolicy
	PinnedVersion    APIVersion
}

type UnsupportedFeatureError struct {
	Product          Product
	Feature          Feature
	ComponentVersion string
	Supported        []APIVersion
	Handlers         []APIVersion
	PinnedVersion    APIVersion
}

func (e *UnsupportedFeatureError) Error() string {
	if e.PinnedVersion != "" {
		return fmt.Sprintf("%s.%s does not support pinned api version %s; target supports %s and command supports %s",
			e.Product, e.Feature, e.PinnedVersion, joinAPIVersions(e.Supported), joinAPIVersions(e.Handlers))
	}
	return fmt.Sprintf("%s.%s is not supported by component version %s; target supports %s and command supports %s",
		e.Product, e.Feature, e.ComponentVersion, joinAPIVersions(e.Supported), joinAPIVersions(e.Handlers))
}

func ResolveAPIVersion(request VersionResolutionRequest) (APIVersion, error) {
	if request.Policy == "" {
		request.Policy = VersionPolicyLatestCompatible
	}
	if request.Product == "" {
		return "", errors.New("product is required")
	}
	if request.ComponentVersion == "" {
		return "", errors.New("component version is required")
	}
	if len(request.HandlerVersions) == 0 {
		return "", errors.New("handler versions are required")
	}

	supported, err := request.Compatibility.APIVersionsFor(request.Product, request.ComponentVersion)
	if err != nil {
		return "", err
	}
	handlers := UniqueSortedAPIVersions(request.HandlerVersions)

	if request.Policy == VersionPolicyPinned {
		if request.PinnedVersion == "" {
			return "", errors.New("pinned api version is required")
		}
		if containsAPIVersion(supported, request.PinnedVersion) && containsAPIVersion(handlers, request.PinnedVersion) {
			return request.PinnedVersion, nil
		}
		return "", &UnsupportedFeatureError{
			Product:          request.Product,
			Feature:          request.Feature,
			ComponentVersion: request.ComponentVersion,
			Supported:        supported,
			Handlers:         handlers,
			PinnedVersion:    request.PinnedVersion,
		}
	}

	intersection := intersectAPIVersions(supported, handlers)
	if selected, ok := HighestAPIVersion(intersection); ok {
		return selected, nil
	}
	return "", &UnsupportedFeatureError{
		Product:          request.Product,
		Feature:          request.Feature,
		ComponentVersion: request.ComponentVersion,
		Supported:        supported,
		Handlers:         handlers,
	}
}

func intersectAPIVersions(a, b []APIVersion) []APIVersion {
	seen := map[APIVersion]struct{}{}
	for _, version := range a {
		seen[version] = struct{}{}
	}
	var ret []APIVersion
	for _, version := range b {
		if _, ok := seen[version]; ok {
			ret = append(ret, version)
		}
	}
	return UniqueSortedAPIVersions(ret)
}

func containsAPIVersion(versions []APIVersion, version APIVersion) bool {
	for _, candidate := range versions {
		if candidate == version {
			return true
		}
	}
	return false
}

func joinAPIVersions(versions []APIVersion) string {
	versions = UniqueSortedAPIVersions(versions)
	if len(versions) == 0 {
		return "<none>"
	}
	parts := make([]string, len(versions))
	for i, version := range versions {
		parts[i] = string(version)
	}
	return strings.Join(parts, ",")
}
