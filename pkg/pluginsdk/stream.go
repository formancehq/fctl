package pluginsdk

import (
	"context"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// StreamingPlugin is an optional extension. Ordinary commands keep using Execute.
// Commands with Streams must use ExecuteStream; unsupported hosts fail explicitly.
type StreamingPlugin interface {
	Plugin
	ExecuteStream(context.Context, ExecuteRequest, ExecutionStreams) (ExecuteResponse, error)
}

// StreamSpec declares raw byte streams. FileSpec still selects host-owned sources,
// destinations and formats. Stream output bypasses the JSON/table renderer.
type StreamSpec struct {
	Input  bool `json:"input,omitzero"`
	Output bool `json:"output,omitzero"`
}

// ExecutionStreams live only for this invocation. The host owns and closes the
// input/output. Plugins must propagate read, write and Report errors and must not
// retain streams or reporters after returning. Total input size is optional.
type ExecutionStreams struct {
	Input     io.Reader
	InputSize *int64
	Output    io.Writer
	Report    func(ProgressEvent) error
}

// ProgressEvent is an absolute business counter, never terminal text. Total is
// optional; absence means indeterminate. Completed counts confirmed work for the
// named phase/unit. A different phase may describe bytes read before acceptance.
type ProgressEvent struct {
	Phase     string `json:"phase"`
	Completed int64  `json:"completed"`
	Total     *int64 `json:"total,omitzero"`
	Unit      string `json:"unit"`
}

// Validate rejects malformed events before they cross a transport or terminal.
func (p ProgressEvent) Validate() error {
	if !progressLabel(p.Phase) || !progressLabel(p.Unit) || p.Completed < 0 ||
		(p.Total != nil && (*p.Total < 0 || p.Completed > *p.Total)) {
		return fmt.Errorf("invalid plugin progress event")
	}
	return nil
}

func progressLabel(label string) bool {
	return len(label) > 0 && len(label) <= 128 && utf8.ValidString(label) &&
		!containsControl(label)
}

func containsControl(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) || unicode.Is(unicode.Cf, char) {
			return true
		}
	}
	return false
}
