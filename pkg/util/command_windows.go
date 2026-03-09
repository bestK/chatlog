//go:build windows

package util

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func applyCommandOptions(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
