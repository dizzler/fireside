#!/bin/bash

_script_pwd="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

export GOPATH=${_script_pwd}/go/

export GOROOT=/usr/local/go/

export GOBIN=/usr/local/go/bin/

