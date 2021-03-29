#!/bin/sh
# NOTE: This is meant to be run inside the repo root as ./tools/check-license-headers.sh

echo "Checking files for license header..."

WRONG_FORMAT=$(addlicense -f tools/license_header.txt --check $(git ls-files --exclude '*_mock.go' '*.go') | sort)

# if gofmt found no files with wrong formatting, exit ok
[ -z "$WRONG_FORMAT" ] && exit 0

# else print and fail
printf >&2 "Files with missing license header found found!\n\n${WRONG_FORMAT}\n\nPlease add license header!"
exit 1
