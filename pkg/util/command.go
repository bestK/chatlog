package util

import (
	"context"
	"os/exec"
)

func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	applyCommandOptions(cmd)
	return cmd
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	applyCommandOptions(cmd)
	return cmd
}
