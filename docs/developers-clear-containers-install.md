# Developers Clear Containers 3.0 Install

This guide is not intended for end-users. Instead, this guide provides
instructions for any developers eager to try Clear Containers 3.0 and who
want to build Clear Containers from the source code and are familiar with the
process.

## Initial setup

The recommended way to create a development environment is to first install the
packaged versions of the Clear Containers components to create a working
system:

  * [Fedora*](fedora-installation-guide.md)
  * [Ubuntu*](ubuntu-installation-guide.md)

The installation guide instructions will install all packaged
components, plus docker, the hypervisor and the Clear Containers image
and kernel.

## Requirements to build individual components

  * [go 1.8.3](https://golang.org/)
  * [gcc](https://gcc.gnu.org/) and associated C language build tooling
    such as `make`, `autoconf` and `libtool` which are required
    to build `cc-shim`

## Clear Containers 3.0 components

  * [Runtime](https://github.com/clearcontainers/runtime)
  * [Proxy](https://github.com/clearcontainers/proxy)
  * [Shim](https://github.com/clearcontainers/shim)
  * [Agent](https://github.com/clearcontainers/agent)

Since the installation guide will have installed packaged versions of
all required components, it is only necessary to install the source for
the component(s) you wish to develop with.

**IMPORTANT:** Do not combine [Clear Containers 2.x](https://github.com/01org/cc-oci-runtime) and [Clear Containers 3.x](https://github.com/clearcontainers) on the same system.
Both projects ship ``cc-proxy`` and ``cc-shim`` which are not compatible with each other.
See [the upgrading document](upgrading.md) for further details.

### Setup the environment

1. Define GOPATH

   ```bash
   $ export GOPATH=$HOME/go
   ```

2. Create GOPATH Directory

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

1. Proxy

   ```bash
   $ cd $GOPATH/src/github.com/clearcontainers/proxy
   $ make
   $ sudo make install
   ```

2. Shim

   ```bash
   $ cd $GOPATH/src/github.com/clearcontainers/shim
   $ ./autogen.sh
   $ make
   $ sudo make install
   ```

3. Runtime

   ```bash
   $ cd $GOPATH/src/github.com/clearcontainers/runtime
   $ make build-cc-system
   $ sudo -E PATH=$PATH make install-cc-system
   ```

4. Agent

   ```bash
   $ cd $GOPATH/src/github.com/clearcontainers/agent
   $ make
   ```

   The agent is installed inside the root filesystem image
   used by the hypervisor, hence to test a new agent version it is
   necessary to create a custom rootfs image. The example below
   demonstrates how to do this using the osbuilder tooling.

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
   /etc/clear-containers/configuration.toml
   
For more details on the runtime's build system, run:

```bash
$ make help
```

## See Also

  * [General Debugging](../README.md#debugging)
  * [Debugging the agent inside the hypervisor](debug-agent.md)
