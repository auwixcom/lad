// Copyright (c) 2016-2022 Uber Technologies, Inc.
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

package observer_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"

	//revive:disable:dot-imports
	. "github.com/auwixcom/lad/ladtest/observer"
)

func assertEmpty(t testing.TB, logs *ObservedLogs) {
	assert.Equal(t, 0, logs.Len(), "Expected empty ObservedLogs to have zero length.")
	assert.Equal(t, []LoggedEntry{}, logs.All(), "Unexpected LoggedEntries in empty ObservedLogs.")
}

func TestObserver(t *testing.T) {
	observer, logs := New(lad.InfoLevel)
	assertEmpty(t, logs)

	t.Run("LevelOf", func(t *testing.T) {
		assert.Equal(t, lad.InfoLevel, ladcore.LevelOf(observer), "Observer reported the wrong log level.")
	})

	assert.NoError(t, observer.Sync(), "Unexpected failure in no-op Sync")

	obs := lad.New(observer).With(lad.Int("i", 1))
	obs.Info("foo")
	obs.Debug("bar")
	want := []LoggedEntry{{
		Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "foo"},
		Context: []ladcore.Field{lad.Int("i", 1)},
	}}

	assert.Equal(t, 1, logs.Len(), "Unexpected observed logs Len.")
	assert.Equal(t, want, logs.AllUntimed(), "Unexpected contents from AllUntimed.")

	all := logs.All()
	require.Equal(t, 1, len(all), "Unexpected number of LoggedEntries returned from All.")
	assert.NotEqual(t, time.Time{}, all[0].Time, "Expected non-zero time on LoggedEntry.")

	// copy & zero time for stable assertions
	untimed := append([]LoggedEntry{}, all...)
	untimed[0].Time = time.Time{}
	assert.Equal(t, want, untimed, "Unexpected LoggedEntries from All.")

	assert.Equal(t, all, logs.TakeAll(), "Expected All and TakeAll to return identical results.")
	assertEmpty(t, logs)
}

func TestObserverWith(t *testing.T) {
	sf1, logs := New(lad.InfoLevel)

	// need to pad out enough initial fields so that the underlying slice cap()
	// gets ahead of its len() so that the sf3/4 With append's could choose
	// not to copy (if the implementation doesn't force them)
	sf1 = sf1.With([]ladcore.Field{lad.Int("a", 1), lad.Int("b", 2)})

	sf2 := sf1.With([]ladcore.Field{lad.Int("c", 3)})
	sf3 := sf2.With([]ladcore.Field{lad.Int("d", 4)})
	sf4 := sf2.With([]ladcore.Field{lad.Int("e", 5)})
	ent := ladcore.Entry{Level: lad.InfoLevel, Message: "hello"}

	for i, core := range []ladcore.Core{sf2, sf3, sf4} {
		if ce := core.Check(ent, nil); ce != nil {
			ce.Write(lad.Int("i", i))
		}
	}

	assert.Equal(t, []LoggedEntry{
		{
			Entry: ent,
			Context: []ladcore.Field{
				lad.Int("a", 1),
				lad.Int("b", 2),
				lad.Int("c", 3),
				lad.Int("i", 0),
			},
		},
		{
			Entry: ent,
			Context: []ladcore.Field{
				lad.Int("a", 1),
				lad.Int("b", 2),
				lad.Int("c", 3),
				lad.Int("d", 4),
				lad.Int("i", 1),
			},
		},
		{
			Entry: ent,
			Context: []ladcore.Field{
				lad.Int("a", 1),
				lad.Int("b", 2),
				lad.Int("c", 3),
				lad.Int("e", 5),
				lad.Int("i", 2),
			},
		},
	}, logs.All(), "expected no field sharing between With siblings")
}

func TestFilters(t *testing.T) {
	logs := []LoggedEntry{
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "log a"},
			Context: []ladcore.Field{lad.String("fStr", "1"), lad.Int("a", 1)},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "log a"},
			Context: []ladcore.Field{lad.String("fStr", "2"), lad.Int("b", 2)},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "log b"},
			Context: []ladcore.Field{lad.Int("a", 1), lad.Int("b", 2)},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "log c"},
			Context: []ladcore.Field{lad.Int("a", 1), lad.Namespace("ns"), lad.Int("a", 2)},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "msg 1"},
			Context: []ladcore.Field{lad.Int("a", 1), lad.Namespace("ns")},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "any map"},
			Context: []ladcore.Field{lad.Any("map", map[string]string{"a": "b"})},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "any slice"},
			Context: []ladcore.Field{lad.Any("slice", []string{"a"})},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "msg 2"},
			Context: []ladcore.Field{lad.Int("b", 2), lad.Namespace("filterMe")},
		},
		{
			Entry:   ladcore.Entry{Level: lad.InfoLevel, Message: "any slice"},
			Context: []ladcore.Field{lad.Any("filterMe", []string{"b"})},
		},
		{
			Entry:   ladcore.Entry{Level: lad.WarnLevel, Message: "danger will robinson"},
			Context: []ladcore.Field{lad.Int("b", 42)},
		},
		{
			Entry:   ladcore.Entry{Level: lad.ErrorLevel, Message: "warp core breach"},
			Context: []ladcore.Field{lad.Int("b", 42)},
		},
		{
			Entry:   ladcore.Entry{Level: lad.ErrorLevel, Message: "msg", LoggerName: "my.logger"},
			Context: []ladcore.Field{lad.Int("b", 42)},
		},
	}

	logger, sink := New(lad.InfoLevel)
	for _, log := range logs {
		assert.NoError(t, logger.Write(log.Entry, log.Context), "Unexpected error writing log entry.")
	}

	tests := []struct {
		msg      string
		filtered *ObservedLogs
		want     []LoggedEntry
	}{
		{
			msg:      "filter by message",
			filtered: sink.FilterMessage("log a"),
			want:     logs[0:2],
		},
		{
			msg:      "filter by field",
			filtered: sink.FilterField(lad.String("fStr", "1")),
			want:     logs[0:1],
		},
		{
			msg:      "filter by message and field",
			filtered: sink.FilterMessage("log a").FilterField(lad.Int("b", 2)),
			want:     logs[1:2],
		},
		{
			msg:      "filter by field with duplicate fields",
			filtered: sink.FilterField(lad.Int("a", 2)),
			want:     logs[3:4],
		},
		{
			msg:      "filter doesn't match any messages",
			filtered: sink.FilterMessage("no match"),
			want:     []LoggedEntry{},
		},
		{
			msg:      "filter by snippet",
			filtered: sink.FilterMessageSnippet("log"),
			want:     logs[0:4],
		},
		{
			msg:      "filter by snippet and field",
			filtered: sink.FilterMessageSnippet("a").FilterField(lad.Int("b", 2)),
			want:     logs[1:2],
		},
		{
			msg:      "filter for map",
			filtered: sink.FilterField(lad.Any("map", map[string]string{"a": "b"})),
			want:     logs[5:6],
		},
		{
			msg:      "filter for slice",
			filtered: sink.FilterField(lad.Any("slice", []string{"a"})),
			want:     logs[6:7],
		},
		{
			msg:      "filter field key",
			filtered: sink.FilterFieldKey("filterMe"),
			want:     logs[7:9],
		},
		{
			msg: "filter by arbitrary function",
			filtered: sink.Filter(func(e LoggedEntry) bool {
				return len(e.Context) > 1
			}),
			want: func() []LoggedEntry {
				// Do not modify logs slice.
				w := make([]LoggedEntry, 0, len(logs))
				w = append(w, logs[0:5]...)
				w = append(w, logs[7])
				return w
			}(),
		},
		{
			msg:      "filter level",
			filtered: sink.FilterLevelExact(lad.WarnLevel),
			want:     logs[9:10],
		},
		{
			msg:      "filter logger name",
			filtered: sink.FilterLoggerName("my.logger"),
			want:     logs[11:12],
		},
	}

	for _, tt := range tests {
		got := tt.filtered.AllUntimed()
		assert.Equal(t, tt.want, got, tt.msg)
	}
}
