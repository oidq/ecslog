package ecslog

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

// handle context used for sync.Pool allocations
type handleContext struct {
	outputBuffer     []byte
	attributesBuffer []slog.Attr
}

const handleCtxMaxAttrsSize = 128
const handleCtxMaxBufferSize = 4096

func newHandleContext() any {
	return &handleContext{
		outputBuffer:     make([]byte, 64),
		attributesBuffer: make([]slog.Attr, 16),
	}
}

func (h *Handler) getHandleCtx() *handleContext {
	return h.handleContextPool.Get().(*handleContext)
}

func (h *Handler) putHandleCtx(handleCtx *handleContext, outputBuff []byte, attrsBuff []slog.Attr) {
	if cap(handleCtx.outputBuffer) > handleCtxMaxBufferSize ||
		cap(handleCtx.attributesBuffer) > handleCtxMaxAttrsSize {

		// do not return enormously large buffers to the pool
		return
	}

	handleCtx.outputBuffer = outputBuff
	handleCtx.attributesBuffer = attrsBuff
	h.handleContextPool.Put(handleCtx)
}

// Handle processes [slog.Record] and writes structured JSON line to given [io.Writer].
// For details about the behavior regarding attributes see [Handler].
//
// It is called by slog package.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	// prepare buffers for the handling
	handleCtx := h.getHandleCtx()
	output := handleCtx.outputBuffer[:0]
	attrs := handleCtx.attributesBuffer[:0]

	// prepopulate the attributes
	attrs = append(attrs, slog.String("log.level", record.Level.String()))
	if h.options.addSource {
		attrs = addSource(attrs, record)
	}

	// insert attributes from Handler
	for _, attr := range h.attributes {
		attrs = append(attrs, attr...)
	}

	// insert attributes from record with respect to h.attrPrefix
	record.Attrs(func(attr slog.Attr) bool {
		// per slog.Handler doc we should ignore empty attributes
		if attr.Equal(slog.Attr{}) {
			return true
		}

		// per slog.Handler doc we should ignore empty groups
		if attr.Value.Kind() == slog.KindGroup && len(attr.Value.Group()) == 0 {
			return true
		}

		if h.attrPrefix != "" {
			attr.Key = h.attrPrefix + attr.Key
		} else if isIgnoredKey(attr.Key) {
			return true // top level ignored key
		}
		attrs = append(attrs, attr)
		return true
	})

	slices.SortStableFunc(attrs, isEarlierAttr)

	var t0 time.Time
	if !h.options.hideTimestamp {
		t0 = record.Time
	}

	// resolve the record to single log line
	output = resolveRecord(output, t0, record.Message, attrs)
	output = append(output, '\n')

	// from io.Write - "Write must not retain p",
	// meaning we can reuse the buffer later
	_, err := h.writer.Write(output)

	// return buffers for further use
	h.putHandleCtx(handleCtx, output, attrs)

	return err
}

func addSource(attrs []slog.Attr, record slog.Record) []slog.Attr {
	src := record.Source()
	if src == nil {
		return attrs
	}

	return append(
		attrs,
		slog.String("log.origin.function", src.Function),
		slog.String("log.origin.file", src.File),
		slog.Int("log.origin.line", src.Line),
	)
}
