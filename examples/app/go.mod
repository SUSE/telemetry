module github.com/SUSE/telemetry/examples/app

go 1.21.9

replace github.com/SUSE/telemetry => ../../

require github.com/SUSE/telemetry v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/xyproto/randomstring v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
