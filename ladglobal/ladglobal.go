package ladglobal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type GlobalLogger interface {
	mode() ladcore.Core
}

const timeFormat = "2006-01-02 15:04:05.000"

type Console struct {
	Level      ladcore.Level
	TimeFormat string
}

func (c *Console) mode() ladcore.Core {
	write := ladcore.AddSync(io.MultiWriter(os.Stdout))
	config := lad.NewProductionEncoderConfig()
	config.EncodeTime = func(t time.Time, pae ladcore.PrimitiveArrayEncoder) {
		if c.TimeFormat == "" {
			pae.AppendString(t.Format(timeFormat))
		} else {
			pae.AppendString(t.Format(c.TimeFormat))
		}
	}

	config.EncodeLevel = ladcore.CapitalColorLevelEncoder
	return ladcore.NewCore(
		ladcore.NewConsoleEncoder(config),
		write,
		c.Level,
	)
}

func defaultConsole() ladcore.Core {
	c := &Console{
		Level:      lad.DebugLevel,
		TimeFormat: timeFormat,
	}

	return c.mode()
}

func DefaultConsole() {
	lad.ReplaceGlobals(lad.New(defaultConsole(), lad.AddCaller()))
}

type File struct {
	// Log level.
	LadLevel ladcore.Level
	// Date format.
	TimeFormat string
	// Log file name.
	Filename string
	// Maximum log file size, default is 100MB.
	MaxSize int
	// Maximum number of backups.
	MaxBackups int
	// Maximum retention time for logs.
	MaxAge int
	// Whether to compress and pack logs.
	Compress bool
}

func (f *File) mode() ladcore.Core {
	hook := &lumberjack.Logger{
		Filename:   f.Filename,
		MaxSize:    f.MaxSize,
		MaxBackups: f.MaxBackups,
		MaxAge:     f.MaxAge,
		Compress:   f.Compress,
	}
	write := ladcore.AddSync(io.MultiWriter(hook))
	config := lad.NewProductionEncoderConfig()
	config.EncodeTime = func(t time.Time, pae ladcore.PrimitiveArrayEncoder) {
		if f.TimeFormat == "" {
			pae.AppendString(t.Format("2006-01-02 15:04:05.000"))
		} else {
			pae.AppendString(t.Format(f.TimeFormat))
		}

	}
	return ladcore.NewCore(
		ladcore.NewConsoleEncoder(config),
		write,
		f.LadLevel,
	)
}

func defaultFile() ladcore.Core {
	var filename string
	execPath, err := os.Executable()
	if err != nil {
		fmt.Println("warn: Failed to get the executable file path:", err)
		filename = "lad.log"
	} else {
		filename = filepath.Base(execPath) + ".log"
	}

	f := &File{
		Filename:   filename,
		LadLevel:   lad.DebugLevel,
		MaxSize:    64,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}

	return f.mode()
}

func DefaultFile() {
	core := ladcore.NewTee(defaultFile())
	lad.ReplaceGlobals(lad.New(core, lad.AddCaller()))
}

func Default() {
	var cores []ladcore.Core
	cores = append(cores, defaultConsole(), defaultFile())

	core := ladcore.NewTee(cores...)

	lad.ReplaceGlobals(lad.New(core, lad.AddCaller()))
}

func New(globalLogger ...GlobalLogger) {
	var cores []ladcore.Core
	if len(globalLogger) == 0 {
		cores = append(cores, (&Console{
			Level: lad.DebugLevel,
		}).mode())
	} else {
		for _, v := range globalLogger {
			cores = append(cores, v.mode())
		}
	}

	core := ladcore.NewTee(cores...)

	lad.ReplaceGlobals(lad.New(core, lad.AddCaller()))
}
