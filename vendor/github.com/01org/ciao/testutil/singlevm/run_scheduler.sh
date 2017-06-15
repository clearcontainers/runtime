#!/bin/bash

ciao_host=$(hostname)

sudo -E "$GOPATH"/bin/ciao-scheduler \
    --cacert="$CIAO_DEMO_PATH"/CAcert-"$ciao_host".pem \
    --cert="$CIAO_DEMO_PATH"/cert-Scheduler-"$ciao_host".pem -v 3 &
