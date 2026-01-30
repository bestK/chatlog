package chatlog

import (
	"io"
	"os"
	"path/filepath"

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

	writers := []io.Writer{zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}}
	if fw := getLogWriter(); fw != nil {
		writers = append(writers, fw)
	}

	log.Logger = log.Output(io.MultiWriter(writers...))
}

func initTuiLog(cmd *cobra.Command, args []string) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	debug, _ := cmd.Flags().GetBool("debug")
	// 也支持环境变量 CHATLOG_DEBUG
	if !debug && os.Getenv("CHATLOG_DEBUG") == "true" {
		debug = true
	}
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// debug 模式时同时输出到控制台和文件
	if debug {
		writers := []io.Writer{zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}}
		if fw := getLogWriter(); fw != nil {
			writers = append(writers, fw)
		}
		log.Logger = log.Output(io.MultiWriter(writers...))
		logrus.SetOutput(os.Stderr)
	} else {
		logOutput := io.Discard
		if fw := getLogWriter(); fw != nil {
			logOutput = fw
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: logOutput, NoColor: true, TimeFormat: "2006-01-02 15:04:05"})
		logrus.SetOutput(logOutput)
	}
}
