package util

import (
	"io"

	"github.com/rs/zerolog"
)

const LogTimeFormat = "2006/01/02 15:04:05"

func NewPlainLogWriter(out io.Writer, noColor bool) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        out,
		NoColor:    noColor,
		TimeFormat: LogTimeFormat,
	}
}
