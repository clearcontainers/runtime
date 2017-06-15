#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if [ ! $(command -v gometalinter) ]
then
	go get github.com/alecthomas/gometalinter
	gometalinter --install --vendor --debug
fi

gometalinter \
	--exclude='error return value not checked.*(Close|Log|Print).*\(errcheck\)$' \
	--exclude='.*_test\.go:.*error return value not checked.*\(errcheck\)$' \
	--exclude='duplicate of.*_test.go.*\(dupl\)$' \
	--exclude='client.go:.*level can be fmt.Stringer' \
	--exclude='client.go:.*source can be fmt.Stringer' \
	--exclude='error: no formatting directive in Logf call \(vet\)$' \
	--disable=aligncheck \
	--disable=gotype \
	--disable=gas \
	--disable=vetshadow \
	--cyclo-over=15 \
	--tests \
	--deadline=600s \
	--vendor \
	./...
