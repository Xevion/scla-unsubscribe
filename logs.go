package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const timeFormat = "2006-01-02 15:04:05"

var (
	standardOut = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: timeFormat}
	errorOut    = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: timeFormat}
)

// logSplitter implements zerolog.LevelWriter
type logSplitter struct{}

// Write should not be called
func (l logSplitter) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

// WriteLevel write to the appropriate output
func (l logSplitter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	if level <= zerolog.WarnLevel {
		return standardOut.Write(p)
	} else {
		return errorOut.Write(p)
	}
}

type loggingLevel int

const (
	DEBUG loggingLevel = iota
	INFO
	WARNING
	ERROR
)

type badgerZerologLogger struct {
	level loggingLevel
}

func (l badgerZerologLogger) Errorf(format string, args ...interface{}) {
	if l.level <= ERROR {
		log.Error().Msg(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
	}
}

func (l badgerZerologLogger) Warningf(format string, args ...interface{}) {
	if l.level <= WARNING {
		log.Warn().Msg(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
	}
}

func (l badgerZerologLogger) Infof(format string, args ...interface{}) {
	if l.level <= INFO {
		log.Info().Msg(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
	}
}

func (l badgerZerologLogger) Debugf(format string, args ...interface{}) {
	if l.level <= DEBUG {
		log.Debug().Msg(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
	}
}
