//go:build !darwin && !linux

package plugin

import (
	"io"
	"os"
)

func openPluginStreamFileSink(file *os.File) (io.Writer, func() error, error) {
	return file, nil, nil
}
