#!/bin/sh
# NOTE: This is meant to be run inside the repo root as ./tools/add-license-headers.sh

addlicense -f tools/license_header.txt $(find . -type f -name '*.go' -not -path '*_mock.go')
