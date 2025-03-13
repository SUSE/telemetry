# /report

Type: ***POST***

## Paramters

| Name | Type | Description | Example |
| ---- | ---- | ----------- | ------- |
| [Authorization](../headers/authorization.md) | header | A `Bearer` schema authorization | `Authorization: Bearer <JWT>`<br>Where \<JWT\> specifies the [JWT](https://jwt.io/) |
| [X-Telemetry-Registration-Id](../headers/telemetry-registration-id.md) | header | The registrationId of the telemetry client<br>submitting the report | `X-Telemetry-Registration-Id: 1234567890` |
| body | object | A [Telemetry Report](../structs/telemetryreport.md) containing<br>one or more [Telemetry Bundles](../structs/telemetrybundles.md) | See [Telemetry Report](../structs/telemetryreport.md) for more details. |

## Responses

| Code | Description | Example |
| ---- | ----------- | ------- |
| 200  | Success<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;processingId integer($int64)<br>&nbsp;&nbsp;processedAt string($rfc3339nano)<br>} | {<br>&nbsp;&nbsp;"processingId": 1234567890<br>&nbsp;&nbsp;"processedAt": "2024-08-01T01:02:03.000000Z"<br>} |
| 400  | Bad Request<br>Missing or incompatible body<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "missing header.reportId field"<br>} |
| 401  | Unauthorized<br>Client (re-)registration required due to one of:<br>- specified client is not registered<br>- invalid clientId provided<br>[WWW-Authenticate](../headers/www-authenticate.md) response header will specify recovery action<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "Client not registered"<br>} |

If the response is a 401 Unauthorized then the `WWW-Authenticate` reponse header will be set.
See [www-authenticate.md](../headers/www-authenticate.md) for more details.
