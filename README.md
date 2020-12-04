# FireSide #

## COPYRIGHT
© 2020 FireEye, Inc. All rights reserved. Use of the below script, with or without
modification, is permitted only for paying customers of FireEye, and is subject
to the terms and conditions of the customer’s agreement with FireEye, including
without limitation all limitations on FireEye’s liability and damages.


## OVERVIEW
FireSide is a FireEye code resository for container runtime security.
FireSide is designed to bring FireEye/Mandiant intelligence to any environment via a
"sidecar" container pattern.

FireSide works by:
1. Collecting runtime security events from one or more inputs (e.g. Envoy, Falco)
2. Processing events into a standard format
3. Retrieving policies from an upstream control server
4. Applying policies to events to create custom alerts
5. Applying "alert actions" for in-line remediation-of / response-to detected issues
6. Sending data:
    6a. forwarding events to configured event outputs (e.g. AWS S3)
    6b. firing alerts to configured alert outputs (e.g. AWS SNS)

## LAYOUT
    fireside.go   # use to "go run" the 'main' fireside program

    README.ME     # this file

    examples/     # example configs for fireside, envoy, falco, etc.

    go/           # directory for storing vendor modules via GOPATH

    pkg/          # directory for storing FireSide code / packages

        configure/          # stores common constants;
                            # parses command-line flags for:
                            #     -config
                            #     -mode
                            # generates FireSide runtime vars from YAML -config file;

        envoy/              # code for FireSide integrations with Envoy proxy

                accesslog/      # code for getting Envoy accesslogs as input events

                xds/            # code for controlling Envoy resources through xDS API

        output_processors/  # code for loading pipeline events to configurable outputs

        pipeline/           #

        tls/                # code for creating TLS trust domains, CAs, certs and keys

        transformers/       # code for processing / transforming pipeline events

    ratchet/      # TEMPORARY workaround for bugs in upstream package

