#!/usr/bin/env sh

set -eux

go get -u github.com/karalabe/xgo
# OSX 10.9 is the most recent version available at time of writing.
# If changing the OS/arch target, adjust the filename in push-aws.sh.
xgo --image cockroachdb/xgo:20170112 --targets=windows-8.1/amd64 --go=1.7.3 --ldflags="$("$(dirname "${0}")"/ldflags.sh) -X github.com/cockroachdb/cockroach/pkg/build.typ=release" "${GOPATH%%:*}"/src/github.com/cockroachdb/cockroach
