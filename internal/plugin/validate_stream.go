package plugin

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func streamInput(spec pluginsdk.CommandSpec) bool {
	return spec.Streams != nil && spec.Streams.Input
}

func validPluginReadFormat(spec pluginsdk.CommandSpec) bool {
	return (streamInput(spec) && spec.Files.ReadFormat == "ndjson") ||
		slices.Contains([]string{"", "string", "json", "yaml"}, spec.Files.ReadFormat)
}

func validateStreamSpec(spec pluginsdk.CommandSpec, flags map[string]pluginsdk.FlagSpec) error {
	if spec.Streams == nil {
		return nil
	}
	if !spec.Runnable || len(spec.Subcommands) != 0 {
		return fmt.Errorf("streams require a runnable leaf")
	}
	if !spec.Streams.Input && !spec.Streams.Output {
		return nil
	}
	if spec.Files == nil {
		return fmt.Errorf("streams require a file specification")
	}
	if spec.Streams.Input {
		if spec.Files.ReadArgument == nil && spec.Files.ReadFlag == "" {
			return fmt.Errorf("input stream requires a declared source")
		}
		if !slices.Contains([]string{"string", "ndjson"}, spec.Files.ReadFormat) {
			return fmt.Errorf("input stream requires a raw string or ndjson format")
		}
	}
	if spec.Streams.Output && spec.Files.FormatFlag != "" {
		return checkPluginStreamFormat(spec, pluginsdk.ExecuteRequest{Flags: map[string]string{
			spec.Files.FormatFlag: flags[spec.Files.FormatFlag].Default,
		}})
	}
	return nil
}

func checkPluginStreamFormat(spec pluginsdk.CommandSpec, request pluginsdk.ExecuteRequest) error {
	if spec.Streams == nil || !spec.Streams.Output {
		return nil
	}
	format, err := pluginOutputFormat(spec.Files, request)
	if err != nil {
		return err
	}
	declared := cmp.Or(spec.Files.WriteFormat, "json")
	if cmp.Or(format, "json") != declared {
		return fmt.Errorf("output stream format must match declared format %q", declared)
	}
	return nil
}
