[![Build Status](https://travis-ci.org/clearcontainers/runtime.svg?branch=master)](https://travis-ci.org/clearcontainers/runtime)
[![Build Status](https://semaphoreci.com/api/v1/clearcontainers/runtime/branches/master/shields_badge.svg)](https://semaphoreci.com/clearcontainers/runtime)
[![Go Report Card](https://goreportcard.com/badge/github.com/clearcontainers/runtime)](https://goreportcard.com/report/github.com/clearcontainers/runtime)
[![Coverage Status](https://coveralls.io/repos/github/clearcontainers/runtime/badge.svg?branch=master)](https://coveralls.io/github/clearcontainers/runtime?branch=master)

# runtime

## Introduction

`cc-runtime` is the next generation Intel® Clear Containers runtime.

This tool, henceforth referred to simply as "the runtime", builds upon
the [virtcontainers](https://github.com/containers/virtcontainers)
project to provide a high-performance standards-compliant runtime that
creates hardware-virtualized containers which leverage
[Intel](https://www.intel.com/)'s VT-x technology.

It is a re-implementation of [`cc-oci-runtime`](https://github.com/01org/cc-oci-runtime) written in the go language and supersedes `cc-oci-runtime` starting from 3.0.0.

The runtime is both [OCI](https://github.com/opencontainers/runtime-spec)-compatible and [CRI-O](https://github.com/kubernetes-incubator/cri-o)-compatible, allowing it to work seamlessly with both Docker and Kubernetes respectively.

## License

The code is licensed under an Apache 2.0 license.

See [the license file](https://github.com/clearcontainers/runtime/blob/master/LICENSE) for further details.

## Hardware requirements

The runtime has a built-in command to determine if your host system is capable of running a Clear Container. Simply run:

```bash
$ cc-runtime cc-check
```

## Quick start for developers

See the [developer's installation guide](https://github.com/clearcontainers/runtime/blob/master/docs/developers-clear-containers-install.md).

## Community

See [the contributing document](https://github.com/clearcontainers/runtime/blob/master/CONTRIBUTING.md).

## Configuration

The runtime uses a single configuration file called `configuration.toml` which is normally located at `/etc/clear-containers/configuration.toml`.

To see details of your systems runtime environment (including the location of the configuration file), run:

```bash
$ cc-runtime cc-env
```

## Debugging

To provide a persistent log of all container activity on the system, the runtime
provides a global logging facility. By default, this feature is disabled.

To enable the global log:

```bash
$ sudo sed -i -e 's/^#\(\[runtime\]\|global_log_path =\)/\1/g' /etc/clear-containers/containers.toml
```

Note: The configuration file on your system may be located at a different path. To
determine the configuration file path for your host, run:

```bash
$ cc-runtime cc-env | grep -A 2 'Runtime.Config.Location'
```

The path to the global log file can be determined subsequently by
running:

```bash
$ cc-runtime cc-env | grep GlobalLogPath
```

Note that it is also possible to enable the global log by setting  the
`CC_RUNTIME_GLOBAL_LOG` environment variable to a suitable path. The
environment variable takes priority over the configuration file.

The global logfile path must be specified as an absolute path. If the
directory specified by the logfile path does not exist, the runtime will
attempt to create it.

It is the Administrator's responsibility to ensure there is sufficient
space for the global log.

## Home Page

The canonical home page for the project is: https://github.com/clearcontainers
