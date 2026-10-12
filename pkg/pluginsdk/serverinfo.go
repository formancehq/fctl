package pluginsdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ServiceVersion accepts the historical enveloped /_info response and the
// current direct response. Conflicting versions are rejected before execution.
func ServiceVersion(data json.RawMessage) (string, error) {
	var info struct {
		Version string `json:"version"`
		Data    struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return "", fmt.Errorf("decode service version: %w", err)
	}
	version := strings.TrimPrefix(info.Version, "v")
	nested := strings.TrimPrefix(info.Data.Version, "v")
	if version != "" && nested != "" && version != nested {
		return "", fmt.Errorf("service reports conflicting versions")
	}
	if version == "" {
		version = nested
	}
	if version == "" || strings.TrimSpace(version) != version {
		return "", fmt.Errorf("service /_info did not report a valid version")
	}
	return version, nil
}
