// Package legacy supplies the historical fctl service commands through the public SDK.
package legacy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

const Version = metadata.Version
const SourceCommit = metadata.SourceCommit

// Compatibility is an explicit supported API family, independent of the plugin revision.
// A prerelease of the next major belongs to that next family.
type Compatibility struct {
	Service  string `json:"service"`
	MinMajor int    `json:"minMajor"`
	MaxMajor int    `json:"maxMajor"`
}

func Compatibilities() []Compatibility {
	return []Compatibility{{"ledger", 1, 2}, {"auth", 1, 2}, {"payments", 1, 3}, {"orchestration", 1, 2}, {"reconciliation", 1, 2}, {"wallets", 1, 2}, {"webhooks", 1, 2}}
}

func Supported(service, version, stackVersion string) bool {
	if stackVersion != "" && !LegacyStack(stackVersion) {
		return false
	}
	major, ok := versionMajor(version)
	if !ok {
		return false
	}
	for _, c := range Compatibilities() {
		if c.Service == service {
			return major >= c.MinMajor && major <= c.MaxMajor
		}
	}
	return false
}

func LegacyStack(version string) bool {
	major, ok := versionMajor(version)
	return ok && major >= 1 && major <= 3
}

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:\.(0|[1-9][0-9]*))?(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?(?:\+[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$`)

func versionMajor(version string) (int, bool) {
	version = strings.TrimPrefix(version, "v")
	match := versionPattern.FindStringSubmatch(version)
	if match == nil {
		return 0, false
	}
	major, err := strconv.Atoi(match[1])
	return major, err == nil
}

func Unsupported(service, version, stackVersion string) error {
	target := service
	if version != "" {
		target += " " + version
	}
	if stackVersion != "" {
		target += " on stack " + stackVersion
	}
	return fmt.Errorf("legacy plugin %s does not support %s; use the matching service plugin", Version, target)
}
