package chatlog

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sjzar/chatlog/pkg/util"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Debug bool

func getLogWriter() io.Writer {
	logpath := util.DefaultWorkDir("")
	if err := util.PrepareDir(logpath); err != nil {
		return nil
	}
	logFD, err := os.OpenFile(filepath.Join(logpath, "chatlog.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return nil
	}
	return logFD
}

func initLog(cmd *cobra.Command, args []string) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	writers := []io.Writer{zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}}
	if fw := getLogWriter(); fw != nil {
		writers = append(writers, fw)
	}

	log.Logger = log.Output(io.MultiWriter(writers...))
}

func initTuiLog(cmd *cobra.Command, args []string) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logOutput := io.Discard
	if fw := getLogWriter(); fw != nil {
		logOutput = fw
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logOutput, NoColor: true, TimeFormat: time.RFC3339})
	logrus.SetOutput(logOutput)
}
