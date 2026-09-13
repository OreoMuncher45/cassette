package v1

import (
	"cassette/core/logger"
	"cassette/ui/v1/app"
)

func RunTui() {
	if err := app.Run(); err != nil {
		logger.Log.Error().Err(err).Msg("failed to run program")
	}
}
