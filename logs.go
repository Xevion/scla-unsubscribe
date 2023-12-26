package main

import (
	"os"

	"github.com/rs/zerolog"
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
