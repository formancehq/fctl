package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func TestListPluginsPassesPaginationAndRendersApprovedColumnsAndContinuation(t *testing.T) {
	var gotOptions connectivityclient.ListOptions
	client := pluginClientMock{list: func(_ context.Context, options connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
		gotOptions = options
		return &connectivityclient.PluginList{
			Items:    []connectivityclient.Plugin{pluginFixture("stripe")},
			Continue: "next-page",
		}, nil
	}}

	command := NewListCommand(factoryReturning(client))
	output, err := executeCommand(command, "--page-size", "7", "--cursor", "current-page")
	if err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	wantOptions := connectivityclient.ListOptions{Limit: 7, Continue: "current-page"}
	if !reflect.DeepEqual(gotOptions, wantOptions) {
		t.Fatalf("ListPlugins options = %#v, want %#v", gotOptions, wantOptions)
	}
	for _, expected := range []string{
		"Name", "Default Version", "Description", "Capabilities", "Phase",
		"stripe", "2.0.0", "Plugin description", "payments, webhooks", "Ready",
		"HasMore", "true", "PageSize", "7", "Next", "next-page",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("plain output missing %q:\n%s", expected, output)
		}
	}
}

func TestListPluginsJSONPreservesCompleteModelsAndContinuation(t *testing.T) {
	plugin := pluginFixture("stripe")
	plugin.Metadata.Labels = map[string]string{"region": "eu"}
	plugin.Spec.ConfigSchema = map[string]any{"type": "object"}
	client := pluginClientMock{list: func(_ context.Context, _ connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
		return &connectivityclient.PluginList{Items: []connectivityclient.Plugin{plugin}, Continue: "next-page"}, nil
	}}

	command := NewListCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	output, err := executeCommand(command, "--output", "json", "--page-size", "4")
	if err != nil {
		t.Fatalf("execute JSON list command: %v", err)
	}

	var envelope struct {
		Data ListStore `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	if !reflect.DeepEqual(envelope.Data.Plugins, []connectivityclient.Plugin{plugin}) {
		t.Fatalf("JSON plugins = %#v, want complete model %#v", envelope.Data.Plugins, plugin)
	}
	if !envelope.Data.Cursor.HasMore || envelope.Data.Cursor.PageSize != 4 || envelope.Data.Cursor.Next == nil || *envelope.Data.Cursor.Next != "next-page" {
		t.Fatalf("JSON cursor = %#v, want continuation and requested page size", envelope.Data.Cursor)
	}
}

func TestPluginRootRegistersApprovedCommandsAndAliases(t *testing.T) {
	command := NewCommand(factoryReturning(pluginClientMock{}))
	if command.Use != "plugins" || !reflect.DeepEqual(command.Aliases, []string{"plugin", "p"}) {
		t.Fatalf("plugin root = %q aliases %v", command.Use, command.Aliases)
	}

	wantAliases := map[string][]string{
		"list": {"ls", "l"},
		"show": {"get", "g", "sh", "s"},
	}
	for name, aliases := range wantAliases {
		child, _, err := command.Find([]string{name})
		if err != nil {
			t.Fatalf("find %s command: %v", name, err)
		}
		if child.Name() != name || !reflect.DeepEqual(child.Aliases, aliases) {
			t.Errorf("%s aliases = %v, want %v", name, child.Aliases, aliases)
		}
	}
}

func TestListPluginsReturnsFactoryAPIAndEmptyResponseErrors(t *testing.T) {
	tests := map[string]struct {
		factory func(*cobra.Command) (connectivityclient.Client, error)
		want    string
	}{
		"factory": {
			factory: func(*cobra.Command) (connectivityclient.Client, error) {
				return nil, errors.New("authentication failed")
			},
			want: "authentication failed",
		},
		"API": {
			factory: factoryReturning(pluginClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
				return nil, errors.New("catalog unavailable")
			}}),
			want: "catalog unavailable",
		},
		"empty response": {
			factory: factoryReturning(pluginClientMock{list: func(context.Context, connectivityclient.ListOptions) (*connectivityclient.PluginList, error) {
				return nil, nil
			}}),
			want: "empty response",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(NewListCommand(test.factory))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want one containing %q", err, test.want)
			}
		})
	}
}
