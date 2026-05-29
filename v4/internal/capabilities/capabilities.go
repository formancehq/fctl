// Package capabilities models stack component versions, API namespaces, and
// generated operation metadata used by the v4 runtime resolver.
package capabilities

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

type Product string
type Feature string
type APIVersion string

type Manifest struct {
	SpecVersion string                      `json:"specVersion"`
	Products    map[Product]ProductManifest `json:"products"`
}

type ProductManifest struct {
	APIVersions []APIVersion                         `json:"apiVersions"`
	Operations  map[Feature]map[APIVersion]Operation `json:"operations"`
}

type Operation struct {
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Tags        []string `json:"tags,omitempty"`
}

type ComponentVersion struct {
	Product Product
	Version string
	Health  bool
}

type ComponentRange struct {
	Product     Product
	Range       string
	APIVersions []APIVersion
}

type ComponentCompatibility []ComponentRange

func (c ComponentCompatibility) APIVersionsFor(product Product, componentVersion string) ([]APIVersion, error) {
	var matches []APIVersion
	for _, candidate := range c {
		if candidate.Product != product {
			continue
		}
		ok, err := MatchVersionRange(componentVersion, candidate.Range)
		if err != nil {
			return nil, fmt.Errorf("match %s range %q: %w", product, candidate.Range, err)
		}
		if ok {
			matches = append(matches, candidate.APIVersions...)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no api versions for %s component version %q", product, componentVersion)
	}
	return UniqueSortedAPIVersions(matches), nil
}

func MatchVersionRange(version string, versionRange string) (bool, error) {
	normalizedVersion, err := normalizeSemver(version)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(versionRange) == "" {
		return false, errors.New("range cannot be empty")
	}

	for _, constraint := range strings.Fields(versionRange) {
		if constraint == "" {
			continue
		}
		if ok, err := matchConstraint(normalizedVersion, constraint); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

func matchConstraint(version string, constraint string) (bool, error) {
	for _, operator := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(constraint, operator) {
			target, err := normalizeSemver(strings.TrimPrefix(constraint, operator))
			if err != nil {
				return false, err
			}
			cmp := semver.Compare(version, target)
			switch operator {
			case ">=":
				return cmp >= 0, nil
			case "<=":
				return cmp <= 0, nil
			case ">":
				return cmp > 0, nil
			case "<":
				return cmp < 0, nil
			case "=":
				return cmp == 0, nil
			}
		}
	}

	target, err := normalizeSemver(constraint)
	if err != nil {
		return false, err
	}
	return semver.Compare(version, target) == 0, nil
}

func normalizeSemver(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", errors.New("version cannot be empty")
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid semver %q", version)
	}
	return version, nil
}

func UniqueSortedAPIVersions(versions []APIVersion) []APIVersion {
	seen := map[APIVersion]struct{}{}
	ret := make([]APIVersion, 0, len(versions))
	for _, version := range versions {
		if _, ok := seen[version]; ok {
			continue
		}
		seen[version] = struct{}{}
		ret = append(ret, version)
	}
	sort.Slice(ret, func(i, j int) bool {
		return compareAPIVersion(ret[i], ret[j]) < 0
	})
	return ret
}

func HighestAPIVersion(versions []APIVersion) (APIVersion, bool) {
	versions = UniqueSortedAPIVersions(versions)
	if len(versions) == 0 {
		return "", false
	}
	return versions[len(versions)-1], true
}

func compareAPIVersion(a, b APIVersion) int {
	an := apiVersionNumber(a)
	bn := apiVersionNumber(b)
	switch {
	case an < bn:
		return -1
	case an > bn:
		return 1
	default:
		return strings.Compare(string(a), string(b))
	}
}

func apiVersionNumber(version APIVersion) int {
	raw := strings.TrimPrefix(string(version), "v")
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}
