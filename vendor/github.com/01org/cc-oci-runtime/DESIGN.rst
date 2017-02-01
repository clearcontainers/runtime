.. contents::
.. sectnum::

Design of ``cc-oci-runtime``
============================

Overview
--------

This document outlines the architecture of ``cc-oci-runtime``
along with implementation details for the curious or those
wishing to get involved in the project.

Henceforth, the tool will simply be referred to as "the runtime".

Introduction
------------

The runtime is written in ANSI C.

The design has been dictated largely by the Open Containers Initiative
(OCI_) "runtime" specification which continues to evolve. As such, the
code is not as simple as it once was in earlier iterations of the
specification.

Since at the time of writing the reference implementation of the OCI_
specification (runc_) deviates from the specification itself and since
that reference implementation is the default runtime used by Docker_,
this runtime strives to be compatible with both. This further
complicates the code.

Summary
-------

At the most basic level, the runtime performs the following steps:

- Reads command-line arguments and options.
- Reads the `OCI configuration file`_.
- Starts a virtual machine.
- Runs the required workload command inside the virtual machine.
- Stops the virtual machine.

Dependencies
------------

The runtime is written in ANSI C and makes heavy use of the GLib_
library. This was chosen for its prevalence, flexibility, comprehensive
documentation and test suite. Since JSON is also used heavily, the
accompanying JSON-GLib_ library was also adopted since it shares the
same set of attributes as GLib_.

Quick Start
-----------

The first places to start to become familiar with the code are:

- The ``main()`` function.
- The ``oci.h`` header file.

The code is fully documented using special comments that are parseable
by the excellent Doxygen_ tool. See README.rst for details of generating
and viewing the extensive code documentation.

Code layout
-----------

- The OCI runtime code lives below the ``src/`` directory.
- The proxy code lives below the ``proxy/`` directory.
- The shim code lives below the ``shim/`` directory.
- Tests:

  - Unit test code lives in the directory ``tests/``.
  - Functional tests live below the directory ``tests/functional/``.
  - Integration tests live below the directory ``tests/integration/``.

Coding style and strategy
-------------------------

- Runtime and shim (in C):

  - The style of the code is similar to that used by the `Linux kernel`_.
  - The code is written to be as clean and readable as possible.
  - Use of "``goto``" is recommended for simplifying error handling and
    avoiding duplicated code.
  - All functions must be documented with a `Doxygen`_ header.
  - All function parameters must be checked and an error returned when an
    unexpected value is found.

- Considerations specific to the runtime:

  - Functions relating to a particular sub-system are separated into their own
    sub-system-specific file and optional header file.
  - A sub-system should expose the smallest possible interface (all other
    functions and data should be "``static``").
  - All sub-system interfaces must be accompanied with unit tests.  For
    example, subsystem "``src/${subsystem}.c``" must have an accompanying
    "``tests/${subsystem}_test.c``". This is a minimum - ideally all functions
    should have a unit test (to test a private function, replace "``static``"
    with "``private``").
  - Most unit tests functions accept an ``cc_oci_config`` object. This is the
    main object which encapsulates the contents of the `OCI configuration
    file`_ along with runtime-specific data.
  - Where possible, all command-line commands and options should be accompanied
    by a functional test. See `How command-line commands are implemented`_.
  - The BATS_ test framework is used for functional and integration tests.

- Proxy (in Go):

  - The usual Go style, enforced by `gofmt`, should be used.
  - The `Go Code Review`_ document contains a few additional useful guidelines.

Files
-----

OCI configuration file
~~~~~~~~~~~~~~~~~~~~~~

The OCI JSON configuration file, ``config.json`` (but represented in the
code by ``CC_OCI_CONFIG_FILE``) is passed to the ``create`` command is
parsed by ``cc_oci_config_file_parse()`` which loads the file into a
tree of ``GNode``'s. This function then calls
``cc_oci_process_config()`` which iterates over the tree and calls
special "handler" functions for each node. This logic is encapsulated by
``spec_handler`` objects which define the name of the node they operate
on and a function to call to handle the node.

The spec handlers used to parse the configuration file for container
creation are encapsulated in the ``start_state_handlers`` array, whilst
those used to stop a container are encapsulated in the
``stop_state_handlers`` array.

Each ``spec_handler`` is defined in a separate file below
`src/spec_handlers/`_.

For example, the ``spec_handler`` to parse the `OCI config root object`_
is `src/spec_handlers/root.c`_.

State file
~~~~~~~~~~

Not all runtime commands are provided with the `OCI configuration
file`_, so when the runtime's ``create`` command is called, it
creates a persistent file containing state information that can be read
by subsequent invocations of the runtime when passed different commands.

The state file is represented by ``CC_OCI_STATE_FILE`` and created by
the ``cc_oci_state_file_create()`` function.

Other commands read the state file into an ``oci_state`` object using
the ``cc_oci_state_file_read()`` function.

Like the `OCI configuration file`_, the state file is loaded into a
``GNode`` tree and has an array of ``spec_handler`` objects deal with
individual JSON objects. The state file spec handlers are encapsulated
in the ``state_handlers`` array.

Note that the ``cc_oci_config`` object includes a similar object in the
form of ``cc_oci_container_state``. But whereas the ``create`` command
has access to the complete ``cc_oci_config`` object, other commands
rely on the partial information provided in the ``oci_state`` object.

However, some part of the code require a ``cc_oci_config`` object, so a
function called ``cc_oci_config_update()`` can be called to create a
partial (but valid) ``cc_oci_config`` object from a ``oci_state`` object.

Configuration files
~~~~~~~~~~~~~~~~~~~

``hypervisor.args``
...................

The ``CC_OCI_HYPERVISOR_CMDLINE_FILE`` file is used to specify the
arguments to use to launch the hypervisor. This file is read by the
``cc_oci_vm_args_get()`` function which also expands the special tags
(variables) which can be included in the file. The expansions are
handled by the ``cc_oci_expand_cmdline()`` function.

``vm.json``
...........

The ``CC_OCI_VM_CONFIG`` file is a valid JSON fragment that is used to
supplement the data provided by the `OCI configuration file``; if that
file does not contain the required virtual machine configuration, the
runtime will attempt to read that from ``CC_OCI_VM_CONFIG`` using the
``get_spec_vm_from_cfg_file()`` function.

Log Files
~~~~~~~~~

See Logging_.

How command-line commands are implemented
-----------------------------------------

The runtime supports the `OCI runtime commands`_ along with additional
commands supported by runc_.

- Every command-line command (or "sub-command") is implemented in its own
  separate file below the `src/commands/`_ directory.
- Each command must define a ``subcommand`` object which specifies:

  - The name of the command as specified on the command-line.
  - A description that will be displayed in usage output.
  - An optional array of command-line options the command accepts.
  - A handler function called when the user specified the command on the command-line.

- Most `OCI runtime commands`_ have a corresponding function (prefixed
  with "``cc_oci``") in `src/oci.c`_.

For a simple example, see `src/commands/version.c`_ which is the
implementation for::

  $ cc-oci-runtime version

All command-line commands should have a corresponding functional test.
For example, the ``version`` command has a BATS_ functional test at
`tests/functional/version.bats`_.

Logging
-------

Message logging is handled by calling the ``cc_oci_log_init()``
function. The code makes heavy use of the GLib_ logging calls such as
``g_critical()``, ``g_warning()`` and ``g_debug()``.

The logging code actually writes to up to *two* files; if a command
specifies the ``--log`` option, all logging calls with write data to
this file. However, since Docker passes this option and sets the path
to the log to a container-specific directory, it is also possible to
specify the ``--global-log`` option to any command regardless of
whether ``--log`` has been specified. The global log is always
written in ASCII format and allows for a single log to be maintained
which all containers can write to if desired.

By default, only a few messages will be written to either log under
normal operation. However, if ``--debug`` is specified, the number of
messages logged rises significantly so care should be taken to ensure
that sufficient disk space is available for the logs and that log files
are rotated and compressed for long-running and/or busy systems.

All writes to either log file are atomic. If no log command-line option
is specified, no logging will occur. If logging fails, the runtime will
attempt to log using ``syslog(3)``.

Code Flow with Docker 1.12
--------------------------

Docker 1.12 Architectural Overview
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This sections gives a broad overview of how Docker 1.12 interacts with
the runtime.

The simplest example to consider is what happens when the user runs::

  $ docker run -ti busybox

The following is a simplified UML sequence diagram showing how the
individual elements interact::

    +------+  +-------+  +----------+
    |docker|  |dockerd|  |containerd|
    +------+  +-------+  +----------+
        |         |           |
  "run" +-------->|           |
        |         +---------->|         +---------------+
        |         |           +-------->|containerd-shim|
        |         |           |         +-------+-------+
        |         |           |                 |          +--------------+
        |         |           |                 |--------->|cc-oci-runtime| "create"
        |         |           |                 |          +------+-------+
        |         |           |                 |                 |
        |         |           |                 |                 | fork()      +---------+
        |         |           |                 |                 +------------>|qemu-lite|
        |         |           |                 |                 |             +------+--+
        |         |           |                 |                 |                    |
        |         |           |                 |                 | write state        |     +-----+
        |         |           |                 |                 +--------------------|---->|state|
        |         |           |                 |                 |                    |     +-----+
        |         |           |                 |                 | exit()             |        ^
        |         |           |                 |<----------------+                    |        |
        |         |           |                 |           +--------------+           |        |
        |         |           +-----------------+---------->|cc-oci-runtime| "start"   |        |
        |         |           |                 |           +-----+--------+           |        |
        |         |           |                 |                 |                    |        |
        |         |           |                 |                 | read state         |        |
        |         |           |                 |                 +--------------------|--------+
        |         |           |                 |                 |                    |        |
        |         |           |                 |                 | enable hypervisor  |        |
        |         |           |                 |                 +------------------->|        |
        |         |           |                 |                 |                    |        |
        |         |           |                 |                 | exit()             |        |
        |         |           |<----------------|-----------------+                    |        |
        |         |           |                 |                                      |        |
        |         |           |                 |                                      | exit() |
        |         |           |<----------------+--------------------------------------+        |
        |         |           |                                                                 |
        |         |           |                             +--------------+                    |
        |         |           |-----------------+---------->|cc-oci-runtime| "delete"           |
        |         |           |                             +-----+--------+                    |
        |         |           |                                   |                             |
        |         |           |                                   | delete state                |
        |         |           |                                   +-----------------------------+
        |         |           |                                   |
        |         |           |                                   | exit()
        |         |           |<----------------+-----------------+
        |         |           |
        |         |           | notify exit()
        |<--------+-----------+
        |         |           |
        |exit()   |           |
       ---        |           |
                  :           :
                  .           .

Notes:

- As the diagram shows, the runtime is called multiple times, each time
  being passed a different argument (``create``, ``start``,
  ``delete``).This reflects the way the OCI_ specification mandates the
  runtime be invoked.

- ``containerd-shim`` is able to detect when the ``qemu-lite`` process
  exits since it registers itself as a "sub-reaper" (or "sub-init") process. 

.. _OCI: https://www.opencontainers.org/
.. _Doxygen: www.doxygen.org/
.. _`OCI runtime commands`: https://github.com/opencontainers/runtime-spec/blob/master/runtime.md
.. _`OCI runtime specification`: `OCI runtime commands`_
.. _`OCI config root object`: https://github.com/opencontainers/runtime-spec/blob/master/config.md#root-configuration
.. _Docker: https://github.com/docker/docker
.. _runc: https://github.com/opencontainers/runc
.. _GLib: https://developer.gnome.org/glib/stable
.. _JSON-GLib: https://developer.gnome.org/json-glib/stable
.. _containerd: https://github.com/docker/containerd
.. _`src/commands/`: https://github.com/01org/cc-oci-runtime/blob/master/src/commands/
.. _`src/commands/version.c`: https://github.com/01org/cc-oci-runtime/blob/master/src/commands/version.c
.. _`src/oci.c`: https://github.com/01org/cc-oci-runtime/blob/master/src/oci.c
.. _`src/spec_handlers/`: https://github.com/01org/cc-oci-runtime/blob/master/src/spec_handlers/
.. _`src/spec_handlers/root.c`: https://github.com/01org/cc-oci-runtime/blob/master/src/spec_handlers/root.c
.. _`tests/functional/version.bats`: https://github.com/01org/cc-oci-runtime/blob/master/tests/functional/version.bats
.. _`Linux kernel`: https://www.kernel.org/
.. _BATS: https://github.com/sstephenson/bats
.. _`Go Code Review`: https://github.com/golang/go/wiki/CodeReviewComments
