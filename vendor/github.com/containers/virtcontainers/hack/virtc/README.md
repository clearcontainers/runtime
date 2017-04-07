# virtc

`virtc` is a simple command-line tool that serves to demonstrate typical usage of the `virt` API.
This is example software; unlike other projects like runc, runv, or rkt, `virt` is not a full container runtime.

## Virtc example

Here we explain how to use the pod API from __virtc__ command line.

### Prepare your environment

#### Get your kernel

_Fedora_
```
$ sudo -E dnf config-manager --add-repo http://download.opensuse.org/repositories/home:clearlinux:preview:clear-containers-2.0/Fedora_24/home:clearlinux:preview:clear-containers-2.0.repo
$ sudo dnf install linux-container 
```

_Ubuntu_
```
$ sudo sh -c "echo 'deb http://download.opensuse.org/repositories/home:/clearlinux:/preview:/clear-containers-2.0/xUbuntu_16.04/ /' >> /etc/apt/sources.list.d/cc-oci-runtime.list"
$ sudo apt install linux-container
```

#### Get your image

It has to be a recent Clear Linux image to make sure it contains hyperstart binary.
You can dowload the following tested [image](https://download.clearlinux.org/releases/12210/clear/clear-12210-containers.img.xz), or any version more recent.

```
$ wget https://download.clearlinux.org/releases/12210/clear/clear-12210-containers.img.xz
$ unxz clear-12210-containers.img.xz
$ sudo cp clear-12210-containers.img /usr/share/clear-containers/clear-containers.img
```

#### Get virtc

_Download virtc_
```
$ go get github.com/containers/virtcontainers
```

_Build and install pause binary_
```
$ cd $GOPATH/src/github.com/containers/virtcontainers
$ make
$ make install
```

_Go to virtc_
```
$ cd hack/virtc
```

### Run virtc

All following commands needs to be run as root. Currently, __virtc__ only starts single container pods.

_Create your container bundle_

As an example we will create a busybox bundle:

```
$ mkdir -p /tmp/bundles/busybox/
$ docker pull busybox
$ cd /tmp/bundles/busybox/
$ mkdir rootfs
$ docker export $(docker create busybox) | tar -C rootfs -xvf -
$ echo -e '#!/bin/sh\ncd "\"\n"sh"' > rootfs/.containerexec
$ echo -e 'HOME=/root\nPATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\nTERM=xterm' > rootfs/.containerenv
```

#### Run a new pod (Create + Start)
```
./virtc pod run --agent="hyperstart" --network="CNI" --proxy="ccProxy"
```
#### Create a new pod
```
./virtc pod create --agent="hyperstart" --network="CNI" --proxy="ccProxy"
```
This should generate that kind of output
```
Created pod 306ecdcf-0a6f-4a06-a03e-86a7b868ffc8
```

#### Start an existing pod
```
./virtc pod start --id=306ecdcf-0a6f-4a06-a03e-86a7b868ffc8
```

#### Stop an existing pod
```
./virtc pod stop --id=306ecdcf-0a6f-4a06-a03e-86a7b868ffc8
```

#### Get the status of an existing pod and its containers
```
./virtc pod status --id=306ecdcf-0a6f-4a06-a03e-86a7b868ffc8
```
This should generate the following output:
```
POD ID                                  STATE   HYPERVISOR      AGENT
306ecdcf-0a6f-4a06-a03e-86a7b868ffc8    running qemu            hyperstart

CONTAINER ID    STATE
```

#### Delete an existing pod
```
./virtc pod delete --id=306ecdcf-0a6f-4a06-a03e-86a7b868ffc8
```

#### List all existing pods
```
./virtc pod list
```
This should generate that kind of output
```
POD ID                                  STATE   HYPERVISOR      AGENT
306ecdcf-0a6f-4a06-a03e-86a7b868ffc8    running qemu            hyperstart
92d73f74-4514-4a0d-81df-db1cc4c59100    running qemu            hyperstart
7088148c-049b-4be7-b1be-89b3ae3c551c    ready   qemu            hyperstart
6d57654e-4804-4a91-b72d-b5fe375ed3e1    ready   qemu            hyperstart
```
