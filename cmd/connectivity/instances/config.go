package instances

import (
	"bufio"
	"fmt"
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

func SchemaFields(plugin *connectivityclient.Plugin) (map[string]SchemaField, error) {
	if plugin == nil {
		return nil, fmt.Errorf("plugin is required")
	}
	if len(plugin.Spec.ConfigSchema) == 0 {
		return map[string]SchemaField{}, nil
	}

	fields := make(map[string]SchemaField)
	sections := []struct {
		name string
		kind ConfigKind
		raw  any
	}{
		{name: "env", kind: ConfigEnv, raw: plugin.Spec.ConfigSchema["env"]},
		{name: "files", kind: ConfigFile, raw: plugin.Spec.ConfigSchema["files"]},
	}
	if sections[0].raw == nil {
		if _, hasLegacyProperties := plugin.Spec.ConfigSchema["properties"]; hasLegacyProperties {
			sections[0].raw = plugin.Spec.ConfigSchema
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
			return nil, fmt.Errorf("plugin config schema %s must be an object", section.name)
		}
		sectionProperties, ok := stringMap(sectionSchema["properties"])
		if !ok {
			if sectionSchema["properties"] == nil {
				sectionProperties = map[string]any{}
			} else {
				return nil, fmt.Errorf("plugin config schema %s properties must be an object", section.name)
			}
		}
		required, err := requiredSet(sectionSchema["required"])
		if err != nil {
			return nil, fmt.Errorf("plugin config schema %s: %w", section.name, err)
		}

		for key, rawDefinition := range sectionProperties {
			if _, exists := fields[key]; exists {
				return nil, fmt.Errorf("configuration key %q is declared in both env and files", key)
			}
			definition, ok := stringMap(rawDefinition)
			if !ok {
				return nil, fmt.Errorf("plugin config schema field %q must be an object", key)
			}
			format, _ := definition["format"].(string)
			legacySecret, _ := definition["x-secret"].(bool)
			description, _ := definition["description"].(string)
			fields[key] = SchemaField{
				Key:         key,
				Kind:        section.kind,
				Required:    required[key],
				Password:    format == "password" || legacySecret,
				Description: description,
			}
		}
	}

	return fields, nil
}

func BuildInstallConfig(cmd *cobra.Command, plugin *connectivityclient.Plugin, inputs InputOptions, read ReadFileFunc) (*connectivityclient.InstanceConfig, error) {
	var base *connectivityclient.InstanceConfig
	if plugin != nil {
		base = plugin.Spec.Defaults
	}
	return buildConfig(cmd, plugin, base, inputs, read)
}

func BuildConfigureConfig(cmd *cobra.Command, plugin *connectivityclient.Plugin, current *connectivityclient.InstanceConfig, inputs InputOptions, read ReadFileFunc) (*connectivityclient.InstanceConfig, error) {
	return buildConfig(cmd, plugin, current, inputs, read)
}

func buildConfig(cmd *cobra.Command, plugin *connectivityclient.Plugin, base *connectivityclient.InstanceConfig, inputs InputOptions, read ReadFileFunc) (*connectivityclient.InstanceConfig, error) {
	fields, err := SchemaFields(plugin)
	if err != nil {
		return nil, err
	}
	config := cloneConfig(base)
	unknown := make(map[string]struct{})

	if inputs.ConfigFile != "" {
		contents, err := readInput(cmd, read, inputs.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("reading config %q: %w", inputs.ConfigFile, err)
		}
		if err := applyConfigDocument(config, fields, contents, unknown); err != nil {
			return nil, fmt.Errorf("parsing config %q: %w", inputs.ConfigFile, err)
		}
	}

	for _, path := range inputs.EnvFiles {
		contents, err := readInput(cmd, read, path)
		if err != nil {
			return nil, fmt.Errorf("reading env file %q: %w", path, err)
		}
		if err := applyDotenv(config, fields, contents, unknown); err != nil {
			return nil, fmt.Errorf("parsing env file %s:%w", path, err)
		}
	}

	for _, assignment := range inputs.SetValues {
		key, value, err := parseAssignment(assignment)
		if err != nil {
			return nil, err
		}
		field, exists := fields[key]
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

func applyDotenv(config *connectivityclient.InstanceConfig, fields map[string]SchemaField, contents string, unknown map[string]struct{}) error {
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
		field, exists := fields[key]
		if !exists || field.Kind != ConfigEnv {
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

func applyConfigDocument(config *connectivityclient.InstanceConfig, fields map[string]SchemaField, contents string, unknown map[string]struct{}) error {
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
			field, exists := fields[key]
			if !exists || field.Kind != ConfigEnv {
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
		field, exists := fields[key]
		if !exists || field.Kind != ConfigEnv {
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
		field, exists := fields[file.Path]
		if !exists || field.Kind != ConfigFile {
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

func applyValue(config *connectivityclient.InstanceConfig, field SchemaField, value parsedValue) {
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

func replaceFile(config *connectivityclient.InstanceConfig, replacement connectivityclient.FileMount) {
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

func hasValue(config *connectivityclient.InstanceConfig, field SchemaField) bool {
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

func cloneConfig(source *connectivityclient.InstanceConfig) *connectivityclient.InstanceConfig {
	clone := &connectivityclient.InstanceConfig{Env: make(map[string]connectivityclient.EnvValue)}
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
