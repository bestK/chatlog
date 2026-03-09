package util

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
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

func HasUsableConsole(file *os.File) bool {
	if file == nil {
		return false
	}
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
