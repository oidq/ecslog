# ECSLog: Slog Handler for ECS logging

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/oidq/ecslog) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/oidq/ecslog/master/LICENSE)

This `slog` handler makes structured ECS logging easier with `slog`
by accepting the dot notation (`event.action`) and producing corresponding
nested JSON objects. This approach eliminates the need for nested grouping when
constructing the log attributes.

**For details see
[documentation](https://pkg.go.dev/github.com/oidq/ecslog#Handler).**

### Features:
- deduplication of scalar attributes
- downstream log instances can set attribute in "group" created upstream
- it has comparable performance to `slog.NewJSONHandler()`

### Performance

The implementation has very similar performance to `slog.NewJSONHandler()`,
based on some local testing with
[zap benchmark](https://github.com/uber-go/zap/tree/master/benchmarks).


## Example

```go
logger := slog.New(ecslog.NewHandler(os.Stderr)).
    With(
        slog.String("event.dataset", "testing"),
        slog.String("log.logger", "slog"),
    )
// ...
logger.Info("Test!",
    slog.String("event.action", "test"),
)
// {
//   "@timestamp": "2026-01-11T17:00:42.42648+01:00",
//   "message": "Test!",
//   "log": {
//     "logger": "slog",
//     "level": "INFO"
//   },
//   "event": {
//     "dataset": "testing",
//     "action": "test"
//   }
// }
```
