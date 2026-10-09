package plugin

import (
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func TestNestedChoiceFieldsAndLiteralPrecedence(t *testing.T) {
	t.Parallel()
	object := map[string]any{"metadata": map[string]any{"name": "nested"}, "metadata.name": "literal", "status": map[string]any{"phase": "Running"}}
	source := pluginsdk.ChoiceSource{ValueField: "metadata.name", LabelFields: []string{"status.phase"}}
	option, err := choiceOption(object, source)
	if err != nil || option.Value != "literal" || option.Label != "Running (literal)" {
		t.Fatalf("option=%+v err=%v", option, err)
	}
	delete(object, "metadata.name")
	option, err = choiceOption(object, source)
	if err != nil || option.Value != "nested" {
		t.Fatalf("nested option=%+v err=%v", option, err)
	}
	if choiceField(object, "metadata.missing.name") != nil {
		t.Fatal("missing nested field must be nil")
	}
}
