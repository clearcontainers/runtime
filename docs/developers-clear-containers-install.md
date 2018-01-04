# Developers Clear Containers 3.0 Install

* [Initial setup](#initial-setup)
* [Requirements to build individual components](#requirements-to-build-individual-components)
* [Clear Containers 3.0 components](#clear-containers-3.0-components)
    * [Setup the environment](#setup-the-environment)
    * [Build and install components](#build-and-install-components)
        * [Proxy](#proxy)
        * [Shim](#shim)
        * [Runtime](#runtime)
        * [Agent](#agent)
        * [Kernel](#kernel)
* [See Also](#see-also)

This guide is not intended for end-users. Instead, this guide provides
instructions for any developers eager to try Clear Containers 3.0 and who
want to build Clear Containers from the source code and are familiar with the
process.

## Initial setup

The recommended way to create a development environment is to first install the
packaged versions of the Clear Containers components to create a working
system:

  * [CentOS*](centos-installation-guide.md)
  * [Clear Linux*](clearlinux-installation-guide.md)
  * [Fedora*](fedora-installation-guide.md)
  * [SLES*](sles-installation-guide.md)
  * [Ubuntu*](ubuntu-installation-guide.md)

The installation guide instructions will install all required Clear Containers
components, plus Docker*, the hypervisor, and the Clear Containers image and
kernel.

## Requirements to build individual components

  * [go 1.8.3](https://golang.org/).
  * [gcc](https://gcc.gnu.org/) and associated C language build tooling
    (e.g. `make`, `autoconf` and `libtool`) are required
    to build `cc-shim`.

## Clear Containers 3.0 components

  * [Runtime](https://github.com/clearcontainers/runtime)
  * [Proxy](https://github.com/clearcontainers/proxy)
  * [Shim](https://github.com/clearcontainers/shim)
  * [Agent](https://github.com/clearcontainers/agent)

Since the installation guide will have installed packaged versions of
all required components, it is only necessary to install the source for
the component(s) you wish to develop with.

**IMPORTANT:** Do not combine [Clear Containers 2.x](https://github.com/01org/cc-oci-runtime) and [Clear Containers 3.x](https://github.com/clearcontainers) on the same system.
Both projects ship `cc-proxy` and `cc-shim` which are not compatible with each other.
See [the upgrading document](upgrading.md) for further details.

### Setup the environment

1. Define `GOPATH`

   ```bash
   $ export GOPATH=$HOME/go
   ```

2. Create `GOPATH` Directory

   ```bash
   $ mkdir -p $GOPATH
   ```

3. Get the code

   ```bash
   $ go get -d github.com/clearcontainers/runtime
   $ go get -d github.com/clearcontainers/proxy
   $ git clone https://github.com/clearcontainers/shim $GOPATH/src/github.com/clearcontainers/shim
   $ go get -d github.com/clearcontainers/agent
   $ go get -d github.com/clearcontainers/tests
   $ go get -d github.com/clearcontainers/osbuilder
   ```

### Build and install components

#### Proxy

```bash
$ cd $GOPATH/src/github.com/clearcontainers/proxy
$ make
$ sudo make install
```

Note: the previous install step will overwrite any proxy binary installed from
the `cc-proxy` package.

#### Shim

```bash
$ cd $GOPATH/src/github.com/clearcontainers/shim
$ ./autogen.sh
$ make
$ sudo make install
```

Note: the previous install step will overwrite any shim binary installed from
the `cc-shim` package.

#### Runtime

```bash
$ cd $GOPATH/src/github.com/clearcontainers/runtime
$ make build-cc-system
$ sudo -E PATH=$PATH make install-cc-system
```

The previous install step will create `/usr/local/bin/cc-runtime`. This
ensures the packaged version of the runtime from the `cc-runtime` package is
not overwritten. However, since the installation guide configured Docker to
call the runtime as `/usr/bin/cc-runtime`, it is necessary to modify the
docker configuration to make use of your newly-installed development runtime
binary:

```bash
$ sudo sed -i 's!cc-runtime=/usr/bin/cc-runtime!cc-runtime=/usr/local/bin/cc-runtime!g' /etc/systemd/system/docker.service.d/clear-containers.conf
```

For more details on the runtime's build system, run:

```bash
$ make help
```

#### Agent

```bash
$ cd $GOPATH/src/github.com/clearcontainers/agent
$ make
```

The agent is installed inside the root filesystem image
used by the hypervisor, hence to test a new agent version it is
necessary to create a custom rootfs image. The example below
demonstrates how to do this using the
	 [osbuilder](https://github.com/clearcontainers/osbuilder) tooling.

```bash
$ cd $GOPATH/src/github.com/clearcontainers/osbuilder

$ # Create a rootfs image
$ sudo -E make rootfs USE_DOCKER=true

$ # Overwrite the default cc-agent binary with a custom built version
$ sudo cp $GOPATH/src/github.com/clearcontainers/agent/cc-agent ./workdir/rootfs/usr/bin/cc-agent

$ # Generate a container.img file
$ sudo -E make image USE_DOCKER=true

$ # Install the custom image
$ sudo install --owner root --group root --mode 0755 workdir/container.img /usr/share/clear-containers/

$ # Update the runtime configuration
$ # (note that this is only an example using default paths).
$ sudo sed -i.bak -e 's!^\(image = ".*"\)!# \1 \
image = "/usr/share/clear-containers/container.img"!g' \
/usr/share/defaults/clear-containers/configuration.toml
```

#### Kernel

The latest kernel for Clear Linux can be found on
[the releases page](https://github.com/clearcontainers/linux/releases).
The Clear Linux kernel can be used to rebuild and modify a custom VM
container kernel as needed. The example below demonstrates how to do this
using the [osbuilder](https://github.com/clearcontainers/osbuilder) tooling.

```bash
$ cd $GOPATH/src/github.com/clearcontainers/osbuilder

$ # Fetch latest kernel sources from github.com/clearcontainers/linux
$ make kernel-src

$ # Optionally modify kernel sources or config at $PWD/workdir/linux

$ # build kernel
$ make kernel

$ # Install the custom image
$ sudo install --owner root --group root --mode 0755 workdir/vmlinuz.container /usr/share/clear-containers/custom-vmlinuz
$ sudo install --owner root --group root --mode 0755 workdir/vmlinux.container /usr/share/clear-containers/custom-vmlinux

$ # Update the runtime configuration
$ # (note that this is only an example using default paths).
$ # Note: vmlinuz is used for pc platform type.
$ #       vmlinux is used for pc-lite and q35-lite platform type.
$ sudo sed -i.bak -e 's!^\(kernel = ".*"\)!# \1\nkernel = "/usr/share/clear-containers/custom-vmlinuz"!g' \
/usr/share/defaults/clear-containers/configuration.toml
```

## See Also

  * [General Debugging](../README.md#debugging)
  * [Debugging the agent inside the hypervisor](debug-agent.md)
  * [Debugging the kernel inside the hypervisor](https://github.com/clearcontainers/runtime/blob/master/docs/debug-kernel.md)
  * Packaging
    The repository used to create the packaged versions of Clear Containers
    components for various Linux distributions is:
      * https://github.com/clearcontainers/packaging
