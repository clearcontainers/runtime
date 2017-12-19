[![Build Status](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-16-04-master/badge/icon)](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-16-04-master/)
[![Build Status](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-17-04-master/badge/icon)](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-17-04-master/)
[![Build Status](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-fedora-26-master/badge/icon)](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-fedora-26-master/)
[![Go Report Card](https://goreportcard.com/badge/github.com/clearcontainers/runtime)](https://goreportcard.com/report/github.com/clearcontainers/runtime)
[![Coverage Status](https://coveralls.io/repos/github/clearcontainers/runtime/badge.svg?branch=master)](https://coveralls.io/github/clearcontainers/runtime?branch=master)

# runtime

* [Introduction](#introduction)
* [License](#license)
* [Hardware requirements](#hardware-requirements)
* [Quick start for users](#quick-start-for-users)
* [Quick start for developers](#quick-start-for-developers)
* [Community](#community)
* [Configuration](#configuration)
* [Logging](#logging)
* [Debugging](#debugging)
* [Limitations](#limitations)
* [Home Page](#home-page)

## Introduction

`cc-runtime` is the next generation of Intel® Clear Containers runtime.

This tool, henceforth referred to simply as "the runtime", builds upon
the [virtcontainers](https://github.com/containers/virtcontainers)
project to provide a high-performance standards-compliant runtime that
creates hardware-virtualized containers which leverage
[Intel](https://www.intel.com/)'s VT-x technology.

It is a re-implementation of [`cc-oci-runtime`](https://github.com/01org/cc-oci-runtime) written in the go language and supersedes `cc-oci-runtime` starting from 3.0.0.

The runtime is both [OCI](https://github.com/opencontainers/runtime-spec)-compatible and [CRI-O](https://github.com/kubernetes-incubator/cri-o)-compatible, allowing it to work seamlessly with both Docker and Kubernetes respectively.

## License

The code is licensed under an Apache 2.0 license.

See [the license file](LICENSE) for further details.

## Hardware requirements

The runtime has a built-in command to determine if your host system is capable of running an Intel® Clear Container. Simply run:

```bash
$ cc-runtime cc-check
```

Note:

If you run the command above as the `root` user, further checks will be
performed (e.g. check if another incompatible hypervisor is running):

```bash
$ sudo cc-runtime cc-check
```

## Quick start for users

See the [installation guides](docs/) available for various operating systems.

## Quick start for developers

See the [developer's installation guide](docs/developers-clear-containers-install.md).

## Community

See [the contributing document](CONTRIBUTING.md).

## Configuration

The runtime uses a single configuration file called `configuration.toml`.
Since the runtime supports a [stateless system](https://clearlinux.org/features/stateless),
it checks for this configuration file in multiple locations. The default
location is `/usr/share/defaults/clear-containers/configuration.toml` for a
standard system. However, if `/etc/clear-containers/configuration.toml`
exists, this will take priority.

To see which paths the runtime will check for a configuration source, run:

```bash
$ cc-runtime --cc-show-default-config-paths
```

To see details of your systems runtime environment (including the location of the configuration file being used), run:

```bash
$ cc-runtime cc-env
```

## Logging

The runtime provides `--log=` and `--log-format=` options. However, you can
also configure it to log to the system log (syslog or `journald`) such that
all log data is sent to both the specified logfile and the system log. The
latter is useful as it is independent of the lifecycle of each container.

To view runtime log output:

```bash
$ sudo journalctl -t cc-runtime
```

To view shim log output:

```bash
$ sudo journalctl -t cc-shim
```

To view proxy log output:

```bash
$ sudo journalctl -t cc-proxy
```

Note:

The proxy log entries also include output from the agent (`cc-agent`) and the
hypervisor, which includes the guest kernel boot-time messages.

## Debugging

The runtime, the shim (`cc-shim`), the proxy (`cc-proxy`),
and the hypervisor all have separate `enable_debug=` debug
options in the [configuration file](#Configuration). All of these debug
options are disabled by default. See the comments in the installed
configuration file for further details.

If you want to enable debug for all components, assuming a standard configuration file path, run:

```bash
$ sudo sed -i -e 's/^#\(enable_debug\).*=.*$/\1 = true/g' /usr/share/defaults/clear-containers/configuration.toml
```

See the [agent debug document](docs/debug-agent.md) and the [kernel debug document](docs/debug-kernel.md) for further details.

## Limitations

See [the limitations file](docs/limitations.md) for further details.

## Home Page

The canonical home page for the project is: https://github.com/clearcontainers
