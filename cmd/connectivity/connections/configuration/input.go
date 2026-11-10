package configuration

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type ConfigKind string

const (
	ConfigEnv  ConfigKind = "env"
	ConfigFile ConfigKind = "file"
)

type SchemaField struct {
	Key         string
	Kind        ConfigKind
	Required    bool
	Password    bool
	Description string
	// Enum lists the allowed values when the schema declares string enums.
	Enum []string
	// Type is the schema format, else its type, for display.
	Type string
	// Default is the schema default when HasDefault is true.
	Default    any
	HasDefault bool
}

type InputOptions struct {
	ConfigFile string
	SetValues  []string
	// Patch records supplied values, including explicit nulls; nil disables intent capture.
	Patch map[string]any
}

type ReadFileFunc func(cmd *cobra.Command, path string) (string, error)

func SchemaFields(version *connectivity.ConnectorVersion) (map[string]SchemaField, error) {
	contract, err := schemaContractFor(version)
	if err != nil {
		return nil, err
	}
	return contract.fields, nil
}

type schemaContract struct {
	fields map[string]SchemaField
	open   map[ConfigKind]bool
}

// schemaFieldInfo is display and completion metadata. The API remains the
// authority on validity; when the CLI cannot derive the exact allowed values
// it offers none rather than a wrong list.
type schemaFieldInfo struct {
	required    bool
	password    bool
	description string
	enum        []string
	// enumUnknown: the allowed values cannot be derived exactly.
	enumUnknown bool
	// never: no value satisfies the schema (false, or an empty intersection).
	never      bool
	format     string
	typ        string
	defaultSet bool
	defaultVal any
}

// schemaWalk carries the recursion guard and a per-reference cache, so a
// schema that references the same definitions many times is walked once per
// definition instead of once per path.
type schemaWalk struct {
	active   map[string]bool
	fields   map[string]schemaFieldInfo
	sections map[string]sectionContract
}

func newSchemaWalk() *schemaWalk {
	return &schemaWalk{active: map[string]bool{}, fields: map[string]schemaFieldInfo{}, sections: map[string]sectionContract{}}
}

func refKey(root map[string]any, ref string) string {
	return fmt.Sprintf("%p%s", root, ref)
}

// schemaRoot is the resource references resolve against: a node with $id
// starts a new resource.
func schemaRoot(node, root map[string]any) map[string]any {
	if id, _ := node["$id"].(string); id != "" {
		return node
	}
	return root
}

type sectionContract struct {
	fields map[string]schemaFieldInfo
	open   bool
	// never: the section accepts nothing (false), so it adds no alternative.
	never bool
	// wildcard: additionalProperties or patternProperties schemas also apply
	// to declared fields, so their allowed values are not derived.
	wildcard bool
}

func schemaContractFor(version *connectivity.ConnectorVersion) (schemaContract, error) {
	if version == nil {
		return schemaContract{}, fmt.Errorf("connector version is required")
	}
	if len(version.ConfigSchema) == 0 {
		// No declared constraint: route literal input and let the API validate.
		return schemaContract{fields: map[string]SchemaField{}, open: map[ConfigKind]bool{ConfigEnv: true, ConfigFile: true}}, nil
	}

	if _, flat := version.ConfigSchema["properties"]; flat {
		return schemaContract{}, fmt.Errorf("connector config schema must use env/files sections; flat schemas are unsupported")
	}
	contract := schemaContract{fields: make(map[string]SchemaField), open: make(map[ConfigKind]bool)}
	sections := []struct {
		name string
		kind ConfigKind
		raw  any
	}{
		{name: "env", kind: ConfigEnv, raw: version.ConfigSchema["env"]},
		{name: "files", kind: ConfigFile, raw: version.ConfigSchema["files"]},
	}

	for _, section := range sections {
		rawSection := section.raw
		if _, boolean := rawSection.(bool); rawSection == nil || boolean {
			// An absent or boolean section declares no field metadata; the
			// API remains the authority on what it accepts.
			contract.open[section.kind] = true
			continue
		}
		sectionSchema, ok := stringMap(rawSection)
		if !ok {
			return schemaContract{}, fmt.Errorf("connector config schema %s must be an object", section.name)
		}
		collected, err := collectSectionSchema(sectionSchema, sectionSchema, 0, newSchemaWalk())
		if err != nil {
			return schemaContract{}, fmt.Errorf("connector config schema %s: %w", section.name, err)
		}
		contract.open[section.kind] = collected.open
		for key, definition := range collected.fields {
			if _, exists := contract.fields[key]; exists {
				return schemaContract{}, fmt.Errorf("configuration key %q is declared in both env and files", key)
			}
			contract.fields[key] = SchemaField{
				Key:         key,
				Kind:        section.kind,
				Required:    definition.required,
				Password:    definition.password,
				Description: definition.description,
				Enum:        knownEnum(definition),
				Type:        displayType(definition),
				Default:     definition.defaultVal,
				HasDefault:  definition.defaultSet,
			}
		}
	}

	return contract, nil
}

const maxSchemaCollectionDepth = 64

func collectSectionSchema(raw any, root map[string]any, depth int, walk *schemaWalk) (sectionContract, error) {
	if depth > maxSchemaCollectionDepth {
		return sectionContract{}, fmt.Errorf("schema nesting exceeds %d", maxSchemaCollectionDepth)
	}
	if accepts, boolean := raw.(bool); boolean {
		// Neither carries field metadata; false contributes no alternative.
		return sectionContract{fields: map[string]schemaFieldInfo{}, open: accepts, never: !accepts}, nil
	}
	node, ok := stringMap(raw)
	if !ok {
		return sectionContract{}, fmt.Errorf("schema must be an object")
	}
	root = schemaRoot(node, root)
	result := sectionContract{fields: map[string]schemaFieldInfo{}, open: sectionOpen(node), wildcard: sectionWildcard(node)}
	properties, ok := stringMap(node["properties"])
	if !ok && node["properties"] != nil {
		return sectionContract{}, fmt.Errorf("properties must be an object")
	}
	required, err := requiredSet(node["required"])
	if err != nil {
		return sectionContract{}, err
	}
	for key, definition := range properties {
		metadata, err := collectFieldMetadata(definition, root, depth+1, walk)
		if err != nil {
			return sectionContract{}, fmt.Errorf("field %q: %w", key, err)
		}
		metadata.required = required[key]
		result.fields[key] = metadata
	}
	for key := range required {
		if _, found := result.fields[key]; !found {
			result.fields[key] = schemaFieldInfo{required: true}
		}
	}
	for _, ref := range schemaLocalReferences(node) {
		key := refKey(root, ref)
		if walk.active[key] {
			continue // The API accepts recursive schemas; metadata is best effort.
		}
		referenced, cached := walk.sections[key]
		if !cached {
			target, resolved := resolveLocalSchemaRef(root, ref)
			if !resolved {
				continue
			}
			walk.active[key] = true
			referenced, err = collectSectionSchema(target, root, depth+1, walk)
			delete(walk.active, key)
			if err != nil {
				return sectionContract{}, err
			}
			walk.sections[key] = referenced
		}
		result = mergeAllSections(result, referenced)
	}
	parts, err := schemaArray(node["allOf"])
	if err != nil {
		return sectionContract{}, fmt.Errorf("allOf: %w", err)
	}
	for _, part := range parts {
		collected, err := collectSectionSchema(part, root, depth+1, walk)
		if err != nil {
			return sectionContract{}, err
		}
		result = mergeAllSections(result, collected)
	}
	for _, keyword := range []string{"anyOf", "oneOf"} {
		parts, err := schemaArray(node[keyword])
		if err != nil {
			return sectionContract{}, fmt.Errorf("%s: %w", keyword, err)
		}
		if len(parts) == 0 {
			continue
		}
		alternatives := make([]sectionContract, 0, len(parts))
		for _, part := range parts {
			collected, err := collectSectionSchema(part, root, depth+1, walk)
			if err != nil {
				return sectionContract{}, err
			}
			alternatives = append(alternatives, collected)
		}
		result = mergeAllSections(result, mergeAlternativeSections(alternatives))
	}
	// Conditional schemas apply only sometimes: their fields are listed as
	// optional, and their allowed values are not derived.
	conditional := []sectionContract{{fields: map[string]schemaFieldInfo{}, open: true}}
	for _, keyword := range []string{"then", "else"} {
		if branch, present := node[keyword]; present {
			collected, err := collectSectionSchema(branch, root, depth+1, walk)
			if err != nil {
				return sectionContract{}, fmt.Errorf("%s: %w", keyword, err)
			}
			conditional = append(conditional, collected)
		}
	}
	if dependent, ok := stringMap(node["dependentSchemas"]); ok {
		for _, name := range sortedSchemaKeys(dependent) {
			collected, err := collectSectionSchema(dependent[name], root, depth+1, walk)
			if err != nil {
				return sectionContract{}, fmt.Errorf("dependentSchemas %q: %w", name, err)
			}
			conditional = append(conditional, collected)
		}
	}
	if len(conditional) > 1 {
		result = mergeAllSections(result, mergeAlternativeSections(conditional))
	}
	if result.wildcard {
		for key, field := range result.fields {
			field.enum, field.enumUnknown = nil, true
			result.fields[key] = field
		}
	}
	return result, nil
}

func schemaArray(raw any) ([]any, error) {
	if raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("must be an array")
	}
	return values, nil
}

// sectionWildcard reports schemas that constrain properties by name pattern
// or as "any other property", which also apply to declared fields.
func sectionWildcard(node map[string]any) bool {
	if _, schema := stringMap(node["additionalProperties"]); schema {
		return true
	}
	patterns, ok := stringMap(node["patternProperties"])
	return ok && len(patterns) > 0
}

func sortedSchemaKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sectionOpen(node map[string]any) bool {
	additional, present := node["additionalProperties"]
	open := !present || additional != false
	if patterns, ok := stringMap(node["patternProperties"]); ok && len(patterns) > 0 {
		return true
	}
	return open
}

func mergeAllSections(left, right sectionContract) sectionContract {
	merged := sectionContract{fields: make(map[string]schemaFieldInfo), open: left.open && right.open, never: left.never || right.never, wildcard: left.wildcard || right.wildcard}
	for key, value := range left.fields {
		merged.fields[key] = value
	}
	for key, value := range right.fields {
		current, exists := merged.fields[key]
		if !exists {
			merged.fields[key] = value
			continue
		}
		combined := mergeFieldMetadata(current, value, intersectEnums)
		combined.required = current.required || value.required
		merged.fields[key] = combined
	}
	if merged.wildcard {
		for key, field := range merged.fields {
			field.enum, field.enumUnknown = nil, true
			merged.fields[key] = field
		}
	}
	return merged
}

func mergeAlternativeSections(all []sectionContract) sectionContract {
	// A false alternative accepts nothing, so it neither adds fields nor makes
	// other alternatives' fields optional.
	alternatives := make([]sectionContract, 0, len(all))
	for _, alternative := range all {
		if !alternative.never {
			alternatives = append(alternatives, alternative)
		}
	}
	if len(alternatives) == 0 {
		return sectionContract{fields: map[string]schemaFieldInfo{}, never: true}
	}
	merged := sectionContract{fields: make(map[string]schemaFieldInfo)}
	for index, alternative := range alternatives {
		merged.open = merged.open || alternative.open
		merged.wildcard = merged.wildcard || alternative.wildcard
		for key, value := range alternative.fields {
			current, exists := merged.fields[key]
			if !exists {
				value.required = index == 0 && value.required
				merged.fields[key] = value
				continue
			}
			combined := mergeFieldMetadata(current, value, unionEnums)
			combined.required = current.required && value.required
			merged.fields[key] = combined
		}
		if index > 0 {
			for key, value := range merged.fields {
				if _, exists := alternative.fields[key]; !exists {
					value.required = false
					merged.fields[key] = value
				}
			}
		}
	}
	// An open alternative that does not declare a field accepts any value for
	// it, so no enum from the other alternatives is exact.
	for key, value := range merged.fields {
		for _, alternative := range alternatives {
			if _, declared := alternative.fields[key]; !declared && alternative.open {
				value.enum, value.enumUnknown = nil, true
				merged.fields[key] = value
				break
			}
		}
	}
	return merged
}

func collectFieldMetadata(raw any, root map[string]any, depth int, walk *schemaWalk) (schemaFieldInfo, error) {
	if depth > maxSchemaCollectionDepth {
		return schemaFieldInfo{}, fmt.Errorf("schema nesting exceeds %d", maxSchemaCollectionDepth)
	}
	if accepts, boolean := raw.(bool); boolean {
		return schemaFieldInfo{never: !accepts}, nil
	}
	node, ok := stringMap(raw)
	if !ok {
		return schemaFieldInfo{}, nil // Other non-object schemas are delegated to the API.
	}
	root = schemaRoot(node, root)
	result := schemaFieldInfo{}
	result.format, _ = node["format"].(string)
	result.password = result.format == "password" || node["writeOnly"] == true
	result.typ = schemaType(node["type"])
	result.description, _ = node["description"].(string)
	result.defaultVal, result.defaultSet = node["default"]
	result.enum, result.enumUnknown = nodeEnum(node)
	for _, ref := range schemaLocalReferences(node) {
		key := refKey(root, ref)
		if walk.active[key] {
			continue // Recursive reference: leave the remaining validation to the API.
		}
		resolved, cached := walk.fields[key]
		if !cached {
			target, found := resolveLocalSchemaRef(root, ref)
			if !found {
				continue
			}
			walk.active[key] = true
			var err error
			resolved, err = collectFieldMetadata(target, root, depth+1, walk)
			delete(walk.active, key)
			if err != nil {
				return schemaFieldInfo{}, err
			}
			walk.fields[key] = resolved
		}
		// A reference applies its constraints in addition to the node's.
		result = mergeFieldMetadata(result, resolved, intersectEnums)
	}
	allOf, err := schemaArray(node["allOf"])
	if err != nil {
		return schemaFieldInfo{}, fmt.Errorf("allOf: %w", err)
	}
	for _, part := range allOf {
		metadata, err := collectFieldMetadata(part, root, depth+1, walk)
		if err != nil {
			return schemaFieldInfo{}, err
		}
		result = mergeFieldMetadata(result, metadata, intersectEnums)
	}
	for _, keyword := range []string{"anyOf", "oneOf"} {
		parts, err := schemaArray(node[keyword])
		if err != nil {
			return schemaFieldInfo{}, fmt.Errorf("%s: %w", keyword, err)
		}
		if len(parts) == 0 {
			continue
		}
		alternatives := make([]schemaFieldInfo, 0, len(parts))
		for _, part := range parts {
			metadata, err := collectFieldMetadata(part, root, depth+1, walk)
			if err != nil {
				return schemaFieldInfo{}, err
			}
			alternatives = append(alternatives, metadata)
		}
		combined := combineAlternatives(alternatives, keyword == "oneOf")
		result = mergeFieldMetadata(result, combined, intersectEnums)
	}
	// not, if/then/else and string length or pattern constraints can exclude
	// listed values in ways the CLI does not evaluate.
	if len(result.enum) > 0 && hasAny(node, "not", "if", "then", "else", "pattern", "minLength", "maxLength") {
		result.enum, result.enumUnknown = nil, true
	}
	return result, nil
}

func hasAny(node map[string]any, keywords ...string) bool {
	for _, keyword := range keywords {
		if _, present := node[keyword]; present {
			return true
		}
	}
	return false
}

// schemaType renders a type or a type array, for example "string | null".
func schemaType(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []any:
		names := make([]string, 0, len(value))
		for _, item := range value {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
		return strings.Join(names, " | ")
	}
	return ""
}

// nodeEnum reads enum and const. A const narrows the enum to one value; a
// non-string enum member or const cannot be completed and makes the allowed
// values unknown rather than unconstrained.
func nodeEnum(node map[string]any) ([]string, bool) {
	var enum []string
	if raw, present := node["enum"]; present {
		values, exact := stringEnum(raw)
		if !exact {
			return nil, true
		}
		enum = values
	}
	if raw, present := node["const"]; present {
		value, ok := raw.(string)
		if !ok {
			return nil, true
		}
		if enum == nil {
			return []string{value}, false
		}
		if !slices.Contains(enum, value) {
			return nil, true
		}
		return []string{value}, false
	}
	return enum, false
}

// combineAlternatives folds anyOf/oneOf alternatives. A false alternative
// matches nothing and is skipped. For oneOf, a value listed by two
// alternatives matches both and is therefore rejected.
func combineAlternatives(alternatives []schemaFieldInfo, exclusive bool) schemaFieldInfo {
	var combined schemaFieldInfo
	started := false
	counts := map[string]int{}
	for _, alternative := range alternatives {
		if alternative.never {
			continue
		}
		for _, value := range alternative.enum {
			counts[value]++
		}
		if !started {
			combined, started = alternative, true
			continue
		}
		combined = mergeFieldMetadata(combined, alternative, unionEnums)
	}
	if !started {
		return schemaFieldInfo{never: true}
	}
	if exclusive && !combined.enumUnknown && len(combined.enum) > 0 {
		unique := make([]string, 0, len(combined.enum))
		for _, value := range combined.enum {
			if counts[value] == 1 {
				unique = append(unique, value)
			}
		}
		combined.enum = unique
		if len(unique) == 0 {
			combined.enum, combined.never = nil, true
		}
	}
	return combined
}

// mergeFieldMetadata combines two descriptions of the same field; combineEnums
// states whether both apply (intersection) or either may (union).
func mergeFieldMetadata(left, right schemaFieldInfo, combineEnums func(schemaFieldInfo, schemaFieldInfo) ([]string, bool, bool)) schemaFieldInfo {
	merged := left
	merged.password = left.password || right.password
	merged.description = firstDescription(left.description, right.description)
	merged.enum, merged.enumUnknown, merged.never = combineEnums(left, right)
	return withDisplay(merged, left, right)
}

func schemaLocalReferences(node map[string]any) []string {
	refs := make([]string, 0, 3)
	for _, keyword := range []string{"$ref", "$dynamicRef", "$recursiveRef"} {
		if ref, ok := node[keyword].(string); ok && strings.HasPrefix(ref, "#") {
			refs = append(refs, ref)
		}
	}
	return refs
}

// resolveLocalSchemaRef follows JSON Pointer and named local anchors. It has
// the same deliberately tolerant contract as the API's schema metadata walk:
// unresolvable or external references are deferred to server-side validation.
func resolveLocalSchemaRef(root map[string]any, ref string) (any, bool) {
	if !strings.HasPrefix(ref, "#") {
		return nil, false
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(ref, "#"))
	if err != nil {
		return nil, false
	}
	if fragment == "" {
		return root, true
	}
	if strings.HasPrefix(fragment, "/") {
		var current any = root
		for _, token := range strings.Split(strings.TrimPrefix(fragment, "/"), "/") {
			token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
			switch container := current.(type) {
			case map[string]any:
				next, ok := container[token]
				if !ok {
					return nil, false
				}
				current = next
			case []any:
				index, err := strconv.Atoi(token)
				if err != nil || index < 0 || index >= len(container) || strconv.Itoa(index) != token {
					return nil, false
				}
				current = container[index]
			default:
				return nil, false
			}
		}
		return current, true
	}
	return findSchemaAnchor(root, fragment)
}

// annotationKeyword lists keywords whose values are instance data, never
// subschemas.
var annotationKeyword = map[string]bool{"default": true, "examples": true, "const": true, "enum": true}

func findSchemaAnchor(root map[string]any, anchor string) (any, bool) {
	remaining := 100000
	var find func(any, int) (any, bool)
	find = func(node any, depth int) (any, bool) {
		if depth > maxSchemaCollectionDepth || remaining == 0 {
			return nil, false
		}
		remaining--
		switch node := node.(type) {
		case map[string]any:
			if id, _ := node["$id"].(string); id != "" && depth > 0 {
				return nil, false // A nested resource has its own anchors.
			}
			for _, keyword := range []string{"$anchor", "$dynamicAnchor"} {
				if declared, _ := node[keyword].(string); declared == anchor {
					return node, true
				}
			}
			for _, key := range sortedSchemaKeys(node) {
				if annotationKeyword[key] {
					continue // Data, not subschemas: anchors there are not schema anchors.
				}
				if target, found := find(node[key], depth+1); found {
					return target, true
				}
			}
		case []any:
			for _, child := range node {
				if target, found := find(child, depth+1); found {
					return target, true
				}
			}
		}
		return nil, false
	}
	return find(root, 0)
}

// stringEnum returns the enum values and whether they are all strings; a
// mixed or non-string enum cannot be completed.
func stringEnum(raw any) ([]string, bool) {
	values, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}

// withDisplay keeps the first declared format, type and default across merged
// metadata; they are informational and never constrain input. Format and type
// are kept apart so a format found anywhere wins over a plain type.
func withDisplay(target, left, right schemaFieldInfo) schemaFieldInfo {
	target.format = firstDescription(left.format, right.format)
	target.typ = firstDescription(left.typ, right.typ)
	target.defaultVal, target.defaultSet = left.defaultVal, left.defaultSet
	if !target.defaultSet {
		target.defaultVal, target.defaultSet = right.defaultVal, right.defaultSet
	}
	return target
}

func displayType(info schemaFieldInfo) string {
	if info.format != "" {
		return info.format
	}
	return info.typ
}

func knownEnum(info schemaFieldInfo) []string {
	if info.enumUnknown || info.never {
		return nil
	}
	return info.enum
}

// intersectEnums combines constraints that all apply. An empty intersection
// means no value is valid.
func intersectEnums(left, right schemaFieldInfo) ([]string, bool, bool) {
	if left.never || right.never {
		return nil, false, true
	}
	if left.enumUnknown || right.enumUnknown {
		return nil, true, false
	}
	if len(left.enum) == 0 {
		return right.enum, false, false
	}
	if len(right.enum) == 0 {
		return left.enum, false, false
	}
	allowed := make(map[string]bool, len(right.enum))
	for _, value := range right.enum {
		allowed[value] = true
	}
	result := make([]string, 0, len(left.enum))
	for _, value := range left.enum {
		if allowed[value] {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil, false, true
	}
	return result, false, false
}

// unionEnums combines alternatives: a value may satisfy either. An
// alternative without an enum accepts any value, so no list is exact; an
// alternative that accepts nothing adds nothing.
func unionEnums(left, right schemaFieldInfo) ([]string, bool, bool) {
	switch {
	case left.never && right.never:
		return nil, false, true
	case left.never:
		return right.enum, right.enumUnknown, false
	case right.never:
		return left.enum, left.enumUnknown, false
	}
	if left.enumUnknown || right.enumUnknown || len(left.enum) == 0 || len(right.enum) == 0 {
		return nil, left.enumUnknown || right.enumUnknown || len(left.enum) > 0 || len(right.enum) > 0, false
	}
	seen := make(map[string]bool, len(left.enum)+len(right.enum))
	result := make([]string, 0, len(left.enum)+len(right.enum))
	for _, value := range append(append([]string{}, left.enum...), right.enum...) {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result, false, false
}

func firstDescription(left, right string) string {
	if left != "" {
		return left
	}
	return right
}

func BuildInstallConfig(cmd *cobra.Command, version *connectivity.ConnectorVersion, inputs InputOptions, read ReadFileFunc) (*connectivity.ConnectionConfiguration, error) {
	return buildConfig(cmd, version, nil, inputs, read, true)
}

func BuildConfigureConfig(cmd *cobra.Command, version *connectivity.ConnectorVersion, current *connectivity.ConnectionConfiguration, inputs InputOptions, read ReadFileFunc) (*connectivity.ConnectionConfiguration, error) {
	return buildConfig(cmd, version, current, inputs, read, false)
}

func buildConfig(cmd *cobra.Command, version *connectivity.ConnectorVersion, base *connectivity.ConnectionConfiguration, inputs InputOptions, read ReadFileFunc, install bool) (*connectivity.ConnectionConfiguration, error) {
	contract, err := schemaContractFor(version)
	if err != nil {
		return nil, err
	}
	fields := contract.fields
	config := cloneConfig(base)
	unknown := make(map[string]struct{})

	if inputs.ConfigFile != "" {
		contents, err := readInput(cmd, read, inputs.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read config %q", inputs.ConfigFile)
		}
		if err := applyConfigDocument(config, contract, contents, unknown, inputs.Patch); err != nil {
			return nil, fmt.Errorf("parsing config %q: %w", inputs.ConfigFile, err)
		}
	}

	for _, assignment := range inputs.SetValues {
		key, value, err := parseAssignment(assignment)
		if err != nil {
			return nil, err
		}
		field, exists := assignmentField(contract, key)
		if !exists {
			unknown[key] = struct{}{}
			continue
		}
		parsed, err := parseSetValue(cmd, read, value)
		if err != nil {
			return nil, fmt.Errorf("configuration key %q: %w", key, err)
		}
		applyValue(config, field, parsed)
		if err := recordValue(inputs.Patch, field, parsed); err != nil {
			return nil, err
		}
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown configuration keys: %s", strings.Join(sortedKeys(unknown), ", "))
	}

	missing := make(map[string]struct{})
	for key, field := range fields {
		if install && field.Required && !hasValue(config, field) {
			missing[key] = struct{}{}
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required configuration keys: %s", strings.Join(sortedKeys(missing), ", "))
	}

	return config, nil
}

func literalValue(raw string) (string, error) {
	if strings.HasPrefix(raw, "secret://") || strings.HasPrefix(raw, "configmap://") {
		return "", fmt.Errorf("Kubernetes references are unsupported; supply a literal value or use @file / @- with --set")
	}
	return raw, nil
}

func parseSetValue(cmd *cobra.Command, read ReadFileFunc, raw string) (string, error) {
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimPrefix(raw, "@")
		if path == "" {
			return "", fmt.Errorf("file value path cannot be empty")
		}
		contents, err := readInput(cmd, read, path)
		if err != nil {
			return "", fmt.Errorf("cannot read configuration value from file")
		}
		return contents, nil
	}
	return literalValue(raw)
}

// Intent is recorded independently of GET: confidential fields may be omitted
// from reads, so a diff against readback would lose an explicit null deletion.
func recordValue(patch map[string]any, field SchemaField, value any) error {
	if patch == nil {
		return nil
	}
	section := "env"
	if field.Kind == ConfigFile {
		section = "files"
	}
	if existing, present := patch[section]; present && existing == nil {
		return fmt.Errorf("cannot combine clearing %s with setting its entries; use separate configure requests", section)
	}
	entries, ok := patch[section].(map[string]any)
	if !ok {
		entries = map[string]any{}
		patch[section] = entries
	}
	entries[field.Key] = value
	return nil
}

func parseAssignment(raw string) (string, string, error) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid --set: expected KEY=value")
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", fmt.Errorf("invalid --set: empty key")
	}
	return key, parts[1], nil
}

func applyDoubleQuoteEscapes(value string) string {
	var decoded strings.Builder
	decoded.Grow(len(value))
	for index := 0; index < len(value); index++ {
		if value[index] == '\\' && index+1 < len(value) {
			index++
			switch value[index] {
			case 'n':
				decoded.WriteByte('\n')
			case 't':
				decoded.WriteByte('\t')
			case 'r':
				decoded.WriteByte('\r')
			case '\\':
				decoded.WriteByte('\\')
			case '"':
				decoded.WriteByte('"')
			default:
				// Unknown escapes stay literal so credentials and paths are
				// sent exactly as written.
				decoded.WriteByte('\\')
				decoded.WriteByte(value[index])
			}
			continue
		}
		decoded.WriteByte(value[index])
	}
	return decoded.String()
}

func applyConfigDocument(config *connectivity.ConnectionConfiguration, contract schemaContract, contents string, unknown map[string]struct{}, patch map[string]any) error {
	var root map[string]any
	if err := yaml.Unmarshal([]byte(contents), &root); err != nil {
		return configDecodeError(err)
	}
	for section, raw := range root {
		kind := ConfigEnv
		switch section {
		case "env":
		case "files":
			kind = ConfigFile
		default:
			unknown[section] = struct{}{}
			continue
		}
		if raw == nil {
			if kind == ConfigEnv {
				config.Env = nil
			} else {
				config.Files = nil
			}
			if patch != nil {
				patch[section] = nil
			}
			continue
		}
		values, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid YAML/JSON configuration: env and files must be mappings of literal strings or null")
		}
		for key, rawValue := range values {
			field, exists := configField(contract, key, kind)
			if !exists {
				unknown[key] = struct{}{}
				continue
			}
			if rawValue == nil {
				if kind == ConfigEnv {
					delete(config.Env, key)
				} else {
					delete(config.Files, key)
				}
				if err := recordValue(patch, field, nil); err != nil {
					return err
				}
				continue
			}
			value, ok := rawValue.(string)
			if !ok {
				return fmt.Errorf("configuration key %q: expected a literal string or null; Kubernetes reference objects are unsupported", key)
			}
			parsed, err := literalValue(value)
			if err != nil {
				return fmt.Errorf("configuration key %q: %w", key, err)
			}
			applyValue(config, field, parsed)
			if err := recordValue(patch, field, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func configDecodeError(err error) error {
	// Decoder messages can contain scalar values, keys, tags, and aliases from
	// the input. Only retain numeric locations; do not wrap the original error.
	messages := []string{strings.TrimPrefix(err.Error(), "yaml: ")}
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		messages = typeErr.Errors
	}
	lineSet := make(map[int]struct{})
	for _, message := range messages {
		var line int
		if _, scanErr := fmt.Sscanf(message, "line %d:", &line); scanErr == nil && line > 0 {
			lineSet[line] = struct{}{}
		}
	}
	lines := make([]int, 0, len(lineSet))
	for line := range lineSet {
		lines = append(lines, line)
	}
	sort.Ints(lines)
	locations := make([]string, 0, len(lines))
	for _, line := range lines {
		locations = append(locations, fmt.Sprint(line))
	}
	location := ""
	if len(locations) > 0 {
		location = " at lines " + strings.Join(locations, ", ")
	}
	return fmt.Errorf("invalid YAML/JSON configuration%s: expected env and files mappings of literal strings or null", location)
}

func configField(contract schemaContract, key string, kind ConfigKind) (SchemaField, bool) {
	if field, exists := contract.fields[key]; exists {
		return field, field.Kind == kind
	}
	if contract.open[kind] {
		return SchemaField{Key: key, Kind: kind}, true
	}
	return SchemaField{}, false
}

func assignmentField(contract schemaContract, key string) (SchemaField, bool) {
	if field, exists := contract.fields[key]; exists {
		return field, true
	}
	kind := ConfigEnv
	if strings.HasPrefix(key, "/") {
		kind = ConfigFile
	}
	return configField(contract, key, kind)
}

func applyValue(config *connectivity.ConnectionConfiguration, field SchemaField, value string) {
	if field.Kind == ConfigEnv {
		if config.Env == nil {
			config.Env = map[string]string{}
		}
		config.Env[field.Key] = value
	} else {
		if config.Files == nil {
			config.Files = map[string]string{}
		}
		config.Files[field.Key] = value
	}
}

func hasValue(config *connectivity.ConnectionConfiguration, field SchemaField) bool {
	if field.Kind == ConfigEnv {
		_, exists := config.Env[field.Key]
		return exists
	}
	_, exists := config.Files[field.Key]
	return exists
}

func cloneConfig(source *connectivity.ConnectionConfiguration) *connectivity.ConnectionConfiguration {
	clone := &connectivity.ConnectionConfiguration{Env: map[string]string{}, Files: map[string]string{}}
	if source != nil {
		maps.Copy(clone.Env, source.Env)
		maps.Copy(clone.Files, source.Files)
	}
	return clone
}

func readInput(cmd *cobra.Command, read ReadFileFunc, path string) (string, error) {
	if read == nil {
		return "", fmt.Errorf("file reader is required")
	}
	return read(cmd, path)
}

func requiredSet(raw any) (map[string]bool, error) {
	set := make(map[string]bool)
	if raw == nil {
		return set, nil
	}
	switch values := raw.(type) {
	case []any:
		for _, rawValue := range values {
			value, ok := rawValue.(string)
			if !ok {
				return nil, fmt.Errorf("required entries must be strings")
			}
			set[value] = true
		}
	case []string:
		for _, value := range values {
			set[value] = true
		}
	default:
		return nil, fmt.Errorf("required must be an array")
	}
	return set, nil
}

func stringMap(raw any) (map[string]any, bool) {
	value, ok := raw.(map[string]any)
	return value, ok
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
