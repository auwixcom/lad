// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ladtest

import (
	"bytes"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
)

// LoggerOption configures the test logger built by NewLogger.
type LoggerOption interface {
	applyLoggerOption(*loggerOptions)
}

type loggerOptions struct {
	Level      ladcore.LevelEnabler
	ladOptions []lad.Option
}

type loggerOptionFunc func(*loggerOptions)

func (f loggerOptionFunc) applyLoggerOption(opts *loggerOptions) {
	f(opts)
}

// Level controls which messages are logged by a test Logger built by
// NewLogger.
func Level(enab ladcore.LevelEnabler) LoggerOption {
	return loggerOptionFunc(func(opts *loggerOptions) {
		opts.Level = enab
	})
}

// WrapOptions adds lad.Option's to a test Logger built by NewLogger.
func WrapOptions(ladOpts ...lad.Option) LoggerOption {
	return loggerOptionFunc(func(opts *loggerOptions) {
		opts.ladOptions = ladOpts
	})
}

// NewLogger builds a new Logger that logs all messages to the given
// testing.TB.
//
//	logger := ladtest.NewLogger(t)
//
// Use this with a *testing.T or *testing.B to get logs which get printed only
// if a test fails or if you ran go test -v.
//
// The returned logger defaults to logging debug level messages and above.
// This may be changed by passing a ladtest.Level during construction.
//
//	logger := ladtest.NewLogger(t, ladtest.Level(lad.WarnLevel))
//
// You may also pass lad.Option's to customize test logger.
//
//	logger := ladtest.NewLogger(t, ladtest.WrapOptions(lad.AddCaller()))
func NewLogger(t TestingT, opts ...LoggerOption) *lad.Logger {
	cfg := loggerOptions{
		Level: ladcore.DebugLevel,
	}
	for _, o := range opts {
		o.applyLoggerOption(&cfg)
	}

	writer := NewTestingWriter(t)
	ladOptions := []lad.Option{
		// Send lad errors to the same writer and mark the test as failed if
		// that happens.
		lad.ErrorOutput(writer.WithMarkFailed(true)),
	}
	ladOptions = append(ladOptions, cfg.ladOptions...)

	return lad.New(
		ladcore.NewCore(
			ladcore.NewConsoleEncoder(lad.NewDevelopmentEncoderConfig()),
			writer,
			cfg.Level,
		),
		ladOptions...,
	)
}

// TestingWriter is a WriteSyncer that writes to the given testing.TB.
type TestingWriter struct {
	t TestingT

	// If true, the test will be marked as failed if this TestingWriter is
	// ever used.
	markFailed bool
}

// NewTestingWriter builds a new TestingWriter that writes to the given
// testing.TB.
//
// Use this if you need more flexibility when creating *lad.Logger
// than ladtest.NewLogger() provides.
//
// E.g., if you want to use custom core with ladtest.TestingWriter:
//
//	encoder := newCustomEncoder()
//	writer := ladtest.NewTestingWriter(t)
//	level := lad.NewAtomicLevelAt(ladcore.DebugLevel)
//
//	core := newCustomCore(encoder, writer, level)
//
//	logger := lad.New(core, lad.AddCaller())
func NewTestingWriter(t TestingT) TestingWriter {
	return TestingWriter{t: t}
}

// WithMarkFailed returns a copy of this TestingWriter with markFailed set to
// the provided value.
func (w TestingWriter) WithMarkFailed(v bool) TestingWriter {
	w.markFailed = v
	return w
}

// Write writes bytes from p to the underlying testing.TB.
func (w TestingWriter) Write(p []byte) (n int, err error) {
	n = len(p)

	// Strip trailing newline because t.Log always adds one.
	p = bytes.TrimRight(p, "\n")

	// Note: t.Log is safe for concurrent use.
	w.t.Logf("%s", p)
	if w.markFailed {
		w.t.Fail()
	}

	return n, nil
}

// Sync commits the current contents (a no-op for TestingWriter).
func (w TestingWriter) Sync() error {
	return nil
}
