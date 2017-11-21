# Using devices with Clear Containers

You can use the docker run command with the --device option to pass host 
devices to a container. The following example passes a device:
```
$ sudo docker run -it --device=/dev/sdb debian
```
Clear Containers supports passing the following devices to the
container with `--device`.

## Virtual Function I/O (VFIO) Devices (PCI Passthrough)

The VFIO driver is a framework for exposing direct device access to userspace, 
in a secure, Input–Output Memory Management Unit (IOMMU) protected environment.
A device can be directly passed to a Clear Containers container using VFIO. 

For information on VFIO, refer to the following links:
 - https://www.kernel.org/doc/Documentation/vfio.txt
 - https://www.linux-kvm.org/images/b/b4/2012-forum-VFIO.pdf

Devices can pe passed to a Clear Container container using VFIO
as shown below:

```
$ sudo docker run -it --device=/dev/vfio/16 centos/tools bash
```
You are required to unbind the device from its host driver, bind the device
to `vfio-pci` driver and then pass the IOMMU group that the device belongs to on
the docker command line.
For detailed steps to bind and unbind devices, see https://github.com/containers/virtcontainers#how-to-pass-a-device-using-vfio-passthrough


## Block Devices

Clear Containers passes block devices to containers using `virtio-block` when you
use --device on the docker command line to pass block devices.
The file system that is present on the block device should be enabled in the 
Clear Container kernel. We currently have support for ext4, xfs and overlay file
systems in the Clear Container kernel.

The following example passes a block device using a fake image:

```
$ # Create fake image
$ fallocate -l 256K /tmp/test.img

$ mkfs.ext4 -F /tmp/test.img

$ # Find a free loop device
$ sudo losetup -f
/dev/loop2

$ # Use the available loop device returned by the command above
$ sudo losetup /dev/loop2 /tmp/test.img

$ # Pass the loop mounted block device to the container.
$ docker run --device=/dev/loop2 busybox stat /dev/loop2
 File: /dev/sdc
   Size: 0         	Blocks: 0          IO Block: 4096   block special file
```

The following example shows device rename support for block devices:
```
$ sudo docker run --device=/dev/loop2:/dev/sdc busybox stat /dev/sdc
root@2aa92d0bb8e0:/# mount /dev/sdc /mnt
root@2aa92d0bb8e0:/# ls /mnt
lost+found
```
