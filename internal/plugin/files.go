package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/muesli/cancelreader"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/internal/command"
)

const maxPluginInput = 4 << 20

func canSupplyFileBody(files *pluginsdk.FileSpec, flag pluginsdk.FlagSpec) bool {
	return flag.Body && files != nil && (files.ReadArgument != nil || files.ReadFlag != "")
}

func pluginFileInputs(spec pluginsdk.CommandSpec, request pluginsdk.ExecuteRequest, bodyFlag string) ([]pluginsdk.InputSpec, error) {
	source, err := pluginInputSource(spec.Files, request)
	if err != nil || source == "" {
		return spec.Inputs, err
	}
	// The file will provide the body after confirmation, so do not ask the
	// user to enter the same content in a form while the file is still unread.
	return slices.DeleteFunc(slices.Clone(spec.Inputs), func(input pluginsdk.InputSpec) bool {
		return input.BodyPointer != "" || (bodyFlag != "" && input.Flag == bodyFlag)
	}), nil
}

func pluginInputSource(files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest) (string, error) {
	if files == nil {
		return "", nil
	}
	source := ""
	if files.ReadArgument != nil && *files.ReadArgument >= 0 && len(request.Args) > *files.ReadArgument {
		source = request.Args[*files.ReadArgument]
		if source == "" {
			return "", fmt.Errorf("file input argument must not be empty")
		}
	}
	if files.ReadFlag != "" {
		value := request.Flags[files.ReadFlag]
		if request.ChangedFlags[files.ReadFlag] && value == "" {
			return "", fmt.Errorf("--%s file input must not be empty", files.ReadFlag)
		}
		if value != "" {
			if source != "" {
				return "", fmt.Errorf("file argument and --%s cannot be combined", files.ReadFlag)
			}
			source = value
		}
	}
	return source, nil
}

func checkPluginInput(files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest, flags []pluginsdk.FlagSpec) error {
	source, err := pluginInputSource(files, request)
	if err != nil || source == "" {
		return err
	}
	if request.Body != nil {
		return fmt.Errorf("file input cannot be combined with an explicit body")
	}
	for _, flag := range flags {
		if flag.Body && (request.ChangedFlags[flag.Name] || request.Flags[flag.Name] != "") {
			return fmt.Errorf("file input cannot be combined with --%s", flag.Name)
		}
	}
	return nil
}

// ReadPluginInput reads an optional declared source into Body. The caller must
// enforce SDK gates and confirmation before invoking this host-only boundary.
func ReadPluginInput(cmd *cobra.Command, files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteRequest, error) {
	source, err := pluginInputSource(files, request)
	if err != nil || source == "" {
		return request, err
	}
	if request.Body != nil {
		return request, fmt.Errorf("file input cannot be combined with an explicit body")
	}
	data, err := readPluginSource(cmd, source)
	if err != nil {
		return request, fmt.Errorf("read plugin input: %w", err)
	}
	request.Body, err = decodePluginInput(files.ReadFormat, data)
	return request, err
}

func readPluginSource(cmd *cobra.Command, source string) (data []byte, err error) {
	if err := cmd.Context().Err(); err != nil {
		return nil, err
	}
	if source == "-" {
		return readPluginStdin(cmd.Context(), cmd.InOrStdin())
	}
	if scheme := strings.ToLower(source); strings.HasPrefix(scheme, "https:") || strings.HasPrefix(scheme, "http:") {
		client := newPluginURLClient()
		defer client.CloseIdleConnections()
		return readPluginURL(cmd.Context(), source, client)
	}
	info, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxPluginInput {
		return nil, fmt.Errorf("input must be a regular file within 4 MiB")
	}
	file, err := os.Open(source) //nolint:gosec // The source is an explicit user-selected local file, validated as regular and bounded above.
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return readBoundedPluginInput(cmd.Context(), file)
}

func readPluginStdin(ctx context.Context, input io.Reader) (data []byte, err error) {
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		// kqueue-based cancellation readers can wait indefinitely for regular
		// files on macOS. A bounded file read does not need readiness polling.
		if info.Mode().IsRegular() {
			return readBoundedPluginInput(ctx, file)
		}
	}
	reader, err := cancelreader.NewReader(input)
	if err != nil {
		return nil, err
	}
	canceled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		reader.Cancel()
		close(canceled)
	})
	defer func() {
		if !stop() {
			<-canceled
		}
		err = errors.Join(err, reader.Close())
	}()
	data, err = readBoundedPluginInput(ctx, reader)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return data, err
}

type pluginContextReader struct {
	ctx context.Context
	io.Reader
}

func (r pluginContextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(data)
}

func readBoundedPluginInput(ctx context.Context, reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(pluginContextReader{ctx, reader}, maxPluginInput+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPluginInput {
		return nil, fmt.Errorf("input exceeds 4 MiB")
	}
	return data, ctx.Err()
}

func decodePluginInput(format string, data []byte) (json.RawMessage, error) {
	var err error
	switch format {
	case "string":
		data, err = json.Marshal(string(data))
	case "yaml":
		data, err = yaml.YAMLToJSONStrict(data)
	case "", "json":
	default:
		return nil, fmt.Errorf("unsupported input format %q", format)
	}
	if err != nil {
		return nil, fmt.Errorf("decode plugin input: %w", err)
	}
	if len(data) > maxPluginInput || !json.Valid(data) {
		return nil, fmt.Errorf("decoded plugin input must be valid JSON within 4 MiB")
	}
	return data, nil
}

func pluginOutputFormat(files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest) (string, error) {
	format := ""
	if files != nil {
		format = files.WriteFormat
		if files.FormatFlag != "" && request.Flags[files.FormatFlag] != "" {
			format = request.Flags[files.FormatFlag]
		}
	}
	if format == "yml" {
		format = "yaml"
	}
	if !slices.Contains([]string{"", "json", "ndjson", "yaml"}, format) {
		return "", fmt.Errorf("unsupported output format %q", format)
	}
	return format, nil
}

func pluginOutputDestination(files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest) (string, error) {
	if files == nil || files.WriteFlag == "" {
		return "", nil
	}
	destination := request.Flags[files.WriteFlag]
	if request.ChangedFlags[files.WriteFlag] && destination == "" {
		return "", fmt.Errorf("--%s output file must not be empty", files.WriteFlag)
	}
	return destination, nil
}

func renderPluginOutput(cmd *cobra.Command, files *pluginsdk.FileSpec, request pluginsdk.ExecuteRequest, response pluginsdk.ExecuteResponse, execErr error) error {
	if len(response.Data) == 0 {
		return execErr
	}
	// Preserve partial-result reporting, but never turn failed execution into
	// a successful export or replace the destination with partial data.
	if execErr != nil {
		return errors.Join(execErr, command.WriteJSON(cmd.OutOrStdout(), response.Data))
	}
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	format, err := pluginOutputFormat(files, request)
	if err != nil {
		return err
	}
	destination, err := pluginOutputDestination(files, request)
	if err != nil {
		return err
	}
	if destination == "" || destination == "-" {
		return writePluginOutput(cmd.OutOrStdout(), format, response.Data)
	}
	var output bytes.Buffer
	if err := writePluginOutput(&output, format, response.Data); err != nil {
		return err
	}
	return writePluginFile(cmd.Context(), destination, output.Bytes())
}

func writePluginOutput(writer io.Writer, format string, data json.RawMessage) error {
	switch format {
	case "", "json":
		return command.WriteJSON(writer, data)
	case "yaml":
		if !json.Valid(data) {
			return fmt.Errorf("plugin output must be valid JSON")
		}
		output, err := yaml.JSONToYAML(data)
		if err != nil {
			return err
		}
		_, err = writer.Write(output)
		return err
	case "ndjson":
		return writePluginNDJSON(writer, data)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writePluginNDJSON(writer io.Writer, data json.RawMessage) error {
	if !json.Valid(data) {
		return fmt.Errorf("plugin output must be valid JSON")
	}
	data, err := pluginNDJSONRecords(data)
	if err != nil {
		return err
	}
	var records []json.RawMessage
	if err := json.Unmarshal(data, &records); err != nil || bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("NDJSON output requires an array or a data/logs array envelope")
	}
	var output bytes.Buffer
	for _, record := range records {
		if err := json.Compact(&output, record); err != nil {
			return err
		}
		if err := output.WriteByte('\n'); err != nil {
			return err
		}
	}
	_, err = writer.Write(output.Bytes())
	return err
}

func pluginNDJSONRecords(data json.RawMessage) (json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return data, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	for _, name := range []string{"logs", "data"} {
		candidate := bytes.TrimSpace(envelope[name])
		if len(candidate) > 0 && candidate[0] == '[' {
			return candidate, nil
		}
	}
	return data, nil
}

func writePluginFile(ctx context.Context, destination string, data []byte) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(destination), ".fctl-export-*")
	if err != nil {
		return fmt.Errorf("create plugin export: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			err = errors.Join(err, file.Close())
		}
		if removeErr := os.Remove(file.Name()); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(file.Name(), destination)
}
