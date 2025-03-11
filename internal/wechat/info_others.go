//go:build !windows

package wechat

import "github.com/shirou/gopsutil/v4/process"

// Giao~
// 还没来得及写，Mac 版本打算通过 vmmap 检查内存区域，再用 lldb 读取内存来检查 Key，需要关 SIP 或自签名应用，稍晚再填坑

func (i *Info) initialize(p *process.Process) error {
	return nil
}

func (i *Info) GetKey() (string, error) {
	return "mock-key", nil
}
