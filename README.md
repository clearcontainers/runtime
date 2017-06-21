[![Build Status](https://travis-ci.org/clearcontainers/runtime.svg?branch=master)](https://travis-ci.org/clearcontainers/runtime)
[![Build Status](https://semaphoreci.com/api/v1/clearcontainers/runtime/branches/master/shields_badge.svg)](https://semaphoreci.com/clearcontainers/runtime)
[![Go Report Card](https://goreportcard.com/badge/github.com/clearcontainers/runtime)](https://goreportcard.com/report/github.com/clearcontainers/runtime)
[![Coverage Status](https://coveralls.io/repos/github/clearcontainers/runtime/badge.svg?branch=master)](https://coveralls.io/github/clearcontainers/runtime?branch=master)

# runtime

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
