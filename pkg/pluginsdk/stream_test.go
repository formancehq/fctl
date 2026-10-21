package pluginsdk

import "testing"

func TestProgressEventValidation(t *testing.T) {
	for _, event := range []ProgressEvent{
		{Phase: "import", Unit: "logs", Completed: 123},
		{Phase: "read", Unit: "bytes", Total: new(int64(0))},
		{Phase: "read", Unit: "bytes", Completed: 8, Total: new(int64(10))},
	} {
		if err := event.Validate(); err != nil {
			t.Fatalf("valid event %+v: %v", event, err)
		}
	}
	for _, event := range []ProgressEvent{
		{Phase: "", Unit: "logs"}, {Phase: "import", Unit: ""},
		{Phase: "\x1b[2J", Unit: "logs"}, {Phase: "import", Unit: "\u202elogs"},
		{Phase: "import", Unit: "logs", Completed: -1},
		{Phase: "import", Unit: "logs", Total: new(int64(-1))},
		{Phase: "import", Unit: "logs", Completed: 2, Total: new(int64(1))},
	} {
		if err := event.Validate(); err == nil {
			t.Fatalf("invalid event accepted: %+v", event)
		}
	}
}
