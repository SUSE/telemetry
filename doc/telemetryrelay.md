# Telemetry Relay

A telemetry relay is both a telemetry client of an upstream telemetry
server, as well as being a telemetry server that registered clients can
submit telemetry to, and is responsible for relaying the telemetry that
it receives from registered clients to the upstream server.

## Overview

When a client submits a [telemetry report](api/structs/telemetryreport.md)
to a telemetry relay using a [/report request](api/requests/report.md) it
will be processed as follows:

1. The telemetry relay will process the telemetry report to extract
   the [telemetry bundles](api/structs/telemetrybundle.md).
2. The telemetry relay will annotate each bundle with a new 
   [RELAYED_VIA tag](telemetrytag.md) whose value consists of the
   [registration Id](telemetryregistrationid.md) of the client that
   submitted the report, and the telemetry relay's own registration Id,
   joined with a `:`.
3. The telemetry relay will stage the received telemetry bundles
   locally.
4. Once sufficient bundles are available, or enough time has passed,
   the telemetry server will submit one or more /report requests to
   the upstream server containing the locally staged bundles.
5. Upon successfully submitting the telemetry to the upstream server
   the telemetry server can delete the locally staged telemetry bundles.

## Persistent Staging and Aggregation

One of the primary goals of a telemetry relay is to aggregate the
telemetry bundles it has received from registered clients into
reports that it sends to the upstream server.

To support this it needs to be able to stage received telemetry
bundles in a persistent fashion so that they can be aggregated into
a future telemetry report.

# Non-standard Telemetry Relay Scenarios

The following scenarios outline non-standard telemetry relay scenarios
which will deviate from the normal telemetry relay processing workflow.

## Synthesized Client Submissions

Situations may arrise where a client system doesn't report telemetry
directly, and instead it's telemetry submissions may be synthesised
by management frame work, such as SUSE Manager, and submitted to the
upstream telemetry server as though they had been received from the
client.

In such cases the the management framework will need to do the
following:

1. Assign a persistent unique telemetry client id to each of the
   client systems that it will be synthesizing telemetry for. 
2. Generate a [telemetry bundle](api/structs/telemetrybundle.md)
   with that client system's telemetry client id set as the
   bundleClientId in the header, containing the desired
   [telemetry data item(s)](api/structs/telemetrydataitem.md).
3. Ensure that the telemetry bundle is annotated with a
   RELAYED_VIA tag whose value is the client system's assigned
   telemetry client id and the management framework's telemetry
   client id, joined with a `:`.
4. Either send the appropriate generated telemetry bundle as
   the sole content of a telemetry report, via a /report request,
   or preferrably, aggregate multiple telemetry bundles into a
   single telemetry report before sending it.

## Relay with no persistent storage

In the case where a telemetry relay is running without any persistent
storage available, it will not be able to stage received bundles for
later submission, nor can it complete the received /report request
until it has submitted the received telemetry to the upstream telemetry
server.

For this scenario the telemetry relay will need to proxy the /report
requests that it receives, with the handler having to process the
request as follows:

1. Extract the telemetry bundle(s) from the incoming /report request.
2. Annotate the telemetry bundle(s) with a RELAYED_VIA tag whose
   value is the reporting client's id and the relay's client id,
   joined with a `:`.
3. Submit a new outgoing /report request to the upstream server,
   retrying at most once if appropriate.
4. If the upstream /report request succeeds, then successfully
   complete the original incoming /report request from the client.
   Otherwise fail the incoming /report request; the client should
   retry again at a later time.