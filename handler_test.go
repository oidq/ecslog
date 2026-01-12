package ecslog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type val = map[string]any
type arr = []interface{}

func injectTimestamp(val val) func() any {
	return func() any {
		val["@timestamp"] = time.Now().Truncate(1 * time.Nanosecond).Format(time.RFC3339Nano)
		return val
	}
}

type logValuer struct {
	val slog.Value
}

func (l logValuer) LogValue() slog.Value {
	return l.val
}

var handeTestObj = []struct {
	name           string
	f              func(l *slog.Logger)
	expectedOutput func() any
}{
	{
		name: "Basic",
		f: func(l *slog.Logger) {
			l.With().Info("Hello World")
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO"},
		}),
	},
	{
		name: "LogSource",
		f: func(l *slog.Logger) {
			l.With().Info("Hello World", slog.String("log.source", "Test()"))
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO", "source": "Test()"},
		}),
	},
	{
		name: "FullExampleWith",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.String("log.logger", "ecslog"),
			)
			nLog.Info("Hello World",
				slog.String("log.syslog.hostname", "ecslog"),
				slog.String("event.action", "test"),
				slog.String("log.source", "Test()"),
				slog.String("file.device", "sda"),
				slog.Int("log.syslog.priority", 1),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"file": val{
				"device": "sda",
			},
			"event": val{
				"action": "test",
			},
			"log": val{
				"logger": "ecslog",
				"level":  "INFO",
				"source": "Test()",
				"syslog": val{
					"hostname": "ecslog",
					"priority": float64(1), // json.Unmarshal decodes nums as float64
				},
			},
		}),
	},
	{
		name: "FullExampleWithGroup",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.String("log.logger", "ecslog"),
				slog.String("file.device", "sda"),
			).WithGroup("log")

			nLog.Info("Hello World",
				slog.String("syslog.hostname", "ecslog"),
				slog.String("source", "Test()"),
				slog.Int("syslog.priority", 1),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"file": val{
				"device": "sda",
			},
			"log": val{
				"logger": "ecslog",
				"level":  "INFO",
				"source": "Test()",
				"syslog": val{
					"hostname": "ecslog",
					"priority": float64(1), // json.Unmarshal decodes nums as float64
				},
			},
		}),
	},
	{
		name: "Overwrite",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.String("log.syslog.hostname", "slog"),
			)

			nLog.Info("Hello World",
				slog.String("log.syslog.hostname", "ecslog"),
				slog.Int("log.syslog.priority", 1),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log": val{
				"level": "INFO",
				"syslog": val{
					"hostname": "ecslog",
					"priority": float64(1), // json.Unmarshal decodes nums as float64
				},
			},
		}),
	},
	{
		name: "OverwriteGroup",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.String("log.syslog.hostname", "slog"),
			).WithGroup("log")

			nLog.Info("Hello World",
				slog.String("syslog.hostname", "ecslog"),
				slog.Int("syslog.priority", 1),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log": val{
				"level": "INFO",
				"syslog": val{
					"hostname": "ecslog",
					"priority": float64(1), // json.Unmarshal decodes nums as float64
				},
			},
		}),
	},
	{
		name: "MergingGroup",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.Group("event",
					slog.String("dataset", "tests"),
				),
			)

			nLog.Info("Hello World",
				slog.String("event.action", "test()"),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO"},
			"event":   val{"dataset": "tests"},
		}),
	},
	{
		name: "Overlap",
		f: func(l *slog.Logger) {
			l.Info("Hello World",
				slog.String("event.action", "test()"),
				slog.String("events.action", "tests"),
				slog.String("event.dataset", "test"),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO"},
			"event": val{
				"dataset": "test",
				"action":  "test()",
			},
			"events": val{
				"action": "tests",
			},
		}),
	},
	{
		name: "SpecialTypes",
		f: func(l *slog.Logger) {
			l.Info("Hello World",
				slog.String("message", "test"),
				slog.Bool("event.bool", true),
				slog.Bool("event.booled", false),
				slog.Uint64("event.int", 123),
				slog.Duration("event.duration", time.Millisecond),
				slog.Float64("event.float", 3.2),
				slog.Time("event.time", time.Now().In(time.UTC)),
				slog.Any("event.any", []string{"ha!", "ha"}),
				slog.Any("event.value", logValuer{val: slog.StringValue("value")}),
				slog.Any("event.obj", val{"baz": "bar"}),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO"},
			"event": val{
				"bool":     true,
				"booled":   false,
				"time":     "2000-01-01T00:00:00Z",
				"int":      float64(123),
				"float":    3.2,
				"duration": float64(time.Millisecond),
				"any":      arr{"ha!", "ha"},
				"value":    "value",
				"obj":      val{"baz": "bar"},
			},
		}),
	},
	{
		name: "SpecialTypesInAttrs",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.String("message", "test"),
				slog.Bool("event.bool", true),
				slog.Bool("event.booled", false),
				slog.Uint64("event.int", 123),
				slog.Duration("event.duration", time.Millisecond),
				slog.Float64("event.float", 3.2),
				slog.Time("event.time", time.Now().In(time.UTC)),
				slog.Any("event.any", []string{"ha!", "ha"}),
				slog.Any("event.value", logValuer{val: slog.StringValue("value")}),
				slog.Any("event.obj", val{"baz": "bar"}),
			)
			nLog.Info("Hello World")
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO"},
			"event": val{
				"bool":     true,
				"booled":   false,
				"time":     "2000-01-01T00:00:00Z",
				"int":      float64(123),
				"float":    3.2,
				"duration": float64(time.Millisecond),
				"any":      arr{"ha!", "ha"},
				"value":    "value",
				"obj":      val{"baz": "bar"},
			},
		}),
	},
	{
		name: "EmptyAttr",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.Attr{},
			)
			nLog.Info("",
				slog.Attr{},
			)
		},
		expectedOutput: injectTimestamp(val{
			"log": val{"level": "INFO"},
		}),
	},
	{
		name: "EmptyGroup",
		f: func(l *slog.Logger) {
			nLog := l.With(
				slog.Group("src"),
			)
			nLog.Info("",
				slog.Group("dst"),
			)
		},
		expectedOutput: injectTimestamp(val{
			"log": val{"level": "INFO"},
		}),
	},
	{
		name: "IgnoredMsg",
		f: func(l *slog.Logger) {
			nL := l.With(
				slog.String("message", "test"),
			)
			nL.Info("Hello World",
				slog.Bool("event.bool", true),
			)
		},
		expectedOutput: injectTimestamp(val{
			"message": "Hello World",
			"log":     val{"level": "INFO"},
			"event":   val{"bool": true},
		}),
	},
}

func TestHandler_Handle_ObjMatch(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	ecs := slog.New(NewHandler(buff))

	for _, data := range handeTestObj {
		t.Run(data.name, func(t *testing.T) {
			buff.Reset()
			var expectedOutput any

			synctest.Test(t, func(t *testing.T) {
				expectedOutput = data.expectedOutput()
				data.f(ecs)
			})

			var output map[string]interface{}
			err := json.Unmarshal(buff.Bytes(), &output)
			if err != nil {
				t.Fatalf("log produced invalid json: %s\nGOT: %s", err, string(buff.Bytes()))
			}

			if !reflect.DeepEqual(output, expectedOutput) {
				b, _ := json.Marshal(expectedOutput)
				t.Errorf("mismatched log data\nEXP: %#v\nGOT: %#v", expectedOutput, output)
				t.Errorf("mismatched log data\nEXP: %s\nGOT: %s", string(b), buff.String())
			}
		})
	}
}

func TestHandler_Handle_Duplicates(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	ecs := slog.New(NewHandler(buff))

	nLog := ecs.With(
		slog.String("log.syslog.hostname", "slog"),
	)

	nLog.Info("Hello World",
		slog.String("log.syslog.hostname", "ecslog"),
		slog.Int("log.syslog.priority", 1),
	)

	count := strings.Count(buff.String(), "hostname")
	if count != 1 {
		t.Errorf("log produced json with duplicate keys: %s", buff.String())
	}
}

func TestHandler_Handle_Source(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	ecs := slog.New(NewHandler(buff, WithSource(true)))

	var file string
	var line int
	synctest.Test(t, func(t *testing.T) {
		_, file, line, _ = runtime.Caller(0)
		ecs.Warn("Hello World",
			slog.String("event.action", "test"),
		)
	})

	expectedOutput := val{
		"message":    "Hello World",
		"@timestamp": "2000-01-01T01:00:00+01:00",
		"event":      val{"action": "test"},
		"log": val{
			"level": "WARN",
			"origin": val{
				"function": "github.com/oidq/ecslog.TestHandler_Handle_Source.func1",
				"file":     file,
				"line":     float64(line + 1),
			},
		},
	}

	var output map[string]interface{}
	err := json.Unmarshal(buff.Bytes(), &output)
	if err != nil {
		t.Fatalf("log produced invalid json: %s\nGOT: %s", err, string(buff.Bytes()))
	}

	if !reflect.DeepEqual(output, expectedOutput) {
		b, _ := json.Marshal(expectedOutput)
		t.Errorf("mismatched log data\nEXP: %#v\nGOT: %#v", expectedOutput, output)
		t.Errorf("mismatched log data\nEXP: %s\nGOT: %s", string(b), buff.String())
	}
}

func TestHandler_Handle_WithLogLevel(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	ecs := slog.New(NewHandler(buff, WithLogLevel(slog.LevelError)))

	ecs.Warn("Hello World",
		slog.String("event.action", "test"),
	)
	if buff.Len() > 0 {
		t.Error("log produced output on lower level")
	}

	ecs.Error("Hello World",
		slog.String("event.action", "test"),
	)
	if buff.Len() < 1 {
		t.Error("log did not produce output on upper level")
	}
}

func TestHandler_Handle_WithAttrs(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	ecs := slog.New(NewHandler(buff, WithTimestamp(false), WithLogLevel(slog.LevelWarn)))

	ecsLog1 := ecs.With(slog.String("foo", "test"))

	ecsLog2 := ecs.With(slog.String("bar", "test"))

	ecsLog1.Warn("")
	ecsLog2.Warn("")

	var expectedOutput = []val{
		{"log": val{"level": "WARN"}, "foo": "test"},
		{"log": val{"level": "WARN"}, "bar": "test"},
	}

	output := unmarshalLogs(t, buff)
	if !reflect.DeepEqual(output, expectedOutput) {
		t.Errorf("mismatched log data\nEXP: %#v\nGOT: %#v", expectedOutput, output)
	}
}

func TestHandler_Handle_Any(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	ecs := slog.New(NewHandler(buff, WithTimestamp(false), WithLogLevel(slog.LevelWarn)))

	ecs.Warn("", slog.Any("foo", []string{"test1", "test2"}))

	var expectedOutput = []val{
		{"log": val{"level": "WARN"}, "foo": arr{"test1", "test2"}},
	}

	output := unmarshalLogs(t, buff)
	if !reflect.DeepEqual(output, expectedOutput) {
		t.Errorf("mismatched log data\nEXP: %#v\nGOT: %#v", expectedOutput, output)
	}
}

func unmarshalLogs(t *testing.T, input io.Reader) []map[string]interface{} {
	var lines []map[string]interface{}
	scanner := bufio.NewScanner(input)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		var output map[string]interface{}
		err := json.Unmarshal(scanner.Bytes(), &output)
		if err != nil {
			t.Fatalf("log produced invalid json: %s\nGOT: %s", err, string(scanner.Bytes()))
		}
		lines = append(lines, output)
	}

	return lines
}
