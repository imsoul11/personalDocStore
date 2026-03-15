package log

import (
	"os"
	"strings"
	"github.com/rs/zerolog"
)

var logger zerolog.Logger
// this function runs when first this package is loaded on it own
func init(){
	logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func New(level string){
	var zl zerolog.Level
	switch strings.ToLower(strings.TrimSpace(level)){
	case "debug":
		zl = zerolog.DebugLevel
	case "info":
		zl = zerolog.InfoLevel
	case "warn", "warning":
		zl = zerolog.WarnLevel
	case "error":
		zl = zerolog.ErrorLevel
	case "fatal":
		zl = zerolog.FatalLevel
	case "disabled":
		zl = zerolog.Disabled
	default:
		zl = zerolog.InfoLevel
	}
	logger = zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zl)
}

func Logger() *zerolog.Logger {
	return &logger
}
