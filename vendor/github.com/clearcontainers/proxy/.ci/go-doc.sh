#!/bin/bash -e

set -m

godoc -http=:6061 &
xdg-open http://localhost:6061/pkg/github.com/clearcontainers/proxy/api/
fg
