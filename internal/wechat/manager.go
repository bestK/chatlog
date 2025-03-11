package wechat

import (
	"strings"

	"github.com/shirou/gopsutil/v4/process"
	log "github.com/sirupsen/logrus"
)

const (
	V3ProcessName = "WeChat"
	V4ProcessName = "Weixin"
)

var (
	Items   []*Info
	ItemMap map[string]*Info
)

func Load() {
	Items = make([]*Info, 0, 2)
	ItemMap = make(map[string]*Info)

	processes, err := process.Processes()
	if err != nil {
		log.Println("获取进程列表失败:", err)
		return
	}

	for _, p := range processes {
		name, err := p.Name()
		name = strings.TrimSuffix(name, ".exe")
		if err != nil || name != V3ProcessName && name != V4ProcessName {
			continue
		}

		// v4 存在同名进程，需要继续判断 cmdline
		if name == V4ProcessName {
			cmdline, err := p.Cmdline()
			if err != nil {
				log.Error(err)
				continue
			}
			if strings.Contains(cmdline, "--") {
				continue
			}
		}

		info, err := NewInfo(p)
		if err != nil {
			continue
		}

		Items = append(Items, info)
		ItemMap[info.AccountName] = info
	}
}
