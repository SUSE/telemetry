# Telemetry Client Id Header
When a telemetry client is submitting a telemetry report using
the [/report](../requests/report.md) request it will need to provide a
`X-Telemetry-Client-Id` header specifying the clientId from the client
credentials obtained using the [/register](../requests/register.md) request.

## Format of the Telemetry Client Id Header
The `X-Telemetry-Client-Id` header value should be the string
representation of the int64 clientId value.