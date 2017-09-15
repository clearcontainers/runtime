# Upgrading Clear Containers installations

This document discusses the available options for upgrading a system running Intel® Clear Containers.

## Terminology

There are currently two implementations of Clear Containers:

- Clear Containers 2.x (or just 2.x) refers to the previous implementation
  of Clear Containers which uses a runtime called `cc-oci-runtime`.
  2.x code is available at: https://github.com/01org/cc-oci-runtime

- Clear Container 3.x (or just 3.x) refers to the current implementation
  which uses a runtime called `cc-runtime`. 3.x code is available
  at: https://github.com/clearcontainers/runtime

## Maintenance warning

The 2.x codebase is no longer being developed. Only new releases will be
considered for significant bug fixes.

The main development focus is now on the 3.x branch. All 2.x
users are encouraged to switch to the 3.x implementation.

## Introduction

Clear Containers 2.x and 3.x share certain architectural elements. Both
use a shim process called `cc-shim` and a proxy process called
`cc-proxy`. However, the 2.x versions of these elements *are
incompatible* with their 3.x counterparts.

## Supported scenarios

This section details supported operations and configurations.

### Systems using pre-built packages

For operating systems with pre-built packages (such as Fedora* and
Ubuntu*), upgrading is possible with the following conditions:

- Configuration files are not migrated (see [Configuration files](#configuration-files)).
- It is necessary to perform a few cleanup steps as outlined in the
  subsections below.

Once the cleanup steps have been performed, simply follow the 3.x
installation guides and any existing packaged versions of the 2.x
components will be removed and the new 3.x components installed in their
place.

#### Upgrading an Ubuntu system

Assuming the system was installed by following the [CC 2.x Ubuntu installation guide](https://github.com/01org/cc-oci-runtime/blob/master/documentation/Installing-Clear-Containers-on-Ubuntu.md):

1. Perform the following cleanup steps:
   ```bash
   $ sudo rm -f /etc/systemd/system/docker.service.d/clr-containers.conf
   $ sudo apt-get purge -y docker-engine
   $ sudo rm -f /etc/apt/sources.list.d/cc-oci-runtime.list
   ```

1. Follow the [CC 3.x Ubuntu installation guide](ubuntu-installation-guide.md).

#### Upgrading a Fedora system

Assuming the system was installed by following the [CC 2.x Fedora installation guide](https://github.com/01org/cc-oci-runtime/blob/master/documentation/Installing-Clear-Containers-on-Fedora.md):

1. Perform the following cleanup steps:
   ```bash
   $ sudo rm -f /etc/systemd/system/docker.service.d/clr-containers.conf
   $ sudo rm -f /etc/yum.repos.d/home:clearlinux:preview:clear-containers-2.1.repo
   ```

1. Follow the [CC 3.x Fedora installation guide](fedora-installation-guide.md).

### Not Recommended

#### CentOS*

Since the CentOS installation instructions require the user to manually
build and install various packages, upgrading must also be performed
manually. This is not a recommended strategy: instead, consider
installing 3.x on a system which does not have 2.x installed.

## Unsupported scenarios

This section details operations and configurations that are not
supported.

### Running 2.x and 3.x on the same system

Running both 2.x and 3.x on the same system is explicitly not supported
due to the incompatible elements between the two systems.

If you are using the pre-built packages, installing the 3.x elements
will remove the 2.x ones (see [Systems using pre-built
packages](systems-using-pre-built-packages)).

### Configuration files

The 2.x runtime supported three different configuration files
(`vm.json`, `hypervisor.args` and `cc-oci-runtime.sh.cfg` for the
`cc-oci-runtime.sh` helper script).

The 3.x runtime uses a single configuration file (`configuration.toml`).

Note:

None of the 2.x configuration files are migrated or read by
the 3.x runtime. Any changes made to the 2.x configuration files will
**not** be carried over to the 3.x runtime.

### Downgrading

Systems using packaged versions of 3.x cannot be downgraded to 2.x
simply by following the 2.x installation instructions.

However, it should be possible to switch back to 2.x by first fully
removing 3.x and then following the 2.x [installation instructions](https://github.com/01org/cc-oci-runtime/wiki/Installation).

Using 2.x is now discouraged since it is no longer being developed
(see [Maintenance warning](#maintenance-warning)).
