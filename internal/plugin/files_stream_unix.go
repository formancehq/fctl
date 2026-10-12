//go:build darwin || linux

package plugin

import (
	"errors"
	"io"
	"math"
	"os"
	"runtime"
	"sync"

	"golang.org/x/sys/unix"
)

// Inherited stdout pipes are blocking os.Files and Close alone cannot interrupt
// their Write syscall. An owned nonblocking duplicate uses Go's poller instead.
// The descriptor shares its flags with the original, which we restore on Close.
func openPluginStreamFileSink(original *os.File) (io.Writer, func() error, error) {
	info, err := original.Stat()
	if err != nil || info.Mode().IsRegular() {
		return original, nil, err
	}
	fd := original.Fd()
	if fd > math.MaxInt {
		return nil, nil, os.ErrInvalid
	}
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, nil, err
	}
	duplicate, err := unix.FcntlInt(fd, unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	if err := unix.SetNonblock(duplicate, true); err != nil {
		return nil, nil, errors.Join(err, unix.Close(duplicate))
	}
	file := os.NewFile(uintptr(duplicate), original.Name()) //nolint:gosec // F_DUPFD_CLOEXEC returns a nonnegative file descriptor.
	closeSink := sync.OnceValue(func() error {
		err := file.Close()
		_, restoreErr := unix.FcntlInt(fd, unix.F_SETFL, flags)
		runtime.KeepAlive(original)
		return errors.Join(err, restoreErr)
	})
	return &pluginStreamFileSink{Writer: file, close: closeSink}, closeSink, nil
}

type pluginStreamFileSink struct {
	io.Writer
	close func() error
}

func (w *pluginStreamFileSink) Close() error { return w.close() }
