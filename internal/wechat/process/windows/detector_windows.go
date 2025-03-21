package windows

import (
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
	log "github.com/sirupsen/logrus"

	"github.com/sjzar/chatlog/internal/wechat/model"
)

// initializeProcessInfo 获取进程的数据目录和账户名
func initializeProcessInfo(p *process.Process, info *model.Process) error {
	files, err := p.OpenFiles()
	if err != nil {
		log.Error("获取打开文件列表失败: ", err)
		return err
	}

	dbPath := V3DBFile
	if info.Version == 4 {
		dbPath = V4DBFile
	}

	for _, f := range files {
		if strings.HasSuffix(f.Path, dbPath) {
			filePath := f.Path[4:] // 移除 "\\?\" 前缀
			parts := strings.Split(filePath, string(filepath.Separator))
			if len(parts) < 4 {
				log.Debug("无效的文件路径格式: " + filePath)
				continue
			}

			info.Status = model.StatusOnline
			if info.Version == 4 {
				info.DataDir = strings.Join(parts[:len(parts)-3], string(filepath.Separator))
				info.AccountName = parts[len(parts)-4]
			} else {
				info.DataDir = strings.Join(parts[:len(parts)-2], string(filepath.Separator))
				info.AccountName = parts[len(parts)-3]
			}
			return nil
		}
	}

	return nil
}
