package logger

import (
	"strings"

	"cassette/core/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var Log zerolog.Logger

func init() {
	lumberjackLogger := utils.NewLumberjackLogger("cassette.log")
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	level, invalidLevel := appLogLevel(utils.GetConfig().LogLevel)
	Log = zerolog.New(lumberjackLogger).Level(level).With().Timestamp().Caller().Stack().Logger()
	if invalidLevel != "" {
		Log.Error().Str("configured_level", invalidLevel).Str("fallback_level", zerolog.ErrorLevel.String()).Msg("invalid app log level")
	}
	Log.Debug().Msg("logger initialized")
	Log.Debug().Msgf("config directory: %s", utils.SafeGetConfigDir())
}

func appLogLevel(raw string) (zerolog.Level, string) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return zerolog.ErrorLevel, ""
	}
	level, err := zerolog.ParseLevel(normalized)
	if err != nil {
		return zerolog.ErrorLevel, raw
	}
	return level, ""
}
