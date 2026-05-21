package plugin

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// PrefixedWriter wraps a writer and prefixes each line with a label.
type PrefixedWriter struct {
	prefix string
	out    io.Writer
	mu     sync.Mutex
	buf    bytes.Buffer
}

// NewPrefixedWriter creates a writer that prefixes each line with the given prefix.
func NewPrefixedWriter(prefix string, out io.Writer) *PrefixedWriter {
	return &PrefixedWriter{
		prefix: prefix,
		out:    out,
	}
}

func (w *PrefixedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(p)

	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			// Incomplete line, put it back
			w.buf.Write(line)
			break
		}
		fmt.Fprintf(w.out, "  %s  %s", w.prefix, string(line))
	}

	return len(p), nil
}

// Flush writes any remaining buffered content.
func (w *PrefixedWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buf.Len() > 0 {
		fmt.Fprintf(w.out, "  %s  %s\n", w.prefix, w.buf.String())
		w.buf.Reset()
	}
}

// CapturedWriter captures all written content for later retrieval,
// while optionally forwarding to another writer.
type CapturedWriter struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	forward io.Writer
}

// NewCapturedWriter creates a writer that captures output and optionally forwards it.
func NewCapturedWriter(forward io.Writer) *CapturedWriter {
	return &CapturedWriter{forward: forward}
}

func (w *CapturedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(p)
	if w.forward != nil {
		return w.forward.Write(p)
	}
	return len(p), nil
}

// String returns all captured content.
func (w *CapturedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// PluginStderrConfig configures how plugin stderr is handled.
type PluginStderrConfig struct {
	Debug    bool
	Captured *CapturedWriter
}

// NewPluginStderr creates the stderr writer for a plugin process.
// In debug mode, stderr is prefixed with [plugin:{name}] and forwarded to os.Stderr.
// In all cases, stderr is captured for crash reporting.
func NewPluginStderr(name string, debug bool) (*CapturedWriter, io.Writer) {
	var forward io.Writer
	if debug {
		forward = NewPrefixedWriter(fmt.Sprintf("[plugin:%s]", name), os.Stderr)
	}
	captured := NewCapturedWriter(forward)
	return captured, captured
}

// FormatPluginCrash formats a plugin crash error with captured stderr.
func FormatPluginCrash(name, version string, exitErr error, stderr string) error {
	msg := fmt.Sprintf("plugin %q crashed during execution", name)
	if exitErr != nil {
		msg += fmt.Sprintf("\n\n  %v", exitErr)
	}
	if stderr != "" {
		// Limit stderr to last 20 lines
		lines := splitLines(stderr)
		if len(lines) > 20 {
			lines = lines[len(lines)-20:]
		}
		msg += "\n\n  Stderr:\n"
		for _, line := range lines {
			msg += fmt.Sprintf("    %s\n", line)
		}
	}
	msg += fmt.Sprintf("\n  This is a bug in the %s plugin (v%s), not in fctl.", name, version)
	return fmt.Errorf("%s", msg)
}

func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewBufferString(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
