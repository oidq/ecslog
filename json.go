package ecslog

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/oidq/ecslog/internal/sjson"
)

// preformattedKinds is lookup of which kinds should be pre-formatted when creating new Handler with attributes.
//
// It is fairly cheap to format for example Kind.Uint64 to given buffer for every log entry,
// but it makes sense to pre-format these kinds to newly allocated buffers.
var preformattedKinds = [16]bool{
	slog.KindAny:       true,
	slog.KindGroup:     true,
	slog.KindLogValuer: true,
}

type preformattedValue struct {
	// value contains preformatted value with necessary quotations
	value []byte
}

func resolveRecord(output []byte, t0 time.Time, msg string, sortedAttrs []slog.Attr) []byte {
	output = append(output, '{')

	// @timestamp and message has special treatment to prevent unnecessary operations regarding
	// slog.Time and slog.String
	hasValue := false
	if !t0.IsZero() {
		output = appendKey(output, false, "@timestamp")
		output = appendTime(output, t0)
		hasValue = true
	}
	if msg != "" {
		output = appendKey(output, hasValue, "message")
		output = appendJsonString(output, msg)
		hasValue = true
	}

	output = resolveGroupContent(output, hasValue, 0, sortedAttrs)

	output = append(output, '}')

	return output
}

func resolveGroup(output []byte, prefixLen int, attributes []slog.Attr) []byte {
	output = append(output, '{')
	output = resolveGroupContent(output, false, prefixLen, attributes)
	output = append(output, '}')

	return output
}

func resolveGroupPtr(output []byte, prefixLen int, attributes []slog.Attr) []byte {
	output = append(output, '{')
	output = resolveGroupContent(output, false, prefixLen, attributes)
	output = append(output, '}')

	return output
}

func resolveGroupContent(output []byte, hasValue bool, prefixLen int, attributes []slog.Attr) []byte {

	// slice of fields in the current group,
	// it is created by slicing the attributes parameter
	var currentGroupFields []slog.Attr
	currentGroup := ""

	var key string
	for i := range attributes {
		attr := attributes[i]
		key = attr.Key[prefixLen:]
		groupSeparatorIndex := strings.IndexRune(key, groupSeparator)

		if groupSeparatorIndex != -1 {
			// continuing of an established group
			if currentGroup == "" || currentGroup == key[:groupSeparatorIndex] {
				currentGroup = key[:groupSeparatorIndex]
				currentGroupFields = attributes[i-len(currentGroupFields) : i+1]
				continue
			}

			// we encountered a new group -> flush the current one
			output = appendKey(output, hasValue, currentGroup)
			output = resolveGroupPtr(output, prefixLen+len(currentGroup)+1, currentGroupFields)
			hasValue = true

			// create new group
			currentGroup = key[:groupSeparatorIndex]
			currentGroupFields = attributes[i : i+1]
			continue
		}

		// top level attribute with the same name as the group (we necessarily have a group open)
		if currentGroup == key {
			// "Group" exists, but top-level atomic attribute is present
			// which overrides the value and we use it.
			currentGroup = ""
			currentGroupFields = currentGroupFields[:0]

			output = appendJsonKV(output, hasValue, key, attr.Value)
			hasValue = true
			continue
		}

		// top level attribute with the same name as the previous one -> skip
		if i < len(attributes)-1 && attributes[i+1].Key == attributes[i].Key {
			continue
		}

		attr.Key = key

		output = appendJsonKV(output, hasValue, key, attr.Value)
		hasValue = true
	}

	// deal with any unfinished group
	if currentGroup != "" {
		output = appendKey(output, hasValue, currentGroup)
		output = resolveGroupPtr(output, prefixLen+len(currentGroup)+1, currentGroupFields)
		hasValue = true
	}
	return output
}

func appendKey(output []byte, hasValue bool, key string) []byte {
	if hasValue {
		output = append(output, ',')
	}

	output = appendJsonString(output, key)
	output = append(output, ':')
	return output
}

func shouldPreformat(kind slog.Kind) bool {
	if int(kind) > len(preformattedKinds) {
		return false
	}
	return preformattedKinds[kind]
}

func preformatValue(value slog.Value) slog.Value {
	var formattedValue []byte
	formatted := appendJsonValue(formattedValue, value)

	return slog.AnyValue(preformattedValue{value: formatted})

}

func appendJsonKV(output []byte, hasValue bool, key string, value slog.Value) []byte {
	output = appendKey(output, hasValue, key)
	output = appendJsonValue(output, value)
	return output
}

func appendJsonValue(output []byte, value slog.Value) []byte {

	if value.Kind() == slog.KindLogValuer {
		value = value.Resolve()
	}

	switch value.Kind() {
	case slog.KindGroup:
		output = resolveGroup(output, 0, value.Group())
	case slog.KindBool:
		if value.Bool() {
			output = append(output, "true"...)
		} else {
			output = append(output, "false"...)
		}
	case slog.KindTime:
		output = appendTime(output, value.Time())
	case slog.KindUint64:
		output = strconv.AppendUint(output, value.Uint64(), 10)
	case slog.KindInt64:
		output = strconv.AppendInt(output, value.Int64(), 10)
	case slog.KindString:
		output = appendJsonString(output, value.String())
	case slog.KindDuration:
		output = strconv.AppendInt(output, value.Duration().Nanoseconds(), 10)
	case slog.KindFloat64:
		// adhere to json.Marshal with floats
		output = appendMarshal(output, value.Float64())
	case slog.KindAny:
		val := value.Any()
		if pref, ok := val.(preformattedValue); ok {
			return append(output, pref.value...)
		}
		output = appendMarshal(output, val)
	default:
		output = appendJsonString(output, fmt.Sprintf("ERR! invalid value: %#v", value.Any()))

	}
	return output
}

func appendTime(output []byte, t0 time.Time) []byte {
	output = append(output, '"')
	output = t0.AppendFormat(output, time.RFC3339Nano)
	output = append(output, '"')
	return output
}

func appendJsonString(output []byte, value string) []byte {
	output = append(output, '"')
	output = sjson.AppendStringContent(output, value)
	output = append(output, '"')

	return output
}

type simpleWriter struct {
	buffer []byte
}

func (w *simpleWriter) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)
	return len(p), nil
}

func appendMarshal(output []byte, v any) []byte {
	writer := simpleWriter{buffer: output}
	enc := json.NewEncoder(&writer)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	if err != nil {
		output = appendJsonString(output, "ERR!"+err.Error())
	}

	// remove newline
	return writer.buffer[:len(writer.buffer)-1]
}
