package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Logger zerolog.Logger

func Init(environment string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if environment == "production" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}

	log.Logger = Logger
}

func Info() *zerolog.Event {
	return Logger.Info()
}

func Debug() *zerolog.Event {
	return Logger.Debug()
}

func Warn() *zerolog.Event {
	return Logger.Warn()
}

func Error() *zerolog.Event {
	return Logger.Error()
}

func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

func WithField(key string, value interface{}) zerolog.Logger {
	return Logger.With().Interface(key, value).Logger()
}

func WithFields(fields map[string]interface{}) zerolog.Logger {
	ctx := Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}
