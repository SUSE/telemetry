# Telemetry Type

When a [JSON telemetry blob](telemetryblob.md) is provided by a telemetry
source it must be accompanied by a telemetry type.

The telemetry type is a string consisting of 3 or more alphanumeric words
seperated by `-` characters, as follows:

```
<product_family>-<product_stream>-<telemetry_subtype>
```

Where:
* \<product_family\> identifies the product family that the telemetry is assocuated with
* \<product_stream\> identifies the relevent product stream within that product family
* \<telemetry_subtype\> identifies the specific telemetry subtype being reported

***NOTE***: The \<telemetry_subtype\> value can contain additional `-` characters if desired.

Some fictional examples:
* "SLE-Server-HwInfo"
* "ECM-Rancher-Clusters"