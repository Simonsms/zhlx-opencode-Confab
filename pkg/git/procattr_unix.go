//go:build !windows

package git

import "syscall"

func gitProcAttr() *syscall.SysProcAttr {
	return nil
}
