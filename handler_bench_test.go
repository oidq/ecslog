package ecslog

import (
	"log/slog"
	"testing"
)

type nilWriter struct{}

func (n *nilWriter) Write(src []byte) (int, error) {
	return len(src), nil
}

func BenchmarkHandler_Handle(b *testing.B) {
	sl := slog.New(slog.NewJSONHandler(&nilWriter{}, nil))
	b.Run("basic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sl.Info("test",
				slog.Group(
					"log",
					slog.String("file", "test"),
				),
			)
		}
	})

	b.Run("Complex", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sl.Info("Hello World",
				slog.Group("log",
					slog.String("source", "Test()"),
					slog.Group("syslog",
						slog.String("hostname", "ecslog"),
						slog.Int("priority", 1),
					),
				),
				slog.Group("event",
					slog.String("action", "test"),
				),
				slog.Group("file",
					slog.String("device", "sda"),
				),
			)
		}
	})
}

func BenchmarkHandler_HandleEcslog(b *testing.B) {
	ecs := slog.New(NewHandler(&nilWriter{}))

	b.Run("Basic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ecs.Info("test",
				slog.String("log.file", "test"),
			)
		}
	})

	b.Run("Complex", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ecs.Info("Hello World",
				slog.String("log.syslog.hostname", "ecslog"),
				slog.String("event.action", "test"),
				slog.String("log.source", "Test()"),
				slog.String("file.device", "sda"),
				slog.Int("log.syslog.priority", 1),
			)
		}
	})

	b.Run("ComplexPreContext", func(b *testing.B) {
		b.ReportAllocs()
		nLog := ecs.With(
			slog.String("log.syslog.hostname", "ecslog"),
			slog.String("file.device", "sda"),
		)
		for i := 0; i < b.N; i++ {
			nLog.Info("Hello World",
				slog.String("event.action", "test"),
				slog.String("log.source", "Test()"),
				slog.Int("log.syslog.priority", 1),
			)
		}
	})
}
