//go:build windows

package git

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// 后台 daemon 没有控制台；Git 子进程也必须禁止新建控制台，避免每次
// 收集仓库信息都唤起 Windows Terminal。HideWindow 同时设置启动显示状态。
func gitProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
