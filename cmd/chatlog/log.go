package chatlog

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/sjzar/chatlog/pkg/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Debug bool

func initLog(cmd *cobra.Command, args []string) {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			_, filename := path.Split(f.File)
			return "", fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})

	if Debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	}
}

func initTuiLog(cmd *cobra.Command, args []string) {
	logOutput := io.Discard

	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		logpath := util.DefaultWorkDir("")
		util.PrepareDir(logpath)
		logFD, err := os.OpenFile(filepath.Join(logpath, "chatlog.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			panic(err)
		}
		logOutput = logFD
		log.SetReportCaller(true)
	}

	log.SetOutput(logOutput)
}
