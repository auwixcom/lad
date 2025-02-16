// Copyright (c) 2016 Uber Technologies, Inc.
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

package ladgrpc

import (
	"fmt"
	"testing"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
	"github.com/auwixcom/lad/ladtest/observer"

	"github.com/stretchr/testify/require"
)

func TestLoggerInfoExpected(t *testing.T) {
	checkMessages(t, ladcore.DebugLevel, nil, ladcore.InfoLevel, []string{
		"hello",
		"s1s21 2 3s34s56",
		"hello world",
		"",
		"foo",
		"foo bar",
		"s1 s2 1 2 3 s3 4 s5 6",
		"hello",
		"s1s21 2 3s34s56",
		"hello world",
		"",
		"foo",
		"foo bar",
		"s1 s2 1 2 3 s3 4 s5 6",
	}, func(logger *Logger) {
		logger.Info("hello")
		logger.Info("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Infof("%s world", "hello")
		logger.Infoln()
		logger.Infoln("foo")
		logger.Infoln("foo", "bar")
		logger.Infoln("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Print("hello")
		logger.Print("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Printf("%s world", "hello")
		logger.Println()
		logger.Println("foo")
		logger.Println("foo", "bar")
		logger.Println("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
	})
}

func TestLoggerDebugExpected(t *testing.T) {
	checkMessages(t, ladcore.DebugLevel, []Option{WithDebug()}, ladcore.DebugLevel, []string{
		"hello",
		"s1s21 2 3s34s56",
		"hello world",
		"",
		"foo",
		"foo bar",
		"s1 s2 1 2 3 s3 4 s5 6",
	}, func(logger *Logger) {
		logger.Print("hello")
		logger.Print("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Printf("%s world", "hello")
		logger.Println()
		logger.Println("foo")
		logger.Println("foo", "bar")
		logger.Println("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
	})
}

func TestLoggerDebugSuppressed(t *testing.T) {
	checkMessages(t, ladcore.InfoLevel, []Option{WithDebug()}, ladcore.DebugLevel, nil, func(logger *Logger) {
		logger.Print("hello")
		logger.Printf("%s world", "hello")
		logger.Println()
		logger.Println("foo")
		logger.Println("foo", "bar")
	})
}

func TestLoggerWarningExpected(t *testing.T) {
	checkMessages(t, ladcore.DebugLevel, nil, ladcore.WarnLevel, []string{
		"hello",
		"s1s21 2 3s34s56",
		"hello world",
		"",
		"foo",
		"foo bar",
		"s1 s2 1 2 3 s3 4 s5 6",
	}, func(logger *Logger) {
		logger.Warning("hello")
		logger.Warning("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Warningf("%s world", "hello")
		logger.Warningln()
		logger.Warningln("foo")
		logger.Warningln("foo", "bar")
		logger.Warningln("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
	})
}

func TestLoggerErrorExpected(t *testing.T) {
	checkMessages(t, ladcore.DebugLevel, nil, ladcore.ErrorLevel, []string{
		"hello",
		"s1s21 2 3s34s56",
		"hello world",
		"",
		"foo",
		"foo bar",
		"s1 s2 1 2 3 s3 4 s5 6",
	}, func(logger *Logger) {
		logger.Error("hello")
		logger.Error("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Errorf("%s world", "hello")
		logger.Errorln()
		logger.Errorln("foo")
		logger.Errorln("foo", "bar")
		logger.Errorln("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
	})
}

func TestLoggerFatalExpected(t *testing.T) {
	checkMessages(t, ladcore.DebugLevel, nil, ladcore.FatalLevel, []string{
		"hello",
		"s1s21 2 3s34s56",
		"hello world",
		"",
		"foo",
		"foo bar",
		"s1 s2 1 2 3 s3 4 s5 6",
	}, func(logger *Logger) {
		logger.Fatal("hello")
		logger.Fatal("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
		logger.Fatalf("%s world", "hello")
		logger.Fatalln()
		logger.Fatalln("foo")
		logger.Fatalln("foo", "bar")
		logger.Fatalln("s1", "s2", 1, 2, 3, "s3", 4, "s5", 6)
	})
}

func TestLoggerV(t *testing.T) {
	tests := []struct {
		ladLevel     ladcore.Level
		grpcEnabled  []int
		grpcDisabled []int
	}{
		{
			ladLevel:     ladcore.DebugLevel,
			grpcEnabled:  []int{grpcLvlInfo, grpcLvlWarn, grpcLvlError, grpcLvlFatal},
			grpcDisabled: []int{}, // everything is enabled, nothing is disabled
		},
		{
			ladLevel:     ladcore.InfoLevel,
			grpcEnabled:  []int{grpcLvlInfo, grpcLvlWarn, grpcLvlError, grpcLvlFatal},
			grpcDisabled: []int{}, // everything is enabled, nothing is disabled
		},
		{
			ladLevel:     ladcore.WarnLevel,
			grpcEnabled:  []int{grpcLvlWarn, grpcLvlError, grpcLvlFatal},
			grpcDisabled: []int{grpcLvlInfo},
		},
		{
			ladLevel:     ladcore.ErrorLevel,
			grpcEnabled:  []int{grpcLvlError, grpcLvlFatal},
			grpcDisabled: []int{grpcLvlInfo, grpcLvlWarn},
		},
		{
			ladLevel:     ladcore.DPanicLevel,
			grpcEnabled:  []int{grpcLvlFatal},
			grpcDisabled: []int{grpcLvlInfo, grpcLvlWarn, grpcLvlError},
		},
		{
			ladLevel:     ladcore.PanicLevel,
			grpcEnabled:  []int{grpcLvlFatal},
			grpcDisabled: []int{grpcLvlInfo, grpcLvlWarn, grpcLvlError},
		},
		{
			ladLevel:     ladcore.FatalLevel,
			grpcEnabled:  []int{grpcLvlFatal},
			grpcDisabled: []int{grpcLvlInfo, grpcLvlWarn, grpcLvlError},
		},
	}
	for _, tst := range tests {
		for _, grpcLvl := range tst.grpcEnabled {
			t.Run(fmt.Sprintf("enabled %s %d", tst.ladLevel, grpcLvl), func(t *testing.T) {
				checkLevel(t, tst.ladLevel, true, func(logger *Logger) bool {
					return logger.V(grpcLvl)
				})
			})
		}
		for _, grpcLvl := range tst.grpcDisabled {
			t.Run(fmt.Sprintf("disabled %s %d", tst.ladLevel, grpcLvl), func(t *testing.T) {
				checkLevel(t, tst.ladLevel, false, func(logger *Logger) bool {
					return logger.V(grpcLvl)
				})
			})
		}
	}
}

func checkLevel(
	t testing.TB,
	enab ladcore.LevelEnabler,
	expectedBool bool,
	f func(*Logger) bool,
) {
	withLogger(enab, nil, func(logger *Logger, observedLogs *observer.ObservedLogs) {
		actualBool := f(logger)
		if expectedBool {
			require.True(t, actualBool)
		} else {
			require.False(t, actualBool)
		}
	})
}

func checkMessages(
	t testing.TB,
	enab ladcore.LevelEnabler,
	opts []Option,
	expectedLevel ladcore.Level,
	expectedMessages []string,
	f func(*Logger),
) {
	if expectedLevel == ladcore.FatalLevel {
		expectedLevel = ladcore.WarnLevel
	}
	withLogger(enab, opts, func(logger *Logger, observedLogs *observer.ObservedLogs) {
		f(logger)
		logEntries := observedLogs.All()
		require.Equal(t, len(expectedMessages), len(logEntries))
		for i, logEntry := range logEntries {
			require.Equal(t, expectedLevel, logEntry.Level)
			require.Equal(t, expectedMessages[i], logEntry.Message)
		}
	})
}

func withLogger(
	enab ladcore.LevelEnabler,
	opts []Option,
	f func(*Logger, *observer.ObservedLogs),
) {
	core, observedLogs := observer.New(enab)
	f(NewLogger(lad.New(core), append(opts, withWarn())...), observedLogs)
}
