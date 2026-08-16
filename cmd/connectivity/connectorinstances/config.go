package connectorinstances

import (
	"bufio"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
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
}

type InputOptions struct {
	ConfigFile string
	EnvFiles   []string
	SetValues  []string
}

type ReadFileFunc func(cmd *cobra.Command, path string) (string, error)

func SchemaFields(version *connectivityclient.ConnectorVersion) (map[string]SchemaField, error) {
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

type schemaFieldInfo struct {
	required    bool
	password    bool
	description string
}

type sectionContract struct {
	fields map[string]schemaFieldInfo
	open   bool
}

func schemaContractFor(version *connectivityclient.ConnectorVersion) (schemaContract, error) {
	if version == nil {
		return schemaContract{}, fmt.Errorf("connector version is required")
	}
	if len(version.ConfigSchema) == 0 {
		return schemaContract{fields: map[string]SchemaField{}, open: map[ConfigKind]bool{}}, nil
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
	if sections[0].raw == nil && sections[1].raw == nil {
		if _, hasLegacyProperties := version.ConfigSchema["properties"]; hasLegacyProperties {
			sections[0].raw = version.ConfigSchema
		}
	}

	for _, section := range sections {
		rawSection := section.raw
		exists := rawSection != nil
		if !exists {
			continue
		}
		sectionSchema, ok := stringMap(rawSection)
		if !ok {
			return schemaContract{}, fmt.Errorf("connector config schema %s must be an object", section.name)
		}
		collected, err := collectSectionSchema(sectionSchema, sectionSchema, 0, map[string]bool{})
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
			}
		}
	}

	return contract, nil
}

const maxSchemaCollectionDepth = 64

func collectSectionSchema(raw any, root map[string]any, depth int, refs map[string]bool) (sectionContract, error) {
	if depth > maxSchemaCollectionDepth {
		return sectionContract{}, fmt.Errorf("schema nesting exceeds %d", maxSchemaCollectionDepth)
	}
	node, ok := stringMap(raw)
	if !ok {
		return sectionContract{}, fmt.Errorf("schema must be an object")
	}
	result := sectionContract{fields: map[string]schemaFieldInfo{}, open: sectionOpen(node)}
	properties, ok := stringMap(node["properties"])
	if !ok && node["properties"] != nil {
		return sectionContract{}, fmt.Errorf("properties must be an object")
	}
	required, err := requiredSet(node["required"])
	if err != nil {
		return sectionContract{}, err
	}
	for key, definition := range properties {
		metadata, err := collectFieldMetadata(definition, root, depth+1, refs)
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
		if refs[ref] {
			continue // The API accepts recursive schemas; metadata is best effort.
		}
		if target, resolved := resolveLocalSchemaRef(root, ref); resolved {
			refs[ref] = true
			referenced, err := collectSectionSchema(target, root, depth+1, refs)
			delete(refs, ref)
			if err != nil {
				return sectionContract{}, err
			}
			result = mergeAllSections(result, referenced)
		}
	}
	for _, keyword := range []string{"allOf"} {
		parts, err := schemaArray(node[keyword])
		if err != nil {
			return sectionContract{}, fmt.Errorf("%s: %w", keyword, err)
		}
		for _, part := range parts {
			collected, err := collectSectionSchema(part, root, depth+1, refs)
			if err != nil {
				return sectionContract{}, err
			}
			result = mergeAllSections(result, collected)
		}
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
			collected, err := collectSectionSchema(part, root, depth+1, refs)
			if err != nil {
				return sectionContract{}, err
			}
			alternatives = append(alternatives, collected)
		}
		result = mergeAllSections(result, mergeAlternativeSections(alternatives))
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

func sectionOpen(node map[string]any) bool {
	additional, present := node["additionalProperties"]
	open := !present || additional != false
	if patterns, ok := stringMap(node["patternProperties"]); ok && len(patterns) > 0 {
		return true
	}
	return open
}

func mergeAllSections(left, right sectionContract) sectionContract {
	merged := sectionContract{fields: make(map[string]schemaFieldInfo), open: left.open && right.open}
	for key, value := range left.fields {
		merged.fields[key] = value
	}
	for key, value := range right.fields {
		current, exists := merged.fields[key]
		if !exists {
			merged.fields[key] = value
			continue
		}
		merged.fields[key] = schemaFieldInfo{required: current.required || value.required, password: current.password || value.password, description: firstDescription(current.description, value.description)}
	}
	return merged
}

func mergeAlternativeSections(alternatives []sectionContract) sectionContract {
	merged := sectionContract{fields: make(map[string]schemaFieldInfo)}
	for index, alternative := range alternatives {
		merged.open = merged.open || alternative.open
		for key, value := range alternative.fields {
			current, exists := merged.fields[key]
			if !exists {
				value.required = index == 0 && value.required
				merged.fields[key] = value
				continue
			}
			current.required = current.required && value.required
			current.password = current.password || value.password
			current.description = firstDescription(current.description, value.description)
			merged.fields[key] = current
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
	return merged
}

func collectFieldMetadata(raw any, root map[string]any, depth int, refs map[string]bool) (schemaFieldInfo, error) {
	if depth > maxSchemaCollectionDepth {
		return schemaFieldInfo{}, fmt.Errorf("schema nesting exceeds %d", maxSchemaCollectionDepth)
	}
	node, ok := stringMap(raw)
	if !ok {
		return schemaFieldInfo{}, nil // Boolean schemas are delegated to the API.
	}
	result := schemaFieldInfo{}
	if format, _ := node["format"].(string); format == "password" {
		result.password = true
	}
	if secret, _ := node["x-secret"].(bool); secret {
		result.password = true
	}
	result.description, _ = node["description"].(string)
	for _, ref := range schemaLocalReferences(node) {
		if refs[ref] {
			continue // Recursive reference: leave the remaining validation to the API.
		}
		if target, found := resolveLocalSchemaRef(root, ref); found {
			refs[ref] = true
			resolved, err := collectFieldMetadata(target, root, depth+1, refs)
			delete(refs, ref)
			if err != nil {
				return schemaFieldInfo{}, err
			}
			result.password = result.password || resolved.password
			result.description = firstDescription(result.description, resolved.description)
		}
	}
	for _, part := range appendSchemaCompositions(node) {
		metadata, err := collectFieldMetadata(part, root, depth+1, refs)
		if err != nil {
			return schemaFieldInfo{}, err
		}
		result.password = result.password || metadata.password
		result.description = firstDescription(result.description, metadata.description)
	}
	return result, nil
}

func appendSchemaCompositions(node map[string]any) []any {
	parts := make([]any, 0)
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		if values, ok := node[keyword].([]any); ok {
			parts = append(parts, values...)
		}
	}
	return parts
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
			object, ok := stringMap(current)
			if !ok {
				return nil, false
			}
			current, ok = object[strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")]
			if !ok {
				return nil, false
			}
		}
		return current, true
	}
	return findSchemaAnchor(root, fragment)
}

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
			for _, keyword := range []string{"$anchor", "$dynamicAnchor"} {
				if declared, _ := node[keyword].(string); declared == anchor {
					return node, true
				}
			}
			keys := make([]string, 0, len(node))
			for key := range node {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
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

func firstDescription(left, right string) string {
	if left != "" {
		return left
	}
	return right
}

func BuildInstallConfig(cmd *cobra.Command, version *connectivityclient.ConnectorVersion, inputs InputOptions, read ReadFileFunc) (*connectivityclient.ConnectorInstanceConfig, error) {
	return buildConfig(cmd, version, nil, inputs, read)
}

func BuildConfigureConfig(cmd *cobra.Command, version *connectivityclient.ConnectorVersion, current *connectivityclient.ConnectorInstanceConfig, inputs InputOptions, read ReadFileFunc) (*connectivityclient.ConnectorInstanceConfig, error) {
	return buildConfig(cmd, version, current, inputs, read)
}

func buildConfig(cmd *cobra.Command, version *connectivityclient.ConnectorVersion, base *connectivityclient.ConnectorInstanceConfig, inputs InputOptions, read ReadFileFunc) (*connectivityclient.ConnectorInstanceConfig, error) {
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
			return nil, fmt.Errorf("reading config %q: %w", inputs.ConfigFile, err)
		}
		if err := applyConfigDocument(config, contract, contents, unknown); err != nil {
			return nil, fmt.Errorf("parsing config %q: %w", inputs.ConfigFile, err)
		}
	}

	for _, path := range inputs.EnvFiles {
		contents, err := readInput(cmd, read, path)
		if err != nil {
			return nil, fmt.Errorf("reading env file %q: %w", path, err)
		}
		if err := applyDotenv(config, contract, contents, unknown); err != nil {
			return nil, fmt.Errorf("parsing env file %s:%w", path, err)
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
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown configuration keys: %s", strings.Join(sortedKeys(unknown), ", "))
	}

	missing := make(map[string]struct{})
	for key, field := range fields {
		if field.Required && !hasValue(config, field) {
			missing[key] = struct{}{}
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required configuration keys: %s", strings.Join(sortedKeys(missing), ", "))
	}

	return config, nil
}

type parsedValue struct {
	value        *string
	secretRef    *connectivityclient.KeyRef
	configMapRef *connectivityclient.KeyRef
}

type configDocument struct {
	Env   map[string]documentValue `yaml:"env"`
	Files []documentFile           `yaml:"files"`
}

type documentValue struct {
	Value        *string      `yaml:"value"`
	SecretRef    *documentRef `yaml:"secretRef"`
	ConfigMapRef *documentRef `yaml:"configMapRef"`
}

type documentRef struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

type documentFile struct {
	Path         string       `yaml:"path"`
	Value        *string      `yaml:"value"`
	SecretRef    *documentRef `yaml:"secretRef"`
	ConfigMapRef *documentRef `yaml:"configMapRef"`
	Mode         *int32       `yaml:"mode"`
}

func parseSetValue(cmd *cobra.Command, read ReadFileFunc, raw string) (parsedValue, error) {
	if strings.HasPrefix(raw, "secret://") {
		ref, err := parseReference(strings.TrimPrefix(raw, "secret://"), "secret")
		return parsedValue{secretRef: ref}, err
	}
	if strings.HasPrefix(raw, "configmap://") {
		ref, err := parseReference(strings.TrimPrefix(raw, "configmap://"), "configmap")
		return parsedValue{configMapRef: ref}, err
	}
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimPrefix(raw, "@")
		if path == "" {
			return parsedValue{}, fmt.Errorf("file value path cannot be empty")
		}
		contents, err := readInput(cmd, read, path)
		if err != nil {
			return parsedValue{}, fmt.Errorf("reading value from %q: %w", path, err)
		}
		return parsedValue{value: stringPointer(contents)}, nil
	}
	return parsedValue{value: stringPointer(raw)}, nil
}

func parseReference(raw, kind string) (*connectivityclient.KeyRef, error) {
	if strings.Count(raw, "/") != 1 {
		return nil, fmt.Errorf("malformed %s reference: expected %s://name/key", kind, kind)
	}
	parts := strings.SplitN(raw, "/", 2)
	if parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("malformed %s reference: expected %s://name/key", kind, kind)
	}
	return &connectivityclient.KeyRef{Name: parts[0], Key: parts[1]}, nil
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

func applyDotenv(config *connectivityclient.ConnectorInstanceConfig, contract schemaContract, contents string, unknown map[string]struct{}) error {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		raw = strings.TrimPrefix(raw, "export ")
		equals := strings.IndexByte(raw, '=')
		if equals <= 0 {
			return fmt.Errorf("%d: expected KEY=value", line)
		}
		key := strings.TrimSpace(raw[:equals])
		if key == "" {
			return fmt.Errorf("%d: expected KEY=value", line)
		}
		value := unquoteEnvValue(raw[equals+1:])
		field, exists := configField(contract, key, ConfigEnv)
		if !exists {
			unknown[key] = struct{}{}
			continue
		}
		applyValue(config, field, parsedValue{value: stringPointer(value)})
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func unquoteEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if comment := dotenvCommentIndex(value); comment >= 0 {
		value = strings.TrimSpace(value[:comment])
	}
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			return applyDoubleQuoteEscapes(value[1 : len(value)-1])
		}
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func dotenvCommentIndex(value string) int {
	var quote byte
	for index := 0; index < len(value); index++ {
		character := value[index]
		if quote != 0 {
			if quote == '"' && character == '\\' && index+1 < len(value) {
				index++
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '"' || character == '\'' {
			quote = character
			continue
		}
		if character == '#' && (index == 0 || value[index-1] == ' ' || value[index-1] == '\t') {
			return index
		}
	}
	return -1
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
				decoded.WriteByte(value[index])
			}
			continue
		}
		decoded.WriteByte(value[index])
	}
	return decoded.String()
}

func applyConfigDocument(config *connectivityclient.ConnectorInstanceConfig, contract schemaContract, contents string, unknown map[string]struct{}) error {
	var root map[string]any
	if err := yaml.Unmarshal([]byte(contents), &root); err != nil {
		return err
	}
	if root == nil {
		return nil
	}

	_, hasEnv := root["env"]
	_, hasFiles := root["files"]
	if !hasEnv && !hasFiles {
		for key, raw := range root {
			field, exists := configField(contract, key, ConfigEnv)
			if !exists {
				unknown[key] = struct{}{}
				continue
			}
			value, err := scalarString(raw)
			if err != nil {
				return fmt.Errorf("configuration key %q: %w", key, err)
			}
			applyValue(config, field, parsedValue{value: stringPointer(value)})
		}
		return nil
	}

	for key := range root {
		if key != "env" && key != "files" {
			unknown[key] = struct{}{}
		}
	}
	var document configDocument
	if err := yaml.Unmarshal([]byte(contents), &document); err != nil {
		return err
	}
	for key, value := range document.Env {
		_, exists := configField(contract, key, ConfigEnv)
		if !exists {
			unknown[key] = struct{}{}
			continue
		}
		converted := connectivityclient.EnvValue{
			Value:        cloneString(value.Value),
			SecretRef:    value.SecretRef.clientRef(),
			ConfigMapRef: value.ConfigMapRef.clientRef(),
		}
		if err := validateSources(converted.Value, converted.SecretRef, converted.ConfigMapRef); err != nil {
			return fmt.Errorf("configuration key %q: %w", key, err)
		}
		config.Env[key] = converted
	}
	for _, file := range document.Files {
		if strings.TrimSpace(file.Path) == "" {
			return fmt.Errorf("structured configuration file path is required")
		}
		_, exists := configField(contract, file.Path, ConfigFile)
		if !exists {
			unknown[file.Path] = struct{}{}
			continue
		}
		converted := connectivityclient.FileMount{
			Path:         file.Path,
			Value:        cloneString(file.Value),
			SecretRef:    file.SecretRef.clientRef(),
			ConfigMapRef: file.ConfigMapRef.clientRef(),
			Mode:         cloneMode(file.Mode),
		}
		if err := validateSources(converted.Value, converted.SecretRef, converted.ConfigMapRef); err != nil {
			return fmt.Errorf("configuration key %q: %w", file.Path, err)
		}
		replaceFile(config, converted)
	}
	return nil
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

func (ref *documentRef) clientRef() *connectivityclient.KeyRef {
	if ref == nil {
		return nil
	}
	return &connectivityclient.KeyRef{Name: ref.Name, Key: ref.Key}
}

func validateSources(value *string, secretRef, configMapRef *connectivityclient.KeyRef) error {
	count := 0
	if value != nil {
		count++
	}
	if secretRef != nil {
		count++
		if secretRef.Name == "" || secretRef.Key == "" {
			return fmt.Errorf("secret reference requires name and key")
		}
	}
	if configMapRef != nil {
		count++
		if configMapRef.Name == "" || configMapRef.Key == "" {
			return fmt.Errorf("configmap reference requires name and key")
		}
	}
	if count != 1 {
		return fmt.Errorf("exactly one of value, secretRef, or configMapRef is required")
	}
	return nil
}

func applyValue(config *connectivityclient.ConnectorInstanceConfig, field SchemaField, value parsedValue) {
	if field.Kind == ConfigEnv {
		config.Env[field.Key] = connectivityclient.EnvValue{
			Value:        value.value,
			SecretRef:    value.secretRef,
			ConfigMapRef: value.configMapRef,
		}
		return
	}
	replaceFile(config, connectivityclient.FileMount{
		Path:         field.Key,
		Value:        value.value,
		SecretRef:    value.secretRef,
		ConfigMapRef: value.configMapRef,
	})
}

func replaceFile(config *connectivityclient.ConnectorInstanceConfig, replacement connectivityclient.FileMount) {
	for index := range config.Files {
		if config.Files[index].Path == replacement.Path {
			if replacement.Mode == nil {
				replacement.Mode = cloneMode(config.Files[index].Mode)
			}
			config.Files[index] = replacement
			return
		}
	}
	config.Files = append(config.Files, replacement)
}

func hasValue(config *connectivityclient.ConnectorInstanceConfig, field SchemaField) bool {
	if field.Kind == ConfigEnv {
		value, exists := config.Env[field.Key]
		return exists && validateSources(value.Value, value.SecretRef, value.ConfigMapRef) == nil
	}
	for _, file := range config.Files {
		if file.Path == field.Key {
			return validateSources(file.Value, file.SecretRef, file.ConfigMapRef) == nil
		}
	}
	return false
}

func cloneConfig(source *connectivityclient.ConnectorInstanceConfig) *connectivityclient.ConnectorInstanceConfig {
	clone := &connectivityclient.ConnectorInstanceConfig{Env: make(map[string]connectivityclient.EnvValue)}
	if source == nil {
		return clone
	}
	for key, value := range source.Env {
		clone.Env[key] = cloneEnvValue(value)
	}
	clone.Files = make([]connectivityclient.FileMount, len(source.Files))
	for index, file := range source.Files {
		clone.Files[index] = cloneFileMount(file)
	}
	return clone
}

func cloneEnvValue(value connectivityclient.EnvValue) connectivityclient.EnvValue {
	return connectivityclient.EnvValue{
		Value:        cloneString(value.Value),
		SecretRef:    cloneRef(value.SecretRef),
		ConfigMapRef: cloneRef(value.ConfigMapRef),
	}
}

func cloneFileMount(file connectivityclient.FileMount) connectivityclient.FileMount {
	clone := connectivityclient.FileMount{
		Path:         file.Path,
		Value:        cloneString(file.Value),
		SecretRef:    cloneRef(file.SecretRef),
		ConfigMapRef: cloneRef(file.ConfigMapRef),
	}
	clone.Mode = cloneMode(file.Mode)
	return clone
}

func cloneMode(value *int32) *int32 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	return stringPointer(*value)
}

func cloneRef(ref *connectivityclient.KeyRef) *connectivityclient.KeyRef {
	if ref == nil {
		return nil
	}
	return &connectivityclient.KeyRef{Name: ref.Name, Key: ref.Key}
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

func scalarString(raw any) (string, error) {
	switch value := raw.(type) {
	case string:
		return value, nil
	case nil:
		return "", nil
	case bool, int, int64, uint64, float64:
		return fmt.Sprint(value), nil
	default:
		return "", fmt.Errorf("flat configuration values must be scalars")
	}
}

func stringPointer(value string) *string {
	return &value
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
