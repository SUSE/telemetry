# The Telemetry Blob

The goal of the SUSE Telemetry solution is the transportation of a telemetry blob, expressed as a JSON encoded byte sequence.

The telemetry blob byte sequence must satisfy the following requirements:
1. it must be valid JSON.
2. it must be a JSON object, i.e. start with `{` and end with `}`.
3. it must contain a `version` field.
4. it must not exceed certain [limits](../pkg/limits).

## Examples

For examples of theoretical JSON telemetry blob data see [SLE-SERVER-SCCHwInfo](../testdata/telemetry/SLE-SERVER-SCCHwInfo/) and [SLE-SERVER-Test](../examples/telemetry/SLE-SERVER-Test.json)
