[![Build Status](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-16-04/badge/icon)](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-16-04/)
[![Build Status](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-17-04/badge/icon)](http://cc-jenkins-ci.westus2.cloudapp.azure.com/job/clear-containers-runtime-azure-ubuntu-17-04/)
[![Go Report Card](https://goreportcard.com/badge/github.com/clearcontainers/runtime)](https://goreportcard.com/report/github.com/clearcontainers/runtime)
[![Coverage Status](https://coveralls.io/repos/github/clearcontainers/runtime/badge.svg?branch=master)](https://coveralls.io/github/clearcontainers/runtime?branch=master)

# runtime

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

## Debugging

To provide a persistent log of all container activity on the system, the runtime
offers a global logging facility. By default, this feature is disabled
but can be enabled with a simple change to the [configuration](#Configuration) file.

First, to determine the configuration file path for your host run:

```bash
$ cc-runtime cc-env | grep -A 2 'Runtime.Config.Location'
```

To enable the global log:

```bash
$ sudo sed -i -e 's/^#\(\[runtime\]\|global_log_path =\)/\1/g' $path_to_your_config_file
```

The path to the global log file can be determined subsequently by running:

```bash
$ cc-runtime cc-env | grep GlobalLogPath
```

Note that it is also possible to enable the global log by setting the
`CC_RUNTIME_GLOBAL_LOG` environment variable to a suitable path. The
environment variable takes priority over the configuration file.

The global logfile path must be specified as an absolute path. If the
directory specified by the logfile path does not exist, the runtime will
attempt to create it.

It is the Administrator's responsibility to ensure there is sufficient
space for the global log.

## Limitations

See [the limitations file](docs/limitations.md) for further details.

## Home Page

The canonical home page for the project is: https://github.com/clearcontainers
