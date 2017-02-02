#!/bin/bash
ciao_host=$(hostname)

sudo rm ciao-controller.db-shm ciao-controller.db-wal ciao-controller.db /tmp/ciao-controller-stats.db

sudo "$GOPATH"/bin/ciao-controller \
    --cacert="$CIAO_DEMO_PATH"/CAcert-"$ciao_host".pem \
    --cert="$CIAO_DEMO_PATH"/cert-Controller-"$ciao_host".pem --v 3 &
