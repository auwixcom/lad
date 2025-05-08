package ladglobal

import (
	"os"
	"time"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Option configures the logger behavior.
type Option func(*Config)

// Config holds the configured cores and caller flag.
type Config struct {
	cores  []ladcore.Core
	caller bool
}

// FileConfig groups parameters for file output.
type FileConfig struct {
	Level      ladcore.Level // log level
	Filename   string        // log file path
	MaxSizeMB  int           // max size in MB
	MaxBackups int           // max number of backups
	MaxAgeDays int           // retention days
	Compress   bool          // compress old logs
}

// WithConsole adds a console core to the logger.
// level: log level; enableColor: true to output colored level; timeFormat: timestamp format or empty for default.
func WithConsole(level ladcore.Level, enableColor bool, timeFormat string) Option {
	return func(cfg *Config) {
		encCfg := lad.NewProductionEncoderConfig()
		// time formatting
		tf := timeFormat
		if tf == "" {
			tf = "2006-01-02 15:04:05.000"
		}
		encCfg.EncodeTime = func(t time.Time, pae ladcore.PrimitiveArrayEncoder) {
			pae.AppendString(t.Format(tf))
		}
		// level encoding
		if enableColor {
			encCfg.EncodeLevel = ladcore.CapitalColorLevelEncoder
		}
		core := ladcore.NewCore(
			ladcore.NewConsoleEncoder(encCfg),
			ladcore.AddSync(os.Stdout),
			level,
		)
		cfg.cores = append(cfg.cores, core)
	}
}

// WithFile adds a rotating file core to the logger using a FileConfig struct.
func WithFile(fc FileConfig) Option {
	return func(cfg *Config) {
		hook := &lumberjack.Logger{
			Filename:   fc.Filename,
			MaxSize:    fc.MaxSizeMB,
			MaxBackups: fc.MaxBackups,
			MaxAge:     fc.MaxAgeDays,
			Compress:   fc.Compress,
		}

		encCfg := lad.NewProductionEncoderConfig()
		// default timestamp format
		encCfg.EncodeTime = func(t time.Time, pae ladcore.PrimitiveArrayEncoder) {
			pae.AppendString(t.Format("2006-01-02 15:04:05.000"))
		}
		encCfg.EncodeLevel = ladcore.CapitalLevelEncoder

		core := ladcore.NewCore(
			ladcore.NewConsoleEncoder(encCfg), // or JSONEncoder if preferred
			ladcore.AddSync(hook),
			fc.Level,
		)
		cfg.cores = append(cfg.cores, core)
	}
}

// WithCaller enables adding the caller information to logs.
func WithCaller() Option {
	return func(cfg *Config) {
		cfg.caller = true
	}
}

// New configures and replaces the global logger based on the provided options.
// If no cores are added, defaults to a console core at DebugLevel.
func New(opts ...Option) {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if len(cfg.cores) == 0 {
		// default console core
		WithConsole(lad.DebugLevel, true, "")(cfg)
	}

	// combine cores
	core := ladcore.NewTee(cfg.cores...)
	zapOpts := []lad.Option{}
	if cfg.caller {
		zapOpts = append(zapOpts, lad.AddCaller())
	}
	logger := lad.New(core, zapOpts...)
	lad.ReplaceGlobals(logger)
}
