package command

import "golang.org/x/sys/windows"

func consoleInfo(fd uintptr) (bool, int) {
	var mode uint32
	if err := windows.GetConsoleMode(windows.Handle(fd), &mode); err != nil {
		return false, 80
	}
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(fd), &info); err == nil {
		width := int(info.Window.Right-info.Window.Left) + 1
		if width > 0 {
			return true, width
		}
	}
	return true, 80
}
