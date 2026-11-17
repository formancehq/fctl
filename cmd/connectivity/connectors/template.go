package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
)

const (
	templateVersionFlag = "version"
	templateChannelFlag = "channel"
	templateFileFlag    = "file"
	templateForceFlag   = "force"

	// templateStdout is the --file value that prints the template.
	templateStdout = "-"
	// templateWidth is the column comments are wrapped at.
	templateWidth = 88
)

type TemplateStore struct {
	Connector string `json:"connector"`
	Version   string `json:"version"`
	Channel   string `json:"channel,omitempty"`
	// File is where the template was written; empty when printed.
	File    string `json:"file,omitempty"`
	Warning string `json:"warning,omitempty"`
	Content string `json:"content"`
}

type TemplateController struct {
	store *TemplateStore
	now   func() time.Time
	// gitIgnored reports whether path is ignored by git; ok is false when it
	// cannot tell (git missing, not a work tree).
	gitIgnored func(ctx context.Context, path string) (ignored, ok bool)
}

var _ fctl.Controller[*TemplateStore] = (*TemplateController)(nil)

func NewTemplateController() *TemplateController {
	return &TemplateController{
		store:      &TemplateStore{},
		now:        time.Now,
		gitIgnored: gitCheckIgnore,
	}
}

func NewTemplateCommand() *cobra.Command {
	controller := NewTemplateController()
	return fctl.NewCommand(
		"template <connector>",
		fctl.WithAliases("tpl"),
		fctl.WithShortDescription("Generate a commented YAML configuration file for a connector"),
		fctl.WithDescription(strings.Join([]string{
			"Generate a YAML configuration file for a connector, ready to fill in and pass to",
			"'fctl connectivity connections install --config'. Each key is commented with its",
			"description, whether it is required, its type, allowed values and default.",
			"",
			"The file is written to ./<connector>.yaml unless --file is set, with permissions",
			"restricted to the current user. Use --file - to print it instead.",
		}, "\n")),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorNames()),
		fctl.WithStringFlag(templateVersionFlag, "", "Use this exact version's schema, for example v0.5.0-beta.14. Defaults to the latest version"),
		fctl.WithStringFlag(templateChannelFlag, "", "Use the schema of the newest version on this channel: stable, rc, beta or alpha"),
		fctl.WithStringFlag(templateFileFlag, "", "Destination file. Defaults to ./<connector>.yaml; use - to print to stdout"),
		fctl.WithBoolFlag(templateForceFlag, false, "Overwrite the destination file if it already exists"),
		fctl.WithController[*TemplateStore](controller),
		fctl.WithFlagCompletion(templateVersionFlag, completeShowVersions),
		fctl.WithFlagCompletion(templateChannelFlag, connectivityinternal.CompleteChannels()),
		fctl.WithExamples(
			"fctl connectivity connectors template ee-stripeconnect",
			"fctl connectivity connectors template ee-stripeconnect --channel beta --file stripe.yaml",
			"fctl connectivity connectors template ee-stripeconnect --file - | less",
		),
	)
}

func (c *TemplateController) GetStore() *TemplateStore {
	return c.store
}

func (c *TemplateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, stackClient.Connectivity.V1)
}

func (c *TemplateController) run(cmd *cobra.Command, args []string, client *formance.FormanceV1) (fctl.Renderable, error) {
	name := args[0]
	version, channel := fctl.GetString(cmd, templateVersionFlag), fctl.GetString(cmd, templateChannelFlag)
	if version != "" && channel != "" {
		return nil, fmt.Errorf("--version and --channel cannot be combined")
	}
	selected := version
	if selected == "" {
		selected = channel
	}

	file := fctl.GetString(cmd, templateFileFlag)
	if file == "" {
		if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
			return nil, fmt.Errorf("connector name %q cannot be used as a file name; set --file", name)
		}
		file = name + ".yaml"
	}

	connector, err := getConnector(cmd, client, name)
	if err != nil {
		return nil, err
	}
	schema, err := getSchemaVersion(cmd, client, name, *connector, selected)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, fmt.Errorf("connector %q has no published version", name)
	}

	configFile := file
	if file == templateStdout {
		configFile = name + ".yaml"
	}
	content := renderTemplate(templateOptions{
		connector:  name,
		version:    schema.Version,
		channel:    channel,
		configFile: configFile,
		generated:  c.now(),
	}, schemaFields(schema.ConfigSchema), schemaRules(schema.ConfigSchema))

	c.store.Connector, c.store.Version, c.store.Channel, c.store.Content = name, schema.Version, channel, content
	c.store.File, c.store.Warning = "", ""
	if file == templateStdout {
		return c, nil
	}
	if err := writeTemplateFile(file, content, fctl.GetBool(cmd, templateForceFlag)); err != nil {
		return nil, err
	}
	c.store.File = file
	if ignored, ok := c.gitIgnored(cmd.Context(), file); ok && !ignored {
		c.store.Warning = fmt.Sprintf("%s is inside a git repository and not ignored. Add it to .gitignore before filling in credentials.", file)
	}
	return c, nil
}

func (c *TemplateController) Render(cmd *cobra.Command, _ []string) error {
	if c.store.File == "" {
		_, err := fmt.Fprint(cmd.OutOrStdout(), c.store.Content)
		return err
	}
	if c.store.Warning != "" {
		if err := fctl.RenderText(cmd.ErrOrStderr(), "Warning: "+c.store.Warning); err != nil {
			return err
		}
	}
	return fctl.RenderText(cmd.OutOrStdout(), fmt.Sprintf("Configuration template for %q (%s) written to %s.", c.store.Connector, c.store.Version, c.store.File))
}

// writeTemplateFile creates path readable only by the current user. It never
// replaces an existing file unless force is set.
func writeTemplateFile(path, content string, force bool) error {
	path = filepath.Clean(path)
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%s already exists; use --force to overwrite it", path)
	}
	if err != nil {
		return err
	}
	// An overwritten file keeps its previous mode otherwise.
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// gitCheckIgnore asks git whether path is ignored. Exit status 0 means
// ignored, 1 not ignored; anything else (not a repository, git missing)
// means unknown.
func gitCheckIgnore(ctx context.Context, path string) (bool, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false, false
	}
	// #nosec G204 -- fixed git subcommand; the path is passed after "--".
	command := exec.CommandContext(ctx, "git", "-C", filepath.Dir(absolute), "check-ignore", "-q", "--", absolute)
	err = command.Run()
	if err == nil {
		return true, true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, true
	}
	return false, false
}

type templateOptions struct {
	connector, version, channel, configFile string
	generated                               time.Time
}

// renderTemplate builds the YAML document. Required keys are left empty, so
// install reports them as missing until filled in; optional keys are
// commented out with their default. Secrets are never pre-filled.
func renderTemplate(options templateOptions, fields []schemaField, rules []string) string {
	var out strings.Builder
	secrets := false
	for _, field := range fields {
		secrets = secrets || field.secret()
	}

	generatedFrom := options.version
	if options.channel != "" {
		generatedFrom += " (channel: " + options.channel + ")"
	}
	selector := "--version " + options.version
	if options.channel != "" {
		selector = "--channel " + options.channel
	}
	comment(&out, "", fmt.Sprintf("Connection configuration for connector %q", options.connector))
	comment(&out, "", fmt.Sprintf("Generated from version %s on %s.", generatedFrom, options.generated.Format(time.DateOnly)))
	comment(&out, "", "")
	comment(&out, "", "Fill in the values, then install with:")
	comment(&out, "", "  fctl connectivity connections install "+options.connector+" \\")
	comment(&out, "", "    --ledger <ledger> "+selector+" --config "+options.configFile)
	comment(&out, "", "")
	comment(&out, "", "Keys left empty are unset: install fails while a required key is unset.")
	comment(&out, "", `Values are strings: quote numbers and booleans, for example "100" or "true".`)

	if len(rules) > 0 {
		comment(&out, "", "")
		comment(&out, "", "Rules:")
		for _, rule := range rules {
			wrapped(&out, "", "  • ", "    ", rule)
		}
	}

	if secrets {
		comment(&out, "", "")
		comment(&out, "", "⚠ This file contains credentials once filled in. To keep them safe:")
		comment(&out, "", "  • Do not commit it: add it to your .gitignore.")
		comment(&out, "", "  • Keep it readable only by you: chmod 600 "+options.configFile+" (fctl creates it this way).")
		comment(&out, "", "  • Delete it once the connection is installed, or store it in a password manager.")
		comment(&out, "", "  • Prefer sandbox/test credentials while trying things out.")
		comment(&out, "", "  • To keep secrets out of this file entirely, leave them empty here and pass them")
		comment(&out, "", "    at install time instead: --set KEY=@./path/to/secret")
		comment(&out, "", "  • If this file leaks, rotate the credentials with the provider right away.")
	}

	if len(fields) == 0 {
		out.WriteString("\n")
		comment(&out, "", "This connector declares no configuration fields.")
		return out.String()
	}

	sections := []struct{ name, source string }{
		{name: "env", source: "environment"},
		{name: "files", source: "file"},
	}
	for _, section := range sections {
		sectionFields := make([]schemaField, 0, len(fields))
		for _, field := range fields {
			if field.source == section.source {
				sectionFields = append(sectionFields, field)
			}
		}
		if len(sectionFields) == 0 {
			continue
		}
		out.WriteString("\n" + section.name + ":\n")
		for index, field := range sectionFields {
			if title, ok := groupTitle(sectionFields, index); ok {
				out.WriteString("\n")
				comment(&out, "  ", groupRule(title))
			}
			out.WriteString("\n")
			renderTemplateField(&out, field)
		}
	}
	return out.String()
}

// groupTitle returns the separator to print before sectionFields[index],
// following the grouping connectors show uses.
func groupTitle(fields []schemaField, index int) (string, bool) {
	field := fields[index]
	if index > 0 && field.groupIndex == fields[index-1].groupIndex {
		return "", false
	}
	switch {
	case field.group != "" && field.advanced:
		return field.group + " (advanced)", true
	case field.group != "":
		return field.group, true
	case index > 0:
		return "Other", true
	}
	return "", false
}

func groupRule(title string) string {
	rule := "── " + title + " "
	if pad := 60 - len([]rune(rule)); pad > 0 {
		rule += strings.Repeat("─", pad)
	}
	return rule
}

func renderTemplateField(out *strings.Builder, field schemaField) {
	const indent = "  "
	if field.details != "" {
		wrapped(out, indent, "", "", field.details)
	}

	meta := []string{"Optional"}
	if field.required {
		meta[0] = "Required"
	}
	switch {
	case field.secret():
		meta = append(meta, "secret")
	case field.kind != "":
		meta = append(meta, field.kind)
	}
	if field.source == "file" {
		meta = append(meta, "file content, mounted at this path in the connector")
	}
	if len(field.enum) > 0 {
		meta = append(meta, "one of: "+strings.Join(field.enum, ", "))
	}
	prefill := field.hasDefault && !field.secret()
	if prefill {
		meta = append(meta, "default: "+field.defaultText)
	}
	comment(out, indent, strings.Join(meta, " · "))

	line := yamlKey(field.key) + ":"
	if prefill {
		line += " " + yamlString(field.defaultText)
	}
	if field.required {
		out.WriteString(indent + line + "\n")
		return
	}
	out.WriteString(indent + "# " + line + "\n")
}

func comment(out *strings.Builder, indent, text string) {
	if text == "" {
		out.WriteString(indent + "#\n")
		return
	}
	out.WriteString(indent + "# " + text + "\n")
}

// wrapped writes text as comment lines no wider than templateWidth, with
// first prefixed to the first line and rest to the following ones.
func wrapped(out *strings.Builder, indent, first, rest, text string) {
	width := templateWidth - len(indent) - len("# ") - len([]rune(first))
	for index, line := range fctl.Wrap(text, width) {
		prefix := rest
		if index == 0 {
			prefix = first
		}
		comment(out, indent, prefix+line)
	}
}

// yamlKey returns key as a YAML mapping key, quoted only when needed.
func yamlKey(key string) string {
	encoded, err := yaml.Marshal(key)
	if err != nil {
		return yamlString(key)
	}
	text := strings.TrimSuffix(string(encoded), "\n")
	if strings.Contains(text, "\n") {
		return yamlString(key)
	}
	return text
}

// yamlString returns value as a double-quoted YAML string. JSON strings are
// valid YAML double-quoted scalars, so numbers and booleans stay strings.
func yamlString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
