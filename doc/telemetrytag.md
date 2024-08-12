# Telemetry Annotation Tag

When a [JSON telemetry blob](telemetryblob.md) is provided by a telemetry
source it can optionally be accompanied by annotation tags.

These annotation tags can have one of the following formats:

* \<tagname\>
* \<tagname\>=<tagvalue\>

***NOTE***: Annotation tags are not required when generating telemetry.

These tags are intended to provide environmental information that may
be relevant for grouping telemetry submissions, that is not otherwise
captured in the telemetry itself.

Annotation tags can be associated with
[Telemetry Data Items](api/structs/telemetrydataitem.md),
[Telemetry Bundles](api/structs/telemetrybundle.md) or
[Telemetry Reports](api/structs/telemetryreport.md), with the following
inheritance rules:

* [Telemetry Bundles](api/structs/telemetrybundle.md) inherit any tags
  associated with the [Telemetry Report](api/structs/telemetryreport.md)
  they were a part of.
* [Telemetry Data Items](api/structs/telemetrydataitems.md) inherit any tags
  associated with the [Telemetry Bundle](api/structs/telemetrybundle.md)
  they were a part of.

## Telemetry Annotation Tag Examples
Some examples:
* when bundles are relayed via a telemetry relay server a RELAYED_VIA
  tag composed of the combination of the client id of the reporting
  client and the relay server's client id will be added to the relayed
  bundles.
* telemetry relayed via a proxy service may be annotated by the proxy
  type, e.g
  * PROXY_TYPE=RMT for RMT
  * PROXY_TYPE=SUMA for SUSE Manager
  * PROXY_TYPE=SCC for SCC