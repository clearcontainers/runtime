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

Since the installation guide will have installed packaged versions of
all required components, it is only necessary to install the source for
the component(s) you wish to develop with.

**IMPORTANT:** Do not combine [Clear Containers 2.1](https://github.com/01org/cc-oci-runtime) and [Clear Containers 3.0](https://github.com/clearcontainers).
Both projects ship ``cc-proxy`` and they are not compatible with each other.

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
   $ go get -d github.com/clearcontainers/tests
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

For more details on the runtime's build system, run:

```bash
$ make help
```
