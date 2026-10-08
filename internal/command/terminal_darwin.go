package command

import (
	"math"

	"golang.org/x/sys/unix"
)

func consoleInfo(fd uintptr) (bool, int) {
	if fd > math.MaxInt {
		return false, 80
	}
	if _, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA); err != nil {
		return false, 80
	}
	if size, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ); err == nil && size.Col > 0 {
		return true, int(size.Col)
	}
	return true, 80
}
