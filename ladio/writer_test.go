// Copyright (c) 2021 Uber Technologies, Inc.
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

package ladio

import (
	"io"
	"testing"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
	"github.com/auwixcom/lad/ladtest/observer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		level  ladcore.Level // defaults to info
		writes []string
		want   []ladcore.Entry
	}{
		{
			desc: "simple",
			writes: []string{
				"foo\n",
				"bar\n",
				"baz\n",
			},
			want: []ladcore.Entry{
				{Level: lad.InfoLevel, Message: "foo"},
				{Level: lad.InfoLevel, Message: "bar"},
				{Level: lad.InfoLevel, Message: "baz"},
			},
		},
		{
			desc:  "level too low",
			level: lad.DebugLevel,
			writes: []string{
				"foo\n",
				"bar\n",
			},
			want: []ladcore.Entry{},
		},
		{
			desc:  "multiple newlines in a message",
			level: lad.WarnLevel,
			writes: []string{
				"foo\nbar\n",
				"baz\n",
				"qux\nquux\n",
			},
			want: []ladcore.Entry{
				{Level: lad.WarnLevel, Message: "foo"},
				{Level: lad.WarnLevel, Message: "bar"},
				{Level: lad.WarnLevel, Message: "baz"},
				{Level: lad.WarnLevel, Message: "qux"},
				{Level: lad.WarnLevel, Message: "quux"},
			},
		},
		{
			desc:  "message split across multiple writes",
			level: lad.ErrorLevel,
			writes: []string{
				"foo",
				"bar\nbaz",
				"qux",
			},
			want: []ladcore.Entry{
				{Level: lad.ErrorLevel, Message: "foobar"},
				{Level: lad.ErrorLevel, Message: "bazqux"},
			},
		},
		{
			desc: "blank lines in the middle",
			writes: []string{
				"foo\n\nbar\nbaz",
			},
			want: []ladcore.Entry{
				{Level: lad.InfoLevel, Message: "foo"},
				{Level: lad.InfoLevel, Message: ""},
				{Level: lad.InfoLevel, Message: "bar"},
				{Level: lad.InfoLevel, Message: "baz"},
			},
		},
		{
			desc: "blank line at the end",
			writes: []string{
				"foo\nbar\nbaz\n",
			},
			want: []ladcore.Entry{
				{Level: lad.InfoLevel, Message: "foo"},
				{Level: lad.InfoLevel, Message: "bar"},
				{Level: lad.InfoLevel, Message: "baz"},
			},
		},
		{
			desc: "multiple blank line at the end",
			writes: []string{
				"foo\nbar\nbaz\n\n",
			},
			want: []ladcore.Entry{
				{Level: lad.InfoLevel, Message: "foo"},
				{Level: lad.InfoLevel, Message: "bar"},
				{Level: lad.InfoLevel, Message: "baz"},
				{Level: lad.InfoLevel, Message: ""},
			},
		},
	}

	for _, tt := range tests {
		tt := tt // for t.Parallel
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			core, observed := observer.New(lad.InfoLevel)

			w := Writer{
				Log:   lad.New(core),
				Level: tt.level,
			}

			for _, s := range tt.writes {
				_, err := io.WriteString(&w, s)
				require.NoError(t, err, "Writer.Write failed.")
			}

			assert.NoError(t, w.Close(), "Writer.Close failed.")

			// Turn []observer.LoggedEntry => []ladcore.Entry
			got := make([]ladcore.Entry, observed.Len())
			for i, ent := range observed.AllUntimed() {
				got[i] = ent.Entry
			}
			assert.Equal(t, tt.want, got, "Logged entries do not match.")
		})
	}
}

func TestWrite_Sync(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(lad.InfoLevel)

	w := Writer{
		Log:   lad.New(core),
		Level: lad.InfoLevel,
	}

	io.WriteString(&w, "foo")
	io.WriteString(&w, "bar")

	t.Run("no sync", func(t *testing.T) {
		assert.Zero(t, observed.Len(), "Expected no logs yet")
	})

	t.Run("sync", func(t *testing.T) {
		defer observed.TakeAll()

		require.NoError(t, w.Sync(), "Sync must not fail")

		assert.Equal(t, []observer.LoggedEntry{
			{Entry: ladcore.Entry{Message: "foobar"}, Context: []ladcore.Field{}},
		}, observed.AllUntimed(), "Log messages did not match")
	})

	t.Run("sync on empty", func(t *testing.T) {
		require.NoError(t, w.Sync(), "Sync must not fail")
		assert.Zero(t, observed.Len(), "Expected no logs yet")
	})
}

func BenchmarkWriter(b *testing.B) {
	tests := []struct {
		name   string
		writes [][]byte
	}{
		{
			name: "single",
			writes: [][]byte{
				[]byte("foobar\n"),
				[]byte("bazqux\n"),
			},
		},
		{
			name: "splits",
			writes: [][]byte{
				[]byte("foo"),
				[]byte("bar\nbaz"),
				[]byte("qux\n"),
			},
		},
	}

	writer := Writer{
		Log:   lad.New(new(partiallyNopCore)),
		Level: ladcore.DebugLevel,
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for _, bs := range tt.writes {
					writer.Write(bs)
				}
			}
		})
	}
}

// partiallyNopCore behaves exactly like NopCore except it always returns true
// for whether the provided level is enabled, and accepts all Check requests.
//
// This lets us measure the overhead of the writer without measuring the cost
// of logging.
type partiallyNopCore struct{}

func (*partiallyNopCore) Enabled(ladcore.Level) bool { return true }

func (c *partiallyNopCore) Check(ent ladcore.Entry, ce *ladcore.CheckedEntry) *ladcore.CheckedEntry {
	return ce.AddCore(ent, c)
}

func (c *partiallyNopCore) With([]ladcore.Field) ladcore.Core        { return c }
func (*partiallyNopCore) Write(ladcore.Entry, []ladcore.Field) error { return nil }
func (*partiallyNopCore) Sync() error                                { return nil }
