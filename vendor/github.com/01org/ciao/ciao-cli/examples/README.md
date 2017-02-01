# workload examples

A workload can be uploaded into the ciao datastore via the ciao-cli
command.

`ciao-cli workload create -yaml my_workload.yaml`

This directory includes some sample workloads that illustrate a few
basic workload definitions. A workload consists of a yaml file which
defines the properties and resources the workload needs, as well as a
yaml file which contains the cloud config to use when launching the instance.
section.

## workload properties yaml

The workload properties yaml file contains information which define the
workload. This configuration will be used for each instance that is started
with the workload ID.

The first section of the file contains some basic information about the
workload.

```
description: "My awesome container workload"
vm_type: docker
image_name: "ubuntu:latest"
```

`description` is just a human readable string which will be
displayed when you list workloads with `ciao-cli workload list`.

There are two types of workloads - container based, and vm based. The `vm_type`
string specifies whether this workload is a container or a VM. Currently ciao
supports only docker based containers, and qemu for vms. Valid values for
vm_type are `docker` or `qemu`.

The above example is for a container workload. The `vm_type` is set to `docker`
and the docker hub image name is provided by the `image_name` value.

For VMs, the configuration would be slightly different. `description` is still
provided, however, `vm_type` is set to `qemu`. Additionally, a `fw_type` is
required to specify whether the image that is provided has a UEFI based
firmware, or a legacy firmware. If the workload should be booted from an
image, the image must have already been added to the ciao image service.
The `image_id` is the UUID of the image, and can be obtained with `ciao-cli
image list`.

```
description: "My favorite pet VM"
vm_type: qemu
fw_type: legacy
image_id: "73a86d7e-93c0-480e-9c41-ab42f69b7799"
```

Workloads may also be specified to boot from volumes created in ciao's volume
service. To create a workload which boots from an existing volume, leave the
image_id blank and include a definition for a bootable disk.

```
description: "My favorite pet VM"
vm_type: qemu
fw_type: legacy
disks:
- volume_id: "9c858de5-fdd3-42d8-925e-2fdc60768d24"
  bootable: true
```

Disks for the workload may be bootable, or attached. To create a new volume
to be attached to your workload for persistent storage, specify the size
of your new disk in GigaBytes. Newly created disks may be marked as ephemeral
if you wish to not persist the disk after the instance has been destroyed.

```
disks:
- size: 20
  ephemeral: true
```

To attach a volume that has already been created in the ciao volume service,
specify the volume_id of the volume to be attached.

```
disks:
- volume_id: "9c858de5-fdd3-42d8-925e-2fdc60768d24"
```

You can specify in the workload definition whether to create a volume
to either attach or boot from that is cloned from an existing volume or
image by including a `source` definition.

```
disks:
- bootable: true
  source:
     service: volume
     id: "9c858de5-fdd3-42d8-925e-2fdc60768d24"
```

Valid values for the source `service` field are `image` or `volume`

Workload definitions must also contain default values for resources
that the workload will need to use when it runs. There are two resources
which must be specified:

```
defaults:
    vcpus: 2
    mem_mb: 512
```

Finally, the filename for the cloud config file must be included in the
workload definition. This file must be readable by ciao-cli.

```
cloud_init: "fedora_vm.yaml"
```

## cloud config yaml file

Below is an example ciao cloud config file. ciao cloud config files have a
cloud-config section, followed by a user data section. The cloud-config section
must be prefaced by the `---` separator.

```
---
#cloud-config
users:
  - name: demouser
    gecos: CIAO Demo User
    lock-passwd: false
    passwd: $6$rounds=4096$w9I3hR4g/hu$AnYjaC2DfznbPSG3vxsgtgAS4mJwWBkcR74Y/KHNB5OsfAlA4gpU5j6CHWMOkkt9j.9d7OYJXJ4icXHzKXTAO.
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDerQfD+qkb0V0XdQs8SBWqy4sQmqYFP96n/kI4Cq162w4UE8pTxy0ozAPldOvBJjljMvgaNKSAddknkhGcrNUvvJsUcZFm2qkafi32WyBdGFvIc45A+8O7vsxPXgHEsS9E3ylEALXAC3D0eX7pPtRiAbasLlY+VcACRqr3bPDSZTfpCmIkV2334uZD9iwOvTVeR+FjGDqsfju4DyzoAIqpPasE0+wk4Vbog7osP+qvn1gj5kQyusmr62+t0wx+bs2dF5QemksnFOswUrv9PGLhZgSMmDQrRYuvEfIAC7IdN/hfjTn0OokzljBiuWQ4WIIba/7xTYLVujJV65qH3heaSMxJJD7eH9QZs9RdbbdTXMFuJFsHV2OF6wZRp18tTNZZJMqiHZZSndC5WP1WrUo3Au/9a+ighSaOiVddHsPG07C/TOEnr3IrwU7c9yIHeeRFHmcQs9K0+n9XtrmrQxDQ9/mLkfje80Ko25VJ/QpAQPzCKh2KfQ4RD+/PxBUScx/lHIHOIhTSCh57ic629zWgk0coSQDi4MKSa5guDr3cuDvt4RihGviDM6V68ewsl0gh6Z9c0Hw7hU0vky4oxak5AiySiPz0FtsOnAzIL0UON+yMuKzrJgLjTKodwLQ0wlBXu43cD+P8VXwQYeqNSzfrhBnHqsrMf4lTLtc7kDDTcw== ciao@ciao
...
```

The above example shows that the user data section is blank, but the `...`
separator must be present to indicate the end of the cloud-config section.
Be sure to use a cloud-config that is recognized by the host OS you are using
for your workload.
