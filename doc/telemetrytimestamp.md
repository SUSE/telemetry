# Telemetry Timestamp

When timestamps are generated within the telemetry service they are expected to be full data and time UTC timestamps in an [RFC 3339 compatible format](https://www.rfc-editor.org/rfc/rfc3339) with nanosecond resolution. See the [rfc3339nano](https://pkg.go.dev/time#pkg-constants) type for specific details.