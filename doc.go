/*
Package ecslog provides [slog] compatible handler which produces structured JSON.

Conceptually it is very similar with [slog.JSONHandler], but provides one key functionality.
It accepts dot delimited attribute keys and outputs records with appropriate nested JSON objects.

	slog.String("event.action": "test") -> {"event":{"action": "test"}}

This approach can be convenient for Elastic Common Schema ([ECS]),
which includes fair amount of nested JSON objects, and fields are usually described using dot notation
("event.action"). Compared to constructing log records with [slog.Group], it may be more convenient,
but crucially, it allows specifying nested object content in multiple assignments.

	datasetLog := log.WithAttrs(slog.String("event.dataset", "audit"))
	datasetLog.Info(slog.String("event.action", "test"))

Another advantage of this implementation is deduplication. Specifying single field multiple times
will produce deduplicated JSON record, which contains "last" value.

	datasetLog := log.WithAttrs(slog.String("event.action", "audit"))
	datasetLog.Info(slog.String("event.action", "test"))
	// {"event": {"action": "test"}, ...}

# Limitations

The approach with dot notation, combined with aim to be as fast as [slog.JSONHandler] leads to some
limitations. They are mainly related to the use of [slog.Group]. While the handler is compatible with
this attribute kind and will handle it as expected most of the time, sometimes it may differ.

  - Specifying both `slog.Group("event", ...)` and `slog.String("event.log", ...)` will always
    ignore the attributes nested with the dot notation, outputing only the group.
  - Multiple [slog.Group] with same keys are not merged, the last[1] one is used.
  - Attribute [slog.Group] with empty key is not inlined. This rule is outlines by [slog.Handler],
    but due to how groups are implemented, it is not possible without some sacrifices on the way.

[ECS]: https://github.com/elastic/ecs
*/
package ecslog
