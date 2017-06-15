#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Number of VMs must be provided as a command line argument"
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
       value: 128
  instance_uuid: 67d86208-b46c-4465-9018-fe14087d$i
  tenant_uuid: 67d86208-000-4465-9018-fe14087d415f
  image_uuid: clear-5590-cloud-supernova.qcow
  networking:
    vnic_mac: 02:00:fa:69:71:d0
    vnic_uuid: 00d86208-b46c-0000-9018-fe14087d415f
    concentrator_ip: 192.168.42.21
    concentrator_uuid: 67d86208-b46c-4465-0000-fe14087d415f
    subnet: 192.168.8.0/21
    private_ip: 192.168.8.$i
...
---
#cloud-config
runcmd:
  - [ touch, "/etc/bootdone" ]
users:
  - name: demouser
    gecos: CIAO Demo User
    lock-passwd: false
    passwd: \\$1\\$vzmNmLLD\\$04bivxcjdXRzZLUd.enRl1
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDerQfD+qkb0V0XdQs8SBWqy4sQmqYFP96n/kI4Cq162w4UE8pTxy0ozAPldOvBJjljMvgaNKSAddknkhGcrNUvvJsUcZFm2qkafi32WyBdGFvIc45A+8O7vsxPXgHEsS9E3ylEALXAC3D0eX7pPtRiAbasLlY+VcACRqr3bPDSZTfpCmIkV2334uZD9iwOvTVeR+FjGDqsfju4DyzoAIqpPasE0+wk4Vbog7osP+qvn1gj5kQyusmr62+t0wx+bs2dF5QemksnFOswUrv9PGLhZgSMmDQrRYuvEfIAC7IdN/hfjTn0OokzljBiuWQ4WIIba/7xTYLVujJV65qH3heaSMxJJD7eH9QZs9RdbbdTXMFuJFsHV2OF6wZRp18tTNZZJMqiHZZSndC5WP1WrUo3Au/9a+ighSaOiVddHsPG07C/TOEnr3IrwU7c9yIHeeRFHmcQs9K0+n9XtrmrQxDQ9/mLkfje80Ko25VJ/QpAQPzCKh2KfQ4RD+/PxBUScx/lHIHOIhTSCh57ic629zWgk0coSQDi4MKSa5guDr3cuDvt4RihGviDM6V68ewsl0gh6Z9c0Hw7hU0vky4oxak5AiySiPz0FtsOnAzIL0UON+yMuKzrJgLjTKodwLQ0wlBXu43cD+P8VXwQYeqNSzfrhBnHqsrMf4lTLtc7kDDTcw== ciao@ciao
...
---
{
  "uuid": "ciao",
  "hostname": "ciao"
}
...
EOF
  ciaolc startf /tmp/container-$i.yaml
  rm /tmp/container-$i.yaml
done
