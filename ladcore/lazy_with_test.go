// Copyright (c) 2023 Uber Technologies, Inc.
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

package ladcore_test

import (
	"sync/atomic"
	"testing"

	"github.com/auwixcom/lad/ladcore"
	"github.com/auwixcom/lad/ladtest/observer"
	"github.com/stretchr/testify/assert"
)

type proxyCore struct {
	ladcore.Core

	withCount  atomic.Int64
	checkCount atomic.Int64
}

func newProxyCore(inner ladcore.Core) *proxyCore {
	return &proxyCore{Core: inner}
}

func (p *proxyCore) With(fields []ladcore.Field) ladcore.Core {
	p.withCount.Add(1)
	return p.Core.With(fields)
}

func (p *proxyCore) Check(e ladcore.Entry, ce *ladcore.CheckedEntry) *ladcore.CheckedEntry {
	p.checkCount.Add(1)
	return p.Core.Check(e, ce)
}

func withLazyCore(f func(ladcore.Core, *proxyCore, *observer.ObservedLogs), initialFields ...ladcore.Field) {
	infoLogger, infoLogs := observer.New(ladcore.InfoLevel)
	proxyCore := newProxyCore(infoLogger)
	lazyCore := ladcore.NewLazyWith(proxyCore, initialFields)
	f(lazyCore, proxyCore, infoLogs)
}

func TestLazyCore(t *testing.T) {
	tests := []struct {
		name          string
		entries       []ladcore.Entry
		initialFields []ladcore.Field
		withChains    [][]ladcore.Field
		wantLogs      []observer.LoggedEntry
	}{
		{
			name:     "no logging, no with, inner core with never called, inner core check never called",
			wantLogs: []observer.LoggedEntry{},
		},
		{
			name: "2 logs, 1 dropped, no with, inner core with called once, inner core check never called",
			entries: []ladcore.Entry{
				{Level: ladcore.DebugLevel, Message: "log-at-debug"},
				{Level: ladcore.WarnLevel, Message: "log-at-warn"},
			},
			wantLogs: []observer.LoggedEntry{
				{
					Entry: ladcore.Entry{
						Level:   ladcore.WarnLevel,
						Message: "log-at-warn",
					},
					Context: []ladcore.Field{},
				},
			},
		},
		{
			name: "no logs, 2-chained with, inner core with called once, inner core check never called",
			withChains: [][]ladcore.Field{
				{makeInt64Field("a", 11), makeInt64Field("b", 22)},
				{makeInt64Field("c", 33), makeInt64Field("d", 44)},
			},
			wantLogs: []observer.LoggedEntry{},
		},
		{
			name: "2 logs, 1 dropped, 2-chained with, inner core with called once, inner core check never called",
			entries: []ladcore.Entry{
				{Level: ladcore.DebugLevel, Message: "log-at-debug"},
				{Level: ladcore.WarnLevel, Message: "log-at-warn"},
			},
			withChains: [][]ladcore.Field{
				{makeInt64Field("a", 11), makeInt64Field("b", 22)},
				{makeInt64Field("c", 33), makeInt64Field("d", 44)},
			},
			wantLogs: []observer.LoggedEntry{
				{
					Entry: ladcore.Entry{
						Level:   ladcore.WarnLevel,
						Message: "log-at-warn",
					},
					Context: []ladcore.Field{
						makeInt64Field("a", 11),
						makeInt64Field("b", 22),
						makeInt64Field("c", 33),
						makeInt64Field("d", 44),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withLazyCore(func(lazy ladcore.Core, proxy *proxyCore, logs *observer.ObservedLogs) {
				checkCounts := func(withCount int64, msg string) {
					assert.Equal(t, withCount, proxy.withCount.Load(), msg)
				}
				checkCounts(0, "expected no with calls because the logger is not used yet")

				for _, chain := range tt.withChains {
					lazy = lazy.With(chain)
				}
				if len(tt.withChains) > 0 {
					checkCounts(1, "expected with calls because the logger was with-chained")
				} else {
					checkCounts(0, "expected no with calls because the logger is not used yet")
				}

				for _, ent := range tt.entries {
					if ce := lazy.Check(ent, nil); ce != nil {
						ce.Write()
					}
				}
				if len(tt.entries) > 0 || len(tt.withChains) > 0 {
					checkCounts(1, "expected with calls because the logger had entries or with chains")
				} else {
					checkCounts(0, "expected no with calls because the logger is not used yet")
				}
				assert.Zero(t, proxy.checkCount.Load(), "expected no check calls because the inner core is copied")
				assert.Equal(t, tt.wantLogs, logs.AllUntimed())
			}, tt.initialFields...)
		})
	}
}
