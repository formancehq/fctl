package pluginsdk

// FileSpec describes file IO performed by the host, never by the plugin.
// ReadArgument and ReadFlag are alternative, optional sources. A supplied
// source conflicts with an explicit body. "-" selects stdin or stdout.
// The host leaves arguments and flags intact and supplies decoded input in Body.
type FileSpec struct {
	ReadArgument *int   `json:"readArgument,omitzero"`
	ReadFlag     string `json:"readFlag,omitzero"`
	WriteFlag    string `json:"writeFlag,omitzero"`
	// ReadFormat is "string" (JSON-encoded raw text), "json" (the default),
	// or "yaml" (converted to JSON). Input is limited to 4 MiB.
	ReadFormat string `json:"readFormat,omitzero"`
	// WriteFormat is "json" (the default), "ndjson", or "yaml".
	// Without a destination, JSON uses the host's normal output renderer.
	WriteFormat string `json:"writeFormat,omitzero"`
	// FormatFlag optionally overrides WriteFormat using a declared string flag.
	// An empty flag value retains WriteFormat. It accepts json, ndjson, yaml,
	// or the historical yml alias for yaml.
	FormatFlag string `json:"formatFlag,omitzero"`
}
