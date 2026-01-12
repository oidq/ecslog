package ecslog

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
)

// Handler is slog.Handler which produces JSON structured log output to given [io.Writer].
//
// See package documentation for details about record processing.
type Handler struct {
	writer  io.Writer
	options *handlerOptions

	handleContextPool *sync.Pool

	attrPrefix string
	attributes [][]slog.Attr
}

// NewHandler creates a new [slog.Handler] instance with given options.
func NewHandler(writer io.Writer, options ...Option) *Handler {
	return &Handler{
		writer:  writer,
		options: getOptions(options),
		handleContextPool: &sync.Pool{
			New: newHandleContext,
		},
	}
}

// Enabled controls log output. It is called by slog package.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.options.levelF(ctx, level)
}

// WithAttrs creates new [slog.Handler] with given default attributes.
// It is called by slog package.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	//var resolvedFields []slog.Attr

	for i := 0; i < len(attrs); i++ {
		// per slog.Handler doc we should ignore empty attributes
		if attrs[i].Equal(slog.Attr{}) {
			attrs = slices.Delete(attrs, i, i+1)
			i--
			continue
		}
		// per slog.Handler doc we should ignore empty groups
		if attrs[i].Value.Kind() == slog.KindGroup &&
			len(attrs[i].Value.Group()) == 0 {

			attrs = slices.Delete(attrs, i, i+1)
			i--
			continue
		}
		// ignore attributes conflicting with builtins
		if h.attrPrefix == "" && isIgnoredKey(attrs[i].Key) {
			attrs = slices.Delete(attrs, i, i+1)
			i--
			continue
		}
		attrs[i].Key = h.attrPrefix + attrs[i].Key
		if shouldPreformat(attrs[i].Value.Kind()) {
			attrs[i].Value = preformatValue(attrs[i].Value)
		}
	}

	return &Handler{
		writer:            h.writer,
		options:           h.options,
		handleContextPool: h.handleContextPool,
		attrPrefix:        h.attrPrefix,
		attributes:        append(h.attributes, attrs),
	}
}

// WithGroup creates new [slog.Handler] with given default group (see [slog.Handler] interface)
// It is called by slog package.
func (h *Handler) WithGroup(name string) slog.Handler {

	newGroupPrefix := h.attrPrefix + name + "."

	return &Handler{
		writer:            h.writer,
		options:           h.options,
		handleContextPool: h.handleContextPool,
		attrPrefix:        newGroupPrefix,
		attributes:        h.attributes,
	}
}
