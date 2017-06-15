#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Number of containers must be provided as a command line argument"
    exit 1
fi

for i in $(seq -f "%03g" 1 $1)
do
  cat > /tmp/container-$i.yaml <<EOF
---
start:
  requested_resources:
     - type: vcpus
       value: 1
     - type: mem_mb
       value: 16
  instance_uuid: 67d86208-b46c-4465-9018-e14187d4$i
  tenant_uuid: 67d86208-000-4465-9018-fe14087d415f
  docker_image: ubuntu:latest
  vm_type: docker
  networking:
    vnic_mac: CA:FE:00:02:02:03
    vnic_uuid: 67d86208-b46c-0000-9018-fe14087d415f
    concentrator_ip: 192.168.200.200
    concentrator_uuid: 67d86208-b46c-4465-0000-fe14087d415f
    subnet: 192.168.111.0/24
    private_ip: 192.168.111.$i
...
---
#cloud-config
runcmd:
  - [ /usr/bin/python3, -m, http.server]
...
---
{
  "uuid": "67d86208-b46c-4465-0000-fe14087d415f",
  "hostname": "ciao"
}
...
EOF
  ciaolc startf /tmp/container-$i.yaml
  rm /tmp/container-$i.yaml
done


