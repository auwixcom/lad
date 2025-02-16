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
	"bytes"
	"testing"
	"time"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
	"github.com/auwixcom/lad/ladtest"
	"github.com/stretchr/testify/assert"
)

func TestCheckedEntryIllegalReuse(t *testing.T) {
	t.Parallel()

	var errOut bytes.Buffer

	testCore := ladtest.NewLogger(t).Core()
	ce := testCore.Check(ladcore.Entry{
		Level:   ladcore.InfoLevel,
		Time:    time.Now(),
		Message: "hello",
	}, nil)
	ce.ErrorOutput = ladcore.AddSync(&errOut)

	// The first write should succeed.
	ce.Write(lad.String("k", "v"), lad.Int("n", 42))
	assert.Empty(t, errOut.String(), "Expected no errors on first write.")

	// The second write should fail.
	ce.Write(lad.String("foo", "bar"), lad.Int("x", 1))
	assert.Contains(t, errOut.String(), "Unsafe CheckedEntry re-use near Entry",
		"Expected error logged on second write.")
}
