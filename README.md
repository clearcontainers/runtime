[![Build Status](https://travis-ci.org/clearcontainers/runtime.svg?branch=master)](https://travis-ci.org/clearcontainers/runtime)
[![Build Status](https://semaphoreci.com/api/v1/clearcontainers/runtime/branches/master/shields_badge.svg)](https://semaphoreci.com/clearcontainers/runtime)
[![Go Report Card](https://goreportcard.com/badge/github.com/clearcontainers/runtime)](https://goreportcard.com/report/github.com/clearcontainers/runtime)
[![Coverage Status](https://coveralls.io/repos/github/clearcontainers/runtime/badge.svg?branch=master)](https://coveralls.io/github/clearcontainers/runtime?branch=master)

# runtime

## Global logging

Additional to the (global) `--log=` option, the runtime also has the
concept of a global logfile.

The purpose of this secondary logfile is twofold:

- To allow all log output to be recorded in a non-container-specific
  path.

  This is particularly useful under container managers such as Docker
  where if a container fails to start, the container-specific directory
  (which contains the `--log=` logfile) will be deleted on error, making
  debugging a challenge.

- To collate the log output from all runtimes in a single place.

The global logfile comprises one line per entry. Each line contains the
following fields separated by colons:

- timestamp
- PID
- program name
- log level
- log message

The global logfile records all log output sent to the standard logfile.

Note that if output is disabled for the standard logfile, the global log
will still record all logging calls the runtime makes.

The global logfile is disabled by default. It can be enabled either in
the `configuration.toml` configuration file or by setting the
`CC_RUNTIME_GLOBAL_LOG` environment variable to a suitable path. The
environment variable takes priority over the configuration file.

The global logfile path must be specified as an absolute path. If the
directory specified by the logfile path does not exist, the runtime will
attempt to create it.

It is the Administrator's responsibility to ensure there is sufficient
space for the global log.
