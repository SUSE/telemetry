# Authorization Header
The telemetry server expects telemetry clients to provide a valid
`Authorization` header when submiting telemetry reports via the
[/report](../requests/report.md) request.

## Format of Authorization Header
The `Authorization` header value will have the following format:

```
Bearer <authToken>
```

Where:
* `<authToken>` is the [JWT](https://jwt.io/) `authToken` field from
  the client credentials provided by the server in the response to a
  successful [/register](../requests/register.md) request.
