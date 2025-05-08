package ladglobal

import (
	"testing"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
)

func TestNew(t *testing.T) {
	// Initialize global logger: console Info (color), file Warn (rotation)
	New(
		WithConsole(ladcore.InfoLevel, true, ""),
		WithFile(FileConfig{
			Level:      ladcore.WarnLevel,
			Filename:   "app.log",
			MaxSizeMB:  100,
			MaxBackups: 5,
			MaxAgeDays: 7,
			Compress:   true,
		}),
		WithCaller(),
	)

	// Use global logger
	lad.S().Info("Service started successfully")
	lad.S().Warn("This is a warning log")
}
