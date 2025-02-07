# Telemetry Client Registration

To submit telemetry reports to an upstream telemetry service (gateway
or relay) a client must [register](api/requests/register.md) with that
service. As part of the registration process a client will generate a
client registration structure which is used to uniquely identify it as
a telemetry service client.

## Client Registration Structure
This client registration structure has the following format:
```
{
  "clientId": "<client_generated_UUID>",
  "systemUUID": "<optional_system_UUID>",
  "timestamp": "<time_when_client_id_was_generated>"
}
```

## Client Registration Collisions
A client system may be required to generate a new client registration
if the upstream server detects that the registration is a duplicate
of an existing registration, or that the registration's clientId is
a duplicate of existing client's clientId.

The client registration's clientId value is used to identify the
client generating a [telemetry bundle](api/structs/telemetrybundle.md)
and submitting a [telemetry report](api/structs/telemetryreport.md).

## Uniquely Identifying a client
On its own the clientId may not always uniquely identify a client
within the overall pool of telemetry service submissions, because two
client systems could independently generate the same UUID value to use
as a clientId, but a telemetry client's clientId will always be unique
with respect to other clients of the same upstream telemetry service.

This property, that clientIds will always be unique with respect to
other clients of the same upstream telemetry service, can be leveraged
by [telemetry relays](telemetryrelay.md) to assist in uniquely
identifying telemetry clients.  When telemetry is relayed, the relay
will add a RELAYED_VIA tag to the telemetry submission which identifies
both the relay and the client that submitted the telemetry to the
relay. The aggregate of the RELATED_VIA tag values associated with a
telemetry data item can thus be used to uniquely identify the path
from the originating client to the main telemetry service gateway,
and thus uniquely identify a specific client's telemetry submissions.
