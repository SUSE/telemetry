# Telemetry Client Ids

A telemetry server (or relay) manages a pool of telemetry client ids,
current ranging from 1 to MAX_INT64.

When a client [registers](api/requests/register.md) with an upstream
telemetry server (or relay) it is assigned a new client id.

This telemetry client id only has meaning in the context of a specific
combination of telemetry client and telemetry server, and the same
client id can be assigned to multiple telemetry clients so long as they
are talking to different upstream telemery servers (or relays).