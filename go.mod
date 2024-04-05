module github.com/SUSE/telemetry

go 1.21.0

replace github.com/SUSE/telemetry => ../telemetry

require (
	github.com/SUSE/telemetrylib v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
)

replace github.com/SUSE/telemetrylib => ../telemetrylib
