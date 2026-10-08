package pluginsdk

// InputSpec describes a terminal field without depending on a UI library.
// Exactly one binding (Flag, Argument, BodyPointer or Context) is required.
// Explicit arguments, flags and JSON values take precedence over the field.
type InputSpec struct {
	Title       string `json:"title"`
	Description string `json:"description,omitzero"`
	Kind        string `json:"kind"`
	Flag        string `json:"flag,omitzero"`
	Argument    *int   `json:"argument,omitzero"`
	BodyPointer string `json:"bodyPointer,omitzero"`
	Context     string `json:"context,omitzero"`
	ValueType   string `json:"valueType,omitzero"`
	Required    bool   `json:"required,omitzero"`
	Secret      bool   `json:"secret,omitzero"`
	Default     string `json:"default,omitzero"`
	// AlternativeArgument permits a positional name instead of a flag.
	AlternativeArgument *int          `json:"alternativeArgument,omitzero"`
	AlternativeFlag     string        `json:"alternativeFlag,omitzero"`
	Options             []InputOption `json:"options,omitzero"`
	Source              *ChoiceSource `json:"source,omitzero"`
}

type InputOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ChoiceSource invokes a read operation through the same plugin and client.
// A $name argument/flag refers to an already resolved request flag; $stack and
// $organization can also refer to host context. Labels and values are field
// names from each resource object. The host owns rendering and pagination.
type ChoiceSource struct {
	CommandPath  []string          `json:"commandPath"`
	Args         []string          `json:"args,omitzero"`
	Flags        map[string]string `json:"flags,omitzero"`
	ValueField   string            `json:"valueField"`
	LabelFields  []string          `json:"labelFields,omitzero"`
	EmptyMessage string            `json:"emptyMessage,omitzero"`
	// PreferredPrefix orders suggestions when the exact default is unavailable.
	// The user must still select a value explicitly.
	PreferredPrefix string `json:"preferredPrefix,omitzero"`
	// AfterField uses the final row's exact value for keyset pagination.
	// The host requests pages of 100 and follows --after until a shorter page.
	AfterField string `json:"afterField,omitzero"`
	// ExcludeTrueFields removes entries whose declared boolean field is true.
	ExcludeTrueFields []string `json:"excludeTrueFields,omitzero"`
	// MatchFields requires each field to match one of its exact scalar values.
	MatchFields map[string][]string `json:"matchFields,omitzero"`
}
