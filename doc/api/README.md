# Telemetry Client REST API

A telemetry client is expected to support the following workflow:
* register with the upstream telemetry server to obtain client credentials
* submit telemetry reports using authorization obtained when registering as a client
* re-authenticate as needed if report submission fails with a 401 Unauthorized.

## Supported Requests

| Request | Description |
| ------- | ----------- |
| [/register](requests/register.md) | register a system as a telemetry client and obtain client credentials |
| [/report](requests/report.md) | submit a telemetry report containing one or more bundles of telemetry data |
| [/authenticate](requests/authenticate.md) | obtained refreshed client credentials |

# Registration
For a telemetry client to be able to register with an upstream telemetry
server, it will need to generate a registration value, which is used to
uniquely identify a given client system with the upstream server, and should
store this value in a secure fashion so that it can be accessed later when
(re-)authenticating with the upstream telemetry server.

When a telemetry client registers with the upstream telemetry server,
using the [/register](requests/register.md) request, it will send a request payload
containing the registration, and the successful response will provide
a set of client credentials as follows:

| Name | Type | Description |
| ---- | ---- | ----------- |
| registrationId | integer($int64) | ID used to identify the client system to the server |
| authToken | string($([jwt](https://jwt.io/)) | A JSON Web Token ([JWT](https://jwt.io/)) authorization token |
| registrationDate | string($[rfc3339nano](https://pkg.go.dev/time#pkg-constants)) | The client UTC registration timestamp expressed in<br>[RFC3339nano](https://pkg.go.dev/time#pkg-constants) format |

***NOTE***: The telemetry client is responsible for storing these client
credentials in a persistent fashion so that they can subsequently be
used to authenticate telemtry report submissions.

# Reporting Telemetry
For a telemetry client to report telemetry to an upstream telemetry
server it must prove that it has the authorization to do so. This is
achieved by supplying the appropriate request headers:

* [Authorization](headers/authorization.md)
* [X-Telemetry-Registration-Id](headers/telemetry-registration-id.md)

When a telemetry client submits a telemetry report to the upstream
telemetry server, using the [/report](requests/report.md)
request, it will send a request payload containing a
[TelemetryReport](structs/telemetryreport.md) object which holds one or
more [TelemetryBundle](structs/telemetrybundle.md) objects, each of which
may contain one or more [TelemetryDataItem](structs/telemetrydataitem.md)
objects.

# (Re-)Authentication
For a telemetry client to (re-)authenticate with an upstream telemetry
server, it will need to generate a supported hash, e.g. `sha256`, of the
registration to validate that it is in fact that client in question.

When a telemetry client (re-)authenticates with the upstream telemetry
server, using the [/authenticate](requests/authenticate.md) request, it will
send a request payload containing it's registratiionId and an registrationHash,
specifying the hash method and associated value, and the successful
response will provide a set of client credentials, the same as for a
[/register](requests/register.md) request, as described [above][#registration].