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

	writers := []io.Writer{util.NewPlainLogWriter(os.Stderr, false)}
	if fw := getLogWriter(); fw != nil {
		writers = append(writers, util.NewPlainLogWriter(fw, true))
	}

	log.Logger = log.Output(io.MultiWriter(writers...))
}
