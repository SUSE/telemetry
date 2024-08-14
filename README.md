# telemetry
SUSE Telemetry Client Library and associated client CLI tools

See [examples](examples/) directory for examples of how to use the
Telemetry Client library.

See the [doc][doc/] directory for details on the telemetry server
REST API and the fundamental concepts of the telemetry service.

# What's available

## cmd/generator
A simple CLI tool that can be used to add telemetry data items to the
local staging, optionally bundling them up into bundles, collecting them
into reports, and submitting them to the telemetry server.

## cmd/clientds
A simple CLI tool that can report status about the datastores used for
the local staging of telemetry data items, bundles and reports.

## pkg/client
The pkg/client module provides the following functionality:
* Client Regsitration
* Telemetry data item addition to local staging
* Local staging of telemetry bundles, created from locally staged data
  items
* Local staging of telemetry reports, created from locally staged bundles
* Submission of locally staged reports to the Telemetry Server

## pkg/config
The pkg/config module is used to parse client config files.

## pkg/restapi
The pkg/restapi module provides definitions for the client requests and
server reponses.

## pkg/types
The pkg/types module defined useful common types

## pkg/lib
The pkg/lib module provides functionality for managing the local staging
of data items, bundles and reports.

# Testing

## Verification Testing
The verification tests can be run from within the telemetry repo as follows:

```
% cd telemetry
% make test
```

## Local Developer Testing
First ensure that the SUSE/telemetry-server is running with the local
server config file. Then run the cmd/generator tool from the telemetry
tool as follows to generate telemetry data, including a DEVTEST tag,
and submit telemetry to the server, self-registering as a client with
the server if needed:

```
% cd telemetry/cmd/generator
% go run . --config ../../testdata/config/localClient.yaml \
      --telemetry=SLE-SERVER-SCCHwInfo --tag DEVTEST \
      ../../testdata/telemetry/SLE-SERVER-SCCHwInfo/sle12sp5-test.json
```

If you just want to generate but not submit, then you can include the
--nosubmit option.


# See Also
See the companion telemetry-server repo for a basic implementation of
a telemetry server to handle the requests generated by the telemetry
client tools.
