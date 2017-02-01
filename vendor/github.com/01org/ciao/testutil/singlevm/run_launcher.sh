#!/bin/bash

ciao_host=$(hostname)

#Obtain the subnet of the primary interface
default_if=$(ip route list | awk '/^default/ {print $5}')
default_subnet=$(ip -o -f inet addr show $default_if | awk '{print $4}')

#Cleanup artifacts
sudo "$GOPATH"/bin/ciao-launcher --alsologtostderr -v 3 --hard-reset

#Cleanup any prior docker instances and networks
sudo rm -f /var/lib/ciao/networking/docker_plugin.db

#Run launcher
sudo "$GOPATH"/bin/ciao-launcher \
    --cacert="$CIAO_DEMO_PATH"/CAcert-"$ciao_host".pem \
    --cert="$CIAO_DEMO_PATH"/cert-CNAgent-NetworkingAgent-"$ciao_host".pem -v 3  &
