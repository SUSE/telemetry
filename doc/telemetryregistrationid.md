# Telemetry Registration Ids

A telemetry server (or relay) manages a pool of telemetry client 
registration ids, currently ranging from 1 to MAX_INT64.

Once a client has successfully [registered](api/requests/register.md)
it's [client registration](telemetryclientregistration.md) with an
upstream telemetry service the response will contain the client's
credentials, including the registrationId which will be used to set
the [X-Telemetry-Registration-Id](api/headers/telemetry-registration-id.md)
when submitting telemetry requests to the upstream telemetry service.

Note that this telemetry registration id only has meaning in the
context of a specific combination of telemetry client and telemetry
server, and the same registration id may be assigned to multiple
telemetry clients so long as they are talking to different upstream
telemery servers (or relays).