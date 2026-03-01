package windows

import (
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/sjzar/chatlog/internal/wechat/model"
	"github.com/sjzar/chatlog/pkg/util"
)

// initializeProcessInfo 获取进程的数据目录和账户名
func initializeProcessInfo(p *process.Process, info *model.Process) error {
	files, err := p.OpenFiles()
	if err != nil {
		log.Err(err).Msgf("获取进程 %d 的打开文件失败", p.Pid)
		return err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Path, DBFile) {
			filePath := util.CleanExtendedLengthPath(f.Path)
			parts := strings.Split(filePath, string(filepath.Separator))
			if len(parts) < 4 {
				log.Debug().Msg("无效的文件路径: " + filePath)
				continue
			}

			info.Status = model.StatusOnline
			info.DataDir = strings.Join(parts[:len(parts)-3], string(filepath.Separator))
			info.AccountName = parts[len(parts)-4]
			return nil
		}
	}

	return nil
}
