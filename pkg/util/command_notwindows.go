//go:build !windows

package util

import "os/exec"

func applyCommandOptions(cmd *exec.Cmd) {}
