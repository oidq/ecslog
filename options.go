package ecslog

import (
	"context"
	"log/slog"
)

// LogLevelFunc is used in WithLogLevelFunc to control which log entry is going to
// be logged. For basic control based on slog.Level use WithLogLevel().
type LogLevelFunc func(ctx context.Context, level slog.Level) bool

type handlerOptions struct {
	hideTimestamp bool
	addSource     bool

	levelF LogLevelFunc
}

type Option func(*handlerOptions)

func getOptions(options []Option) *handlerOptions {
	hOptions := &handlerOptions{
		levelF: func(_ context.Context, level slog.Level) bool {
			return level >= slog.LevelInfo
		},
	}

	for _, option := range options {
		option(hOptions)
	}
	return hOptions
}

// WithTimestamp option can be used to disable timestamp
// ("@timestamp" field) in generated logs.
func WithTimestamp(showTimestamp bool) Option {
	return func(h *handlerOptions) {
		h.hideTimestamp = !showTimestamp
	}
}

// WithSource option can be used to add log source information to logs.
//
// Keys "log.origin.function", "log.origin.file" and "log.origin.line" will
// filled by information received from [log/slog].
func WithSource(addSource bool) Option {
	return func(h *handlerOptions) {
		h.addSource = addSource
	}
}

// WithLogLevel options sets minimum log level to be outputted.
// Default value is slog.LevelInfo.
//
// This option is exclusive with WithLogLevelFunc.
func WithLogLevel(minLevel slog.Level) Option {
	return func(h *handlerOptions) {
		h.levelF = func(ctx context.Context, level slog.Level) bool {
			return level >= minLevel
		}
	}
}

// WithLogLevelFunc options sets log level decision function.
// Given function will be called for each log entry and if it returns false,
// no log will be produced. Context passed to this function comes from slog.
//
// This option is exclusive with WithLogLevel.
func WithLogLevelFunc(logLevelF LogLevelFunc) Option {
	return func(h *handlerOptions) {
		h.levelF = logLevelF
	}
}
