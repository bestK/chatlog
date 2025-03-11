package wechat

import (
	"github.com/sjzar/chatlog/pkg/dllver"

	"github.com/shirou/gopsutil/v4/process"
	log "github.com/sirupsen/logrus"
)

const (
	StatusInit    = ""
	StatusOffline = "offline"
	StatusOnline  = "online"
)

type Info struct {
	PID         uint32
	ExePath     string
	Version     *dllver.Info
	Status      string
	DataDir     string
	AccountName string
	Key         string
}

func NewInfo(p *process.Process) (*Info, error) {
	info := &Info{
		PID:    uint32(p.Pid),
		Status: StatusOffline,
	}

	var err error
	info.ExePath, err = p.Exe()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	info.Version, err = dllver.New(info.ExePath)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if err := info.initialize(p); err != nil {
		return nil, err
	}

	return info, nil
}
