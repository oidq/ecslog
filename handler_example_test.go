package ecslog_test

import (
	"log/slog"
	"os"

	"github.com/oidq/ecslog"
)

func Example() {
	logger := slog.New(ecslog.NewHandler(os.Stdout, ecslog.WithTimestamp(false))).
		With(
			slog.String("event.dataset", "testing"),
		)
	// ...
	logger.Info("Test!",
		slog.String("event.action", "test"),
	)
	// Output: {"message":"Test!","log":{"level":"INFO"},"event":{"dataset":"testing","action":"test"}}
}
