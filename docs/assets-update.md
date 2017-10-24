# Clear Containers assets updates 

Clear Containers requires additional resources to create a virtual machine
container. These resources are called
[Clear Container assets](https://github.com/clearcontainers/runtime/blob/master/docs/architecture/architecture.md#assets)
which are a `kernel` and a root `filesystem` image. This document describes when
these components are updated.

## Clear Containers Kernel

The [Clear Containers kernel](https://github.com/clearcontainers/linux) is a
Linux\* kernel based on the latest vanilla version of the 
[Longterm kernel](https://www.kernel.org/) and the includes
[patches](https://github.com/clearlinux-pkgs/linux-container/) necessary to run
Clear Containers. The `Longterm` branch is only updated with
[important bug fixes](https://www.kernel.org/category/releases.html)
and in turn using this branch ensures fewer required updates.

### Clear Containers Kernel update

Each time a new kernel version is rolled out it is updated in the Clear
Containers packaging repository and the [Clear Containers Linux
kernel](https://github.com/clearcontainers/linux).  On each Clear Containers
release the latest version in this repository is used as the recommended kernel
for the new Clear Containers version.

### Clear Containers Image

The [Clear Containers image](https://github.com/clearcontainers/runtime/blob/master/docs/architecture/architecture.md#root-filesystem-image)
known as the "mini O/S" is produced from Clear Linux\* operating system
packages/bundles. The image is generated multiple times a day as part of the
Clear Linux release cycle. 

### Update Clear Containers image

The Clear Containers image is updated only when critical updates are done to the
packages used by the Clear Containers guest OS:

- Systemd
- [Clear Containers Agent](https://github.com/clearcontainers/agent)
- iptables
- core-utils

This is verified each release using the
[`get-image-changes.sh`](https://github.com/clearcontainers/packaging/blob/master/scripts/get-image-changes.sh)
script.

The image must be updated in the [Clear Containers packaging
repository](https://github.com/clearcontainers/packaging) and defined in the
[versions.txt](https://github.com/clearcontainers/runtime/blob/master/versions.txt)
file.

## Clear Containers Continuous integration

Official Clear Containers Packages are hosted in the OBS build system. In order
to avoid availability issues the kernel and image for the Clear Containers
continuous integration system used are downloaded from Clear Linux\* packages:

```
https://download.clearlinux.org/releases/${CLEAR_VERSION}/clear/x86_64/os/Packages/linux-container-${KERNEL_VERSION}.rpm
https://download.clearlinux.org/releases/${CLEAR_VERSION}/clear/clear-${CLEAR_VERSION}-containers.img.xz
```

### Security

In the case of critical updates to the kernel or the Clear Containers image a
new Clear Containers release will be rolled out immediately.
