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

package lad_test

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/auwixcom/lad"
	"github.com/auwixcom/lad/ladcore"
)

func Example_presets() {
	// Using lad's preset constructors is the simplest way to get a feel for the
	// package, but they don't allow much customization.
	logger := lad.NewExample() // or NewProduction, or NewDevelopment
	defer logger.Sync()

	const url = "http://example.com"

	// In most circumstances, use the SugaredLogger. It's 4-10x faster than most
	// other structured logging packages and has a familiar, loosely-typed API.
	sugar := logger.Sugar()
	sugar.Infow("Failed to fetch URL.",
		// Structured context as loosely typed key-value pairs.
		"url", url,
		"attempt", 3,
		"backoff", time.Second,
	)
	sugar.Infof("Failed to fetch URL: %s", url)

	// In the unusual situations where every microsecond matters, use the
	// Logger. It's even faster than the SugaredLogger, but only supports
	// structured logging.
	logger.Info("Failed to fetch URL.",
		// Structured context as strongly typed fields.
		lad.String("url", url),
		lad.Int("attempt", 3),
		lad.Duration("backoff", time.Second),
	)
	// Output:
	// {"level":"info","msg":"Failed to fetch URL.","url":"http://example.com","attempt":3,"backoff":"1s"}
	// {"level":"info","msg":"Failed to fetch URL: http://example.com"}
	// {"level":"info","msg":"Failed to fetch URL.","url":"http://example.com","attempt":3,"backoff":"1s"}
}

func Example_basicConfiguration() {
	// For some users, the presets offered by the NewProduction, NewDevelopment,
	// and NewExample constructors won't be appropriate. For most of those
	// users, the bundled Config struct offers the right balance of flexibility
	// and convenience. (For more complex needs, see the AdvancedConfiguration
	// example.)
	//
	// See the documentation for Config and ladcore.EncoderConfig for all the
	// available options.
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "json",
	  "outputPaths": ["stdout", "/tmp/logs"],
	  "errorOutputPaths": ["stderr"],
	  "initialFields": {"foo": "bar"},
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg lad.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger := lad.Must(cfg.Build())
	defer logger.Sync()

	logger.Info("logger construction succeeded")
	// Output:
	// {"level":"info","message":"logger construction succeeded","foo":"bar"}
}

func Example_advancedConfiguration() {
	// The bundled Config struct only supports the most common configuration
	// options. More complex needs, like splitting logs between multiple files
	// or writing to non-file outputs, require use of the ladcore package.
	//
	// In this example, imagine we're both sending our logs to Kafka and writing
	// them to the console. We'd like to encode the console output and the Kafka
	// topics differently, and we'd also like special treatment for
	// high-priority logs.

	// First, define our level-handling logic.
	highPriority := lad.LevelEnablerFunc(func(lvl ladcore.Level) bool {
		return lvl >= ladcore.ErrorLevel
	})
	lowPriority := lad.LevelEnablerFunc(func(lvl ladcore.Level) bool {
		return lvl < ladcore.ErrorLevel
	})

	// Assume that we have clients for two Kafka topics. The clients implement
	// ladcore.WriteSyncer and are safe for concurrent use. (If they only
	// implement io.Writer, we can use ladcore.AddSync to add a no-op Sync
	// method. If they're not safe for concurrent use, we can add a protecting
	// mutex with ladcore.Lock.)
	topicDebugging := ladcore.AddSync(io.Discard)
	topicErrors := ladcore.AddSync(io.Discard)

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	consoleDebugging := ladcore.Lock(os.Stdout)
	consoleErrors := ladcore.Lock(os.Stderr)

	// Optimize the Kafka output for machine consumption and the console output
	// for human operators.
	kafkaEncoder := ladcore.NewJSONEncoder(lad.NewProductionEncoderConfig())
	consoleEncoder := ladcore.NewConsoleEncoder(lad.NewDevelopmentEncoderConfig())

	// Join the outputs, encoders, and level-handling functions into
	// ladcore.Cores, then tee the four cores together.
	core := ladcore.NewTee(
		ladcore.NewCore(kafkaEncoder, topicErrors, highPriority),
		ladcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		ladcore.NewCore(kafkaEncoder, topicDebugging, lowPriority),
		ladcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)

	// From a ladcore.Core, it's easy to construct a Logger.
	logger := lad.New(core)
	defer logger.Sync()
	logger.Info("constructed a logger")
}

func ExampleNamespace() {
	logger := lad.NewExample()
	defer logger.Sync()

	logger.With(
		lad.Namespace("metrics"),
		lad.Int("counter", 1),
	).Info("tracked some metrics")
	// Output:
	// {"level":"info","msg":"tracked some metrics","metrics":{"counter":1}}
}

type addr struct {
	IP   string
	Port int
}

type request struct {
	URL    string
	Listen addr
	Remote addr
}

func (a addr) MarshalLogObject(enc ladcore.ObjectEncoder) error {
	enc.AddString("ip", a.IP)
	enc.AddInt("port", a.Port)
	return nil
}

func (r *request) MarshalLogObject(enc ladcore.ObjectEncoder) error {
	enc.AddString("url", r.URL)
	lad.Inline(r.Listen).AddTo(enc)
	return enc.AddObject("remote", r.Remote)
}

func ExampleObject() {
	logger := lad.NewExample()
	defer logger.Sync()

	req := &request{
		URL:    "/test",
		Listen: addr{"127.0.0.1", 8080},
		Remote: addr{"127.0.0.1", 31200},
	}
	logger.Info("new request, in nested object", lad.Object("req", req))
	logger.Info("new request, inline", lad.Inline(req))
	// Output:
	// {"level":"info","msg":"new request, in nested object","req":{"url":"/test","ip":"127.0.0.1","port":8080,"remote":{"ip":"127.0.0.1","port":31200}}}
	// {"level":"info","msg":"new request, inline","url":"/test","ip":"127.0.0.1","port":8080,"remote":{"ip":"127.0.0.1","port":31200}}
}

func ExampleNewStdLog() {
	logger := lad.NewExample()
	defer logger.Sync()

	std := lad.NewStdLog(logger)
	std.Print("standard logger wrapper")
	// Output:
	// {"level":"info","msg":"standard logger wrapper"}
}

func ExampleRedirectStdLog() {
	logger := lad.NewExample()
	defer logger.Sync()

	undo := lad.RedirectStdLog(logger)
	defer undo()

	log.Print("redirected standard library")
	// Output:
	// {"level":"info","msg":"redirected standard library"}
}

func ExampleReplaceGlobals() {
	logger := lad.NewExample()
	defer logger.Sync()

	undo := lad.ReplaceGlobals(logger)
	defer undo()

	lad.L().Info("replaced lad's global loggers")
	// Output:
	// {"level":"info","msg":"replaced lad's global loggers"}
}

func ExampleAtomicLevel() {
	atom := lad.NewAtomicLevel()

	// To keep the example deterministic, disable timestamps in the output.
	encoderCfg := lad.NewProductionEncoderConfig()
	encoderCfg.TimeKey = ""

	logger := lad.New(ladcore.NewCore(
		ladcore.NewJSONEncoder(encoderCfg),
		ladcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	logger.Info("info logging enabled")

	atom.SetLevel(lad.ErrorLevel)
	logger.Info("info logging disabled")
	// Output:
	// {"level":"info","msg":"info logging enabled"}
}

func ExampleAtomicLevel_config() {
	// The lad.Config struct includes an AtomicLevel. To use it, keep a
	// reference to the Config.
	rawJSON := []byte(`{
		"level": "info",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"encoding": "json",
		"encoderConfig": {
			"messageKey": "message",
			"levelKey": "level",
			"levelEncoder": "lowercase"
		}
	}`)
	var cfg lad.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	logger := lad.Must(cfg.Build())
	defer logger.Sync()

	logger.Info("info logging enabled")

	cfg.Level.SetLevel(lad.ErrorLevel)
	logger.Info("info logging disabled")
	// Output:
	// {"level":"info","message":"info logging enabled"}
}

func ExampleLogger_Check() {
	logger := lad.NewExample()
	defer logger.Sync()

	if ce := logger.Check(lad.DebugLevel, "debugging"); ce != nil {
		// If debug-level log output isn't enabled or if lad's sampling would have
		// dropped this log entry, we don't allocate the slice that holds these
		// fields.
		ce.Write(
			lad.String("foo", "bar"),
			lad.String("baz", "quux"),
		)
	}

	// Output:
	// {"level":"debug","msg":"debugging","foo":"bar","baz":"quux"}
}

func ExampleLogger_Named() {
	logger := lad.NewExample()
	defer logger.Sync()

	// By default, Loggers are unnamed.
	logger.Info("no name")

	// The first call to Named sets the Logger name.
	main := logger.Named("main")
	main.Info("main logger")

	// Additional calls to Named create a period-separated path.
	main.Named("subpackage").Info("sub-logger")
	// Output:
	// {"level":"info","msg":"no name"}
	// {"level":"info","logger":"main","msg":"main logger"}
	// {"level":"info","logger":"main.subpackage","msg":"sub-logger"}
}

func ExampleWrapCore_replace() {
	// Replacing a Logger's core can alter fundamental behaviors.
	// For example, it can convert a Logger to a no-op.
	nop := lad.WrapCore(func(ladcore.Core) ladcore.Core {
		return ladcore.NewNopCore()
	})

	logger := lad.NewExample()
	defer logger.Sync()

	logger.Info("working")
	logger.WithOptions(nop).Info("no-op")
	logger.Info("original logger still works")
	// Output:
	// {"level":"info","msg":"working"}
	// {"level":"info","msg":"original logger still works"}
}

func ExampleWrapCore_wrap() {
	// Wrapping a Logger's core can extend its functionality. As a trivial
	// example, it can double-write all logs.
	doubled := lad.WrapCore(func(c ladcore.Core) ladcore.Core {
		return ladcore.NewTee(c, c)
	})

	logger := lad.NewExample()
	defer logger.Sync()

	logger.Info("single")
	logger.WithOptions(doubled).Info("doubled")
	// Output:
	// {"level":"info","msg":"single"}
	// {"level":"info","msg":"doubled"}
	// {"level":"info","msg":"doubled"}
}

func ExampleDict() {
	logger := lad.NewExample()
	defer logger.Sync()

	logger.Info("login event",
		lad.Dict("event",
			lad.Int("id", 123),
			lad.String("name", "jane"),
			lad.String("status", "pending")))
	// Output:
	// {"level":"info","msg":"login event","event":{"id":123,"name":"jane","status":"pending"}}
}

func ExampleObjects() {
	logger := lad.NewExample()
	defer logger.Sync()

	// Use the Objects field constructor when you have a list of objects,
	// all of which implement ladcore.ObjectMarshaler.
	logger.Debug("opening connections",
		lad.Objects("addrs", []addr{
			{IP: "123.45.67.89", Port: 4040},
			{IP: "127.0.0.1", Port: 4041},
			{IP: "192.168.0.1", Port: 4042},
		}))
	// Output:
	// {"level":"debug","msg":"opening connections","addrs":[{"ip":"123.45.67.89","port":4040},{"ip":"127.0.0.1","port":4041},{"ip":"192.168.0.1","port":4042}]}
}

func ExampleDictObject() {
	logger := lad.NewExample()
	defer logger.Sync()

	// Use DictObject to create ladcore.ObjectMarshaler implementations from Field arrays,
	// then use the Object and Objects field constructors to turn them back into a Field.

	logger.Debug("worker received job",
		lad.Object("w1",
			lad.DictObject(
				lad.Int("id", 402000),
				lad.String("description", "compress image data"),
				lad.Int("priority", 3),
			),
		))

	d1 := 68 * time.Millisecond
	d2 := 79 * time.Millisecond
	d3 := 57 * time.Millisecond

	logger.Info("worker status checks",
		lad.Objects("job batch enqueued",
			[]ladcore.ObjectMarshaler{
				lad.DictObject(
					lad.String("worker", "w1"),
					lad.Int("load", 419),
					lad.Duration("latency", d1),
				),
				lad.DictObject(
					lad.String("worker", "w2"),
					lad.Int("load", 520),
					lad.Duration("latency", d2),
				),
				lad.DictObject(
					lad.String("worker", "w3"),
					lad.Int("load", 310),
					lad.Duration("latency", d3),
				),
			},
		))
	// Output:
	// {"level":"debug","msg":"worker received job","w1":{"id":402000,"description":"compress image data","priority":3}}
	// {"level":"info","msg":"worker status checks","job batch enqueued":[{"worker":"w1","load":419,"latency":"68ms"},{"worker":"w2","load":520,"latency":"79ms"},{"worker":"w3","load":310,"latency":"57ms"}]}
}

func ExampleObjectValues() {
	logger := lad.NewExample()
	defer logger.Sync()

	// Use the ObjectValues field constructor when you have a list of
	// objects that do not implement ladcore.ObjectMarshaler directly,
	// but on their pointer receivers.
	logger.Debug("starting tunnels",
		lad.ObjectValues("addrs", []request{
			{
				URL:    "/foo",
				Listen: addr{"127.0.0.1", 8080},
				Remote: addr{"123.45.67.89", 4040},
			},
			{
				URL:    "/bar",
				Listen: addr{"127.0.0.1", 8080},
				Remote: addr{"127.0.0.1", 31200},
			},
		}))
	// Output:
	// {"level":"debug","msg":"starting tunnels","addrs":[{"url":"/foo","ip":"127.0.0.1","port":8080,"remote":{"ip":"123.45.67.89","port":4040}},{"url":"/bar","ip":"127.0.0.1","port":8080,"remote":{"ip":"127.0.0.1","port":31200}}]}
}
