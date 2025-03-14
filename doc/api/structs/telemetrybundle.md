# TelemetryBundle Data Structure

The TelemetryBundle data structure consists of the following sections:

* header - contains metadata about the telemetry report
  * bundleId - a UUID that differentiates this bundle from other bundles
    generated by the same client
  * bundleTimeStamp - the UTC timestamp for when the bundle was generated,
    formatted in [RFC3339nano](../../telemetrytimestamp.md) format
  * bundleClientId - the clientId of the telemetry client generating this
    bundle, as specified in the registraion used to successfully register
    the client with the upstream telemetry server.
  * bundleCustomerId - a string value specify the customer identifier,
    if any, associated with the telemetry client
  * bundleAnnotations - a possibly empty list of
    [telemetry annotation tags](../../telemetrytag.md)
* payload - a list of one or more [TelemetryDataItem](telemetrydataitem.md) objects
* footer - contains a checksum of the payload section

***NOTE***: All of the [telemetry data items](telemetrydataitem.md)
in a bundle must originate from the same telemetry client.

```
{
	header {
    bundleId          string
    bundleTimeStamp   string($rfc3339nano)
    bundleClientId    string
    bundleCustomerId  string
    bundleAnnotations [
      string...
    ]
  }
	telemetryDataItems [
    TelemetryDataItems...
  ]
	footer {
    checksum string
  }
}
```
