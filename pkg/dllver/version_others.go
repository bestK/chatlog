//go:build !windows

package dllver

func (i *Info) initialize() error {
	return nil
}
