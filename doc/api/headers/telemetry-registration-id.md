# Telemetry Registration Id Header
When a telemetry client is submitting a telemetry report using the
[/report](../requests/report.md) request it will need to provide a
`X-Telemetry-Registration-Id` header specifying the registrationId
from the client credentials obtained using the
[/register](../requests/register.md) request.

## Format of the Telemetry Registration Id Header
The `X-Telemetry-Registration-Id` header value should be the string
representation of the int64 registrationId value.