package chatlog

import (
	"io"
	"os"

	"github.com/sjzar/chatlog/pkg/util"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Debug bool

func getLogWriter() io.Writer {
	logFD, _, err := util.OpenLogFile()
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
