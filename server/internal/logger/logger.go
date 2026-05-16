package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var currentFormat string

func Setup(level, format string) {
	currentFormat = strings.ToLower(format)
	zerolog.TimeFieldFormat = time.RFC3339
	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	rebuildLogger(nil)
}

// AddHubWriter fans log output to w in addition to stdout.
// w always receives raw JSON because zerolog writes JSON before ConsoleWriter
// transforms it; HubWriter can therefore parse the JSON regardless of mode.
func AddHubWriter(w io.Writer) {
	rebuildLogger(w)
}

func rebuildLogger(extra io.Writer) {
	// stdOut is the terminal sink — Console or raw JSON depending on config.
	var stdOut io.Writer
	if currentFormat == "console" {
		stdOut = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		stdOut = os.Stdout
	}

	var out io.Writer
	if extra != nil {
		// zerolog writes JSON to MultiWriter; ConsoleWriter transforms for
		// terminal while extra (HubWriter) receives the same raw JSON.
		out = io.MultiWriter(stdOut, extra)
	} else {
		out = stdOut
	}

	log.Logger = zerolog.New(out).With().Timestamp().Logger()
}

func L() *zerolog.Logger {
	return &log.Logger
}
