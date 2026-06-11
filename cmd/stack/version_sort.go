package stack

import (
	"sort"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
)

func sortRegionVersionsByLatest(versions []components.Version) []components.Version {
	sorted := append([]components.Version(nil), versions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareVersionNamesByLatest(sorted[i].GetName(), sorted[j].GetName())
	})
	return sorted
}

func sortVersionNamesByLatest(versions []string) []string {
	sorted := append([]string(nil), versions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareVersionNamesByLatest(sorted[i], sorted[j])
	})
	return sorted
}

func compareVersionNamesByLatest(a, b string) bool {
	normalizedA, validA := normalizeSemver(a)
	normalizedB, validB := normalizeSemver(b)

	if validA && validB {
		return semver.Compare(normalizedA, normalizedB) > 0
	}
	if validA != validB {
		return validA
	}
	return a > b
}

func isVersionNewerThanCurrent(candidate, current string) bool {
	normalizedCandidate, validCandidate := normalizeSemver(candidate)
	normalizedCurrent, validCurrent := normalizeSemver(current)
	if validCandidate && validCurrent {
		return semver.Compare(normalizedCandidate, normalizedCurrent) > 0
	}
	return true
}

func normalizeSemver(version string) (string, bool) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", false
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if semver.IsValid(version) {
		return version, true
	}

	suffixStart := len(version)
	for _, separator := range []string{"-", "+"} {
		if index := strings.Index(version, separator); index >= 0 && index < suffixStart {
			suffixStart = index
		}
	}

	core := strings.TrimPrefix(version[:suffixStart], "v")
	suffix := version[suffixStart:]
	parts := strings.Split(core, ".")
	if len(parts) > 3 {
		return "", false
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	for _, part := range parts {
		if part == "" {
			return "", false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return "", false
			}
		}
	}

	normalized := "v" + strings.Join(parts, ".") + suffix
	return normalized, semver.IsValid(normalized)
}
