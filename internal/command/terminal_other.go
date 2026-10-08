//go:build !darwin && !linux && !windows

package command

func consoleInfo(_ uintptr) (bool, int) { return false, 80 }
