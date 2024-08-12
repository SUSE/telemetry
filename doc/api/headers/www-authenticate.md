# WWW-Authenticate Header

If a telemetry server request fails with a 401 Unauthorized status code then the response will include a `WWW-Authenticate` header whose value will indicate how to resolve the authorization issue.

## Format of WWW-Authenticate header value
The `WWW-Authenticate` header value will have the following format:

```
<challenge> realm="<realm>" scope="<scope>"
```

Where:
* `<challenge>` is "Bearer"
* `<realm>` is "suse-telemetry-service"
* `<scope>` is either "authenticate" or "register"

## Appropriate authorization actions

| Scope | Handling |
| ----- | -------- |
| authenticate | the telemetry client should perform a [/authenticate](../requests/authenticate.md) request to obtain a new auth token |
| register | the telemetry client should perform a [/register](../requests/register.md) request, discarding any existing registration details |