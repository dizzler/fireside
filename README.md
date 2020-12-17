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
3. Retrieving policies from an upstream control server (WIP)
4. Applying policies to events to create custom alerts (WIP)
5. Applying "alert actions" for in-line remediation-of / response-to detected issues (WIP)
6. Sending data:
    6a. forwarding events to configured event outputs (e.g. AWS S3)
    6b. firing alerts to configured alert outputs (e.g. AWS SNS) (WIP)


## LAYOUT
    env.helper.sh # helper file to source in order to set GO* env vars
                  # for (this) FireSide repo

    examples/     # directory of example configs for fireside, envoy, falco, etc.

    fireside.go   # use to "go run" the 'main' fireside program (for testing)

    README.ME     # this file

    go/           # directory for storing vendor modules via GOPATH

    go.mod        # Golang module file to edit to control vendor package management;
                  # manually EDIT go.mod to update FireSide dependencies

    go.sum        # Automatically updated from go.mod during "go build";
                  # DO NOT EDIT go.sum manually

    pkg/          # directory for storing FireSide code / packages

        configure/    # stores common constants;
                      # parses command-line flags for:
                      #     -config
                      #     -mode
                      # generates FireSide runtime vars from YAML -config file;

        envoy/        # directory for packages related to integrations
                      # with Envoy network proxy

            accesslog/    # code for getting Envoy accesslogs as input events

            xds/          # code for controlling Envoy resources through xDS API

        opa/          # directory for packages related to in-code implementation
                      # of Open Policy Agent (OPA) features and functions

            bundle/       # code for packaging and loading OPA "bundles" of
                          # of (.rego) modules and (.json|.yaml) data files

            policy/       # code for applying policies to input data using OPA
                          # queries to implement policy logic

            query/        # code for preparing and evaluating OPA queries

        pipeline/     # directory for packages related to
                      # Extract/Transform/Load (ETL) pipeline

            conduit/      # code for core pipeline types and functions

            data/         # code for common data management / manipulation

            events/       # code for event-specific ETL tasks

            processors/   # code for independent processors of pipeline data

        tls/        # code for creating TLS trust domains, CAs, certs and keys


## BUILD
Command from the root of this repository in order to build the 'fireside' binary:
> go build


## CONFIG
The FireSide application attempts to bootstrap/load its configuration from a file.

The default config file path is:
> /etc/fireside/config.yaml

Customize the config file path with the '-config' parameter at startup. For example:
> ./fireside -config examples/fireside.config.yaml


## RUN
Multiple run "modes" are available for the FireSide application, meaning that
FireSide's runtime behavior (i.e. mode) can be adapted to meet different use-cases.

Available run modes include:
> ca

^ (only) generates CA-signed TLS certificates

> server

^ default run mode;
^^collects events from server input(s) and applies data processing policies

Customize the run mode with the '-mode' parameter at startup. For example:
> ./fireside -mode ca

