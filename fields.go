package ecslog

import (
	"log/slog"
)

const groupSeparator = '.'

func isIgnoredKey(key string) bool {
	return key == "@timestamp" || key == "message"
}

func isEarlierAttr(a, b slog.Attr) int {
	return isEarlierKey(a.Key, b.Key)
}

func isEarlierKey(a, b string) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] == b[i] {
			continue
		}

		if a[i] == groupSeparator {
			return 1
		}

		if b[i] == groupSeparator {
			return -1
		}

		if a[i] < b[i] {
			return 1
		}

		return -1
	}

	return len(b) - len(a)
}
