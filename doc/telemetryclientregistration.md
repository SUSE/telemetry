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
  "clientId": "<SCC_assigned_or_client_generated_UUID>",
  "systemUUID": "<optional_system_UUID>",
  "timestamp": "<time_when_registration_id_was_generated>"
}
```

## Client Registration Collisions
A client system may be required to generate a new client registration
if the upstream server detects that the registration is a duplicate
of an existing registration.

The client registration's clientId value is used to identify the
client generating a [telemetry bundle](api/structs/telemetrybundle.md)
and submitting a [telemetry report](api/structs/telemetryreport.md).

## Uniquely identifying telemetry submitted by a client
On its own the clientId may not always uniquely identify a client
within the overall pool of telemetry service submissions because,
while extremely unlikely, there is a chance that two independent
client systems could generate the same UUID value for use as a
telemetry client id.

Note that in the case of client systems registered with the SCC,
the risk of duplicates will be further reduced because telemetry
client ids will be assigned by the SCC and will be unique with
respect to other registered SCC clients at that time.

A [telemetry relay](telemetryrelay.md) will include a RELAYED_VIA
tag specifying the client id that submitted the telemetry report,
and the telemetry relay server's telemetry client id. Combining
any RELAYED_VIA tag values associated with a telemetry submission
along with the telemetry client id of the client that generated the
telemetry should further allow for uniquely identifying telemetry
submitted by a specific client.
