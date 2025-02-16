// Copyright (c) 2018 Uber Technologies, Inc.
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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
)

// TestJSONEncodeEntry is an more "integrated" test that makes it easier to get
// coverage on the json encoder (e.g. putJSONEncoder, let alone EncodeEntry
// itself) than the tests in json_encoder_impl_test.go; it needs to be in the
// ladcore_test package, so that it can import the toplevel lad package for
// field constructor convenience.
func TestJSONEncodeEntry(t *testing.T) {
	type bar struct {
		Key string  `json:"key"`
		Val float64 `json:"val"`
	}

	type foo struct {
		A string  `json:"aee"`
		B int     `json:"bee"`
		C float64 `json:"cee"`
		D []bar   `json:"dee"`
	}

	tests := []struct {
		desc     string
		expected string
		ent      ladcore.Entry
		fields   []ladcore.Field
	}{
		{
			desc: "info entry with some fields",
			expected: `{
				"L": "info",
				"T": "2018-06-19T16:33:42.000Z",
				"N": "bob",
				"M": "lob law",
				"so": "passes",
				"answer": 42,
				"a_float32": 2.71,
				"common_pie": 3.14,
				"complex_value": "3.14-2.71i",
				"null_value": null,
				"array_with_null_elements": [{}, null, null, 2],
				"such": {
					"aee": "lol",
					"bee": 123,
					"cee": 0.9999,
					"dee": [
						{"key": "pi", "val": 3.141592653589793},
						{"key": "tau", "val": 6.283185307179586}
					]
				}
			}`,
			ent: ladcore.Entry{
				Level:      ladcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "lob law",
			},
			fields: []ladcore.Field{
				lad.String("so", "passes"),
				lad.Int("answer", 42),
				lad.Float64("common_pie", 3.14),
				lad.Float32("a_float32", 2.71),
				lad.Complex128("complex_value", 3.14-2.71i),
				// Cover special-cased handling of nil in AddReflect() and
				// AppendReflect(). Note that for the latter, we explicitly test
				// correct results for both the nil static interface{} value
				// (`nil`), as well as the non-nil interface value with a
				// dynamic type and nil value (`(*struct{})(nil)`).
				lad.Reflect("null_value", nil),
				lad.Reflect("array_with_null_elements", []interface{}{&struct{}{}, nil, (*struct{})(nil), 2}),
				lad.Reflect("such", foo{
					A: "lol",
					B: 123,
					C: 0.9999,
					D: []bar{
						{"pi", 3.141592653589793},
						{"tau", 6.283185307179586},
					},
				}),
			},
		},
		{
			desc: "zero_time_omitted",
			expected: `{
				"L": "info",
				"N": "name",
				"M": "message"
			}`,
			ent: ladcore.Entry{
				Level:      ladcore.InfoLevel,
				Time:       time.Time{},
				LoggerName: "name",
				Message:    "message",
			},
		},
	}

	enc := ladcore.NewJSONEncoder(ladcore.EncoderConfig{
		MessageKey:     "M",
		LevelKey:       "L",
		TimeKey:        "T",
		NameKey:        "N",
		CallerKey:      "C",
		FunctionKey:    "F",
		StacktraceKey:  "S",
		EncodeLevel:    ladcore.LowercaseLevelEncoder,
		EncodeTime:     ladcore.ISO8601TimeEncoder,
		EncodeDuration: ladcore.SecondsDurationEncoder,
		EncodeCaller:   ladcore.ShortCallerEncoder,
	})

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			buf, err := enc.EncodeEntry(tt.ent, tt.fields)
			if assert.NoError(t, err, "Unexpected JSON encoding error.") {
				assert.JSONEq(t, tt.expected, buf.String(), "Incorrect encoded JSON entry.")
			}
			buf.Free()
		})
	}
}

func TestNoEncodeLevelSupplied(t *testing.T) {
	enc := ladcore.NewJSONEncoder(ladcore.EncoderConfig{
		MessageKey:     "M",
		LevelKey:       "L",
		TimeKey:        "T",
		NameKey:        "N",
		CallerKey:      "C",
		FunctionKey:    "F",
		StacktraceKey:  "S",
		EncodeTime:     ladcore.ISO8601TimeEncoder,
		EncodeDuration: ladcore.SecondsDurationEncoder,
		EncodeCaller:   ladcore.ShortCallerEncoder,
	})

	ent := ladcore.Entry{
		Level:      ladcore.InfoLevel,
		Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
		LoggerName: "bob",
		Message:    "lob law",
	}

	fields := []ladcore.Field{
		lad.Int("answer", 42),
	}

	_, err := enc.EncodeEntry(ent, fields)
	assert.NoError(t, err, "Unexpected JSON encoding error.")
}

func TestJSONEmptyConfig(t *testing.T) {
	tests := []struct {
		name     string
		field    ladcore.Field
		expected string
	}{
		{
			name:     "time",
			field:    lad.Time("foo", time.Unix(1591287718, 0)), // 2020-06-04 09:21:58 -0700 PDT
			expected: `{"foo": 1591287718000000000}`,
		},
		{
			name:     "duration",
			field:    lad.Duration("bar", time.Microsecond),
			expected: `{"bar": 1000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := ladcore.NewJSONEncoder(ladcore.EncoderConfig{})

			buf, err := enc.EncodeEntry(ladcore.Entry{
				Level:      ladcore.DebugLevel,
				Time:       time.Now(),
				LoggerName: "mylogger",
				Message:    "things happened",
			}, []ladcore.Field{tt.field})
			if assert.NoError(t, err, "Unexpected JSON encoding error.") {
				assert.JSONEq(t, tt.expected, buf.String(), "Incorrect encoded JSON entry.")
			}

			buf.Free()
		})
	}
}

// Encodes any object into empty json '{}'
type emptyReflectedEncoder struct {
	writer io.Writer
}

func (enc *emptyReflectedEncoder) Encode(obj interface{}) error {
	_, err := enc.writer.Write([]byte("{}"))
	return err
}

func TestJSONCustomReflectedEncoder(t *testing.T) {
	tests := []struct {
		name     string
		field    ladcore.Field
		expected string
	}{
		{
			name: "encode custom map object",
			field: ladcore.Field{
				Key:  "data",
				Type: ladcore.ReflectType,
				Interface: map[string]interface{}{
					"foo": "hello",
					"bar": 1111,
				},
			},
			expected: `{"data":{}}`,
		},
		{
			name: "encode nil object",
			field: ladcore.Field{
				Key:  "data",
				Type: ladcore.ReflectType,
			},
			expected: `{"data":null}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enc := ladcore.NewJSONEncoder(ladcore.EncoderConfig{
				NewReflectedEncoder: func(writer io.Writer) ladcore.ReflectedEncoder {
					return &emptyReflectedEncoder{
						writer: writer,
					}
				},
			})

			buf, err := enc.EncodeEntry(ladcore.Entry{
				Level:      ladcore.DebugLevel,
				Time:       time.Now(),
				LoggerName: "logger",
				Message:    "things happened",
			}, []ladcore.Field{tt.field})
			if assert.NoError(t, err, "Unexpected JSON encoding error.") {
				assert.JSONEq(t, tt.expected, buf.String(), "Incorrect encoded JSON entry.")
			}
			buf.Free()
		})
	}
}
