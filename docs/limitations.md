# Clear Containers known differences and limitations

* [Pending items](#pending-items)
    * [Docker swarm support](#docker-swarm-support)
    * [Networking](#networking)
        * [Adding networks dynamically](#adding-networks-dynamically)
    * [Resource management](#resource-management)
        * [`docker run --cpus=`](#docker-run---cpus)
        * [`docker run --kernel-memory=`](#docker-run---kernel-memory)
        * [shm](#shm)
        * [cgroup constraints](#cgroup-constraints)
        * [Capabilities](#capabilities)
        * [sysctl](#sysctl)
        * [tmpfs](#tmpfs)
    * [Other](#other)
        * [checkpoint and restore](#checkpointandrestore)
        * [`docker stats`](#docker-stats)
    * [runtime commands](#runtime-commands)
        * [`ps` command](#pscommand)
        * [`events` command](#events-command)
        * [`update` command](#update-command)
  * [Architectural limitations](#architectural-limitations)
      * [Networking](#networking)
        * [Support for joining an existing VM network](#support-for-joining-an-existing-vmnetwork)
        * [`docker --net=host`](#docker---nethost)
        * [`docker run --link`](#docker-run---link)
    * [Host resource sharing](#host-resource-sharing)
        * [`docker --device`](#docker---device)
        * [`docker -v /dev/...`](#docker--v-dev)
        * [`docker run --privileged`](#docker-run---privileged)
    * [Other](#other)
        * [Annotations](#annotations)
    * [runtime commands](#runtime-commands)
        * [`init` command](#init-command)
        * [`spec` command](#spec-command)
* [Other notes](#other-notes)
    * [Host `rdmsr` warnings](#host-rdmsr-warnings)

As Intel® Clear Containers utilises Virtual Machines (VM) to enhance
security and isolation of container workloads, the `cc-runtime` has a
number of differences and limitations when compared with the standard
Docker* runtime, `runc`. Some of these limitations have potential
solutions, whereas others exist due to fundamental architectural
differences generally related to the use of VMs.

The Clear Container runtime launches each container within its own
hardware isolated VM, and each VM has its own kernel. Due to this higher
degree of isolation, there are certain container capabilities that
cannot be supported or are implicitly enabled via the VM.

The below sections describe in brief the known limitations, and where
applicable further links off to the relevant open Issues or pages with
more detailed information.

## Pending items

This section lists items that may technically be fixable:

### Docker swarm support

The newest version of Docker supported is specified by the `docker_version`
variable in the
[versions.txt](https://github.com/clearcontainers/runtime/blob/master/versions.txt)
file.

However, if you wish to use Docker's swarm facility, an older version of Docker is
required. This is specified by the `docker_swarm_version` variable in the
[versions.txt](https://github.com/clearcontainers/runtime/blob/master/versions.txt)
file.

See issue [\#771](https://github.com/clearcontainers/runtime/issues/771) for more
information.

### Networking

#### Adding networks dynamically

The runtime does not currently support adding networks to an already
running container (`docker network connect`).

The VM network configuration is set up with what is defined by the CNM
plugin at startup time. It would be possible to watch the networking
namespace on the host to discover and propagate new networks at runtime
but, it is not implemented today.

See `cc-oci-runtime` issue [\#388](https://github.com/01org/cc-oci-runtime/issues/388) for more information.

### Resource management

Due to the way VMs differ in their CPU and memory allocation and sharing
across the host system, the implementation of an equivalent method for
these commands is potentially challenging.

#### `docker run --cpus=`

The `docker run --cpus=` option is not currently implemented. At the
runtime level, this equates to the `linux.resources.cpu` OCI
configuration. It should be possible to pass this information through to
the QEMU command line CPU configuration options to gain a similar
effect.

Note that the `--cpu-quota` and `--cpu-period` `docker run` options are
supported; in combination, these two options can provide most of the
functionality that `--cpus` would offer.

See issue [\#341](https://github.com/clearcontainers/runtime/issues/341) for more information.

#### `docker run --kernel-memory=`

The `docker run --kernel-memory=` option is not currently implemented.
It should be possible to pass this information through to the QEMU
command line CPU configuration options to gain a similar effect.

See issue [\#388](https://github.com/clearcontainers/runtime/issues/388) for more information.

#### shm

The runtime does not implement the `docker run --shm-size` command to
set the size of the `/dev/shm tmpfs` size within the container. It
should be possible to pass this configuration value into the VM
container and have the appropriate mount command happen at launch time.

#### cgroup constraints

Docker supports cgroup setup and manipulation generally through the
`run` and `update` commands. With the use of VMs in Clear Containers,
the mapping of cgroup functionality to VM functionality is not always
straight forward.

For information on specific support, see the relevant sub-sections
within this document.

Generally, support can come down to a number of methods:

- Implement support inside the VM/container

- Implement support wrapped around the VM/container

- Potentially a combination or sub-set of both inside and outside the
  VM/container

- No implementation necessary, as the VM naturally provides equivalent
  functionality

#### Capabilities

The `docker run --cap-[add|drop]` commands are not supported by the
runtime. At the runtime level, this equates to the
`linux.process.capabilities` OCI configuration. Similar to the cgroup
items, these capabilities could be modified either in the host, in the
VM, or potentially both.

See issue [\#51](https://github.com/clearcontainers/runtime/issues/51) for more information.

#### sysctl

The `docker run --sysctl` feature is not implemented. At the runtime
level, this equates to the `linux.sysctl` OCI configuration. Docker
allows setting of sysctl settings that support namespacing. It may make
sense from a security and isolation point of view to set them in the VM
(effectively isolating sysctl settings). Also given that each Clear
Container has its own kernel, we can support setting of sysctl settings
that are not namespaced. In some cases, we may need to support setting
some of the settings both on the host side Clear Container namespace as
well as the Clear Containers kernel.

#### tmpfs

The `docker run --tmpfs` command is not supported by the runtime. Given
the nature of a tmpfs, it should be feasible to implement this command
as something passed through to the VM kernel startup in order to set up
the appropriate mount point.

### Other

#### checkpoint and restore

The runtime does not provide `checkpoint` and `restore` commands. There
have been discussions around using VM save and restore to give `criu`
like functionality, and it is feasible that some workable solution may
be achievable.

Note that the OCI standard does not specify `checkpoint` and `restore`
commands.

See `cc-oci-runtime` issue [\#22](https://github.com/01org/cc-oci-runtime/issues/22) for more information.

#### `docker stats`

The `docker stats` command does not return meaningful information for
Clear Containers at present. This requires the runtime to support the
`events` command.

Some thought needs to go into if we display information purely from
within the VM, or if the information relates to the resource usage of
the whole VM container as viewed from the host. The latter is likely
more useful from a whole system point of view.

Note that the OCI standard does not specify a `stats` command.

See issue [\#200](https://github.com/clearcontainers/runtime/issues/200) for more information.

### runtime commands

#### `ps` command

The Clear Containers runtime does not currently support the `ps` command.

Note, this is *not* the same as the `docker ps` command. The runtime `ps`
command lists the processes running within a container. The `docker ps`
command lists the containers themselves. The runtime `ps` command is
invoked from `docker top`.

Note that the OCI standard does not specify a `ps` command.

See issue [\#95](https://github.com/clearcontainers/runtime/issues/95) for more information.

#### `events` command

The runtime does not currently implement the `events` command. We may
not be able to perfectly match every sub-part of the `runc` events
command, but we can probably do a subset, and maybe add some VM specific
extensions.

See here for the
[runc implementation](https://github.com/opencontainers/runc/blob/e775f0fba3ea329b8b766451c892c41a3d49594d/events.go).

Note that the OCI standard does not specify an `events` command.

See issue [\#379](https://github.com/clearcontainers/runtime/issues/379) for more information.

#### `update` command

The runtime does not currently implement the `update` command, and hence
does not support some of the `docker update` functionality. Much of the
`update` functionality is based around cgroup configuration.

It may be possible to implement some of the update functionality by
adjusting cgroups either around the VM or inside the container VM
itself, or possibly by some other VM functional equivalent. It needs
more investigation.

Note that the OCI standard does not specify an `update` command.

See issue [\#380](https://github.com/clearcontainers/runtime/issues/380) for more information.

## Architectural limitations

This section lists items that may not be fixed due to fundamental
architectural differences between "soft containers" (traditional Linux
containers) and those based on VMs.

### Networking

#### Support for joining an existing VM network

Docker supports the ability for containers to join another containers
namespace with the `docker run --net=containers` syntax. This allows
multiple containers to share a common network namespace and the network
interfaces placed in the network namespace.  Clear Containers does not
support network namespace sharing. If a Clear Container is setup to
share the network namespace of a `runc` container, the runtime
effectively takes over all the network interfaces assigned to the
namespace and binds them to the VM. So the `runc` container will lose
its network connectivity.

#### `docker --net=host`

Docker host network support (`docker --net=host run`) is not supported.
It is not possible to directly access the host networking configuration
from within the VM.

The `--net=host` option can still be used with `runc` containers and
inter-mixed with with running `cc-runtime` containers, thus still
enabling use of `--net=host` when necessary.

It should be noted, currently passing the `--net=host` option into a
Clear Container may result in the Clear Container networking setup
modifying, re-configuring and therefore possibly breaking the host
networking setup. Do not use `--host=net` with Clear Containers.

#### `docker run --link`

The runtime does not support the `docker run --link` command. This
command is now effectively deprecated by docker, so we have no
intentions of adding support. Equivalent functionality can be achieved
with the newer docker networking commands.

See more documentation at
[docs.docker.com](https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/).

### Host resource sharing

#### `docker --device`

Support has been added to pass devices using [Virtual Function I/O (VFIO)](https://www.kernel.org/doc/Documentation/vfio.txt) 
passthrough. Devices that support the Input–Output Memory Management Unit (IOMMU)
feature can be assigned to the `vfio-pci` driver and passed
on the docker command line with `--device=/dev/vfio/$(iommu_group_number)`, 
where `iommu_group_number` is the IOMMU group that the device belongs to.
If multiple devices belong to the same IOMMU group, they will all be 
assigned to the Clear Containers VM.

Support for passing other devices including block devices with `--device`
is not yet avilable.

#### `docker -v /dev/...`

Docker volume support for devices (`docker run -v /dev/foo`) is not
supported for similar reasons to the `--device` option. Note however
that non-device file volume mounts are supported.

#### `docker run --privileged`

The `docker run --privileged` command is not supported in the runtime.
There is no natural or easy way to grant the VM access to all of the
host devices that this command would need to be complete.

The `--privileged` option can still be used with `runc` containers and
inter-mixed with with running `cc-runtime` containers, thus still
enabling use of `--privileged` when necessary.

### Other

#### Annotations

OCI Annotations are nominally supported, but the OCI specification is
not clear on their purpose. Note that the annotations are not exposed
inside the Clear Container.

### runtime commands

#### `init` command

The runtime does not implement the `init` command as it has no useful
equivalent in the VM implementation, and appears to be primarily for
`runc` internal use.

#### `spec` command

The runtime does not implement the `spec` command. `runc` provides such
a command that generates a JSON-format template specification file that
is useable by the Clear Containers runtime. The addition of a `spec`
command to the Clear Containers runtime would just be duplication that
would likely always be playing catchup with `runc`.

## Other notes

This section contains other useful information realated to limitations,
warnings, etc. to aid in understanding and diagnosis of what is a known
problem, a new problem, or not a problem.

### Host `rdmsr` warnings

When you run Clear Containers, you might see some warnings in the host
`dmesg` log reporting `unhandled rdmsr`, such as:

```
[  319.406575] kvm [2415]: vcpu0 unhandled rdmsr: 0x1c9
[  319.483944] kvm [2415]: vcpu0 unhandled rdmsr: 0x64e
[  319.483966] kvm [2415]: vcpu0 unhandled rdmsr: 0x34

```

These warnings are generated when the Virtual Machine, via `QEMU`, makes
requests to read `MSR` registers from the VM/host that are not supported
by `QEMU/KVM`. In the case of Clear Containers, these warnings are
normally related to unsupported perf event counters.

By default, these benign warnings appear as 'red warnings' in the dmesg
logs, which can be disconcerting for end users and sysadmins.

A patch has been merged into the `v4.15` kernel that should improve
this situation:

https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/arch/x86/kvm/x86.c?id=fab0aa3b776f0a3af1db1f50e04f1884015f9082

It might take some time for those changes to be implemented in
a host distro kernel.

