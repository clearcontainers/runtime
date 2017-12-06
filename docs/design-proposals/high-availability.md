# High Availability with Clear Containers

* [Overview](#overview)
* [Requirements](#requirements)
* [Current situation](#current-situation)
    * [Current Proxy](#current-proxy)
    * [Current Shim](#current-shim)
* [Plan](#plan)
    * [Summary of required component changes](#summary-of-required-component-changes)
        * [Runtime](#runtime)
        * [Proxy](#proxy)
        * [Shim](#shim)
        * [Agent](#agent)
        * [Workload](#workload)
        * [Hypervisor](#hypervisor)
    * [Required test changes](#required-test-changes)
        * [Mock components](#mock-components)
        * [Scenario testing](#scenario-testing)
    * [Scenarios that need testing](#scenarios-that-need-testing)
        * [Disconnects](#disconnects)
        * [`ENOSPC`](#`enospc`)
        * [`ENOMEM`](#`enomem`)
        * [Limits](#limits)
        * [Logging](#logging)

## Overview

This document summarises the current failure behaviour of a Clear Containers
system along with proposals for making it more highly available.

## Requirements

- Ability for the Clear Containers system to be robust against all failure scenarios.

- Ensure no single point of failure.

- Ensure all failure scenarions are reported by the logging mechanisms.

- Ensure sufficient diagnostic details are provided in the event of failure.

## Current situation

A Clear Containers system comprises a number of components:


| Component    | Daemon | Lifespan  | Launcher             | Restartable | Restart mechanism    | Restart logged? | Disconnect recoverable | Disconnect recovery mechanism | Crash recoverable   | Crash recovery mechanism |
|--------------|--------|-----------|----------------------|-------------|----------------------|-----------------|------------------------|-------------------------------|---------------------|--------------------------|
| `cc-agent`   | yes    | pod       | `systemd` (in guest) | yes         | `systemd` (guest)    | no [4]          | no                     | n/a                           | no                  | n/a                      |
| `cc-proxy`   | yes    | system    | `systemd` (on host)  | yes         | `systemd` (host) [3] | yes             | yes ([proxy state PR])   | retry                         | partial ([proxy state PR]) | state files              |
| `cc-runtime` | no     | brief [1] | container manager    | "yes" [2]   | container manager    | yes [5]         | no                     | n/a                           | yes                 | container manager        |
| `cc-shim`    | yes    | container | runtime (vc)         | no          | n/a                  | n/a             | yes ([shim reconnect PR]) | retry                         | no                  | n/a                      |
| hypervisor   | yes    | pod       | runtime (vc)         | "yes" [2]   | container manager    | "yes"           | n/a                    | n/a                           | no                  | n/a                      |
| workload     | maybe  | container | agent                | no          | n/a                  | no              | n/a                    | n/a                           | n/a                 | n/a                      |


Key:

- n/a - "not applicable" (does not apply).

- If a value is double-quoted, it connotect incomplete support (meaning "sometimes" or "partially").

- [1] - The runtime is invoked multiple times for some container manager operations, complying with the OCI runtime specification.

- [2] - Can be re-run by the container manager as necessary, within the bounds of the OCI specification.

- [3] - Once https://github.com/clearcontainers/proxy/pull/153 lands.

- [4] - Currently, if the agent exits, the virtual machine is shut down so no restart is possible.

- [5] - Visible in [global log](https://github.com/clearcontainers/runtime#global-logfile) and container manager logs.

Most of the components are indirectly launched by a container manager
(such as Docker*) which calls the runtime component; if the runtime
detects any errors, these will be reported back to the container manager
which will take appropriate action.

### Current Proxy

The proxy is currently being reworked for HA on the [proxy state PR].

### Current Shim

Thanks to the [shim reconnect PR], the shim now reconnects to the proxy should
the latter die. However, it does not cache signals, input or the last command
run when disconnected from the proxy.


## Plan

### Summary of required component changes

#### Runtime

Make runtime attempt to re-connect to the proxy if it is unable to
connect initially. This requires changes to the `virtcontainers`
`ccProxy.connectProxy()` function to make it:

- Reconnect and timeout after a period (like the [shim reconnect PR]).

- Record details of the last command `virtcontainers` attempted to send to the
  proxy (`hyperstartProxyCmd`) in memory.

- If a proxy reconnect is required, re-send the command.

#### Proxy

No changes.

#### Shim

Make the shim cache all signals, input data and the last command run and
re-send to the proxy when it reconnects after being disconnected from
the proxy.

#### Agent

- Handle gracefully the re-connection of the proxy to continue properly
where we left.

- Buffer all outputs supposed to go through STDOUT/STDERR, so that the
agent can send them after the proxy re-connection. Specific to IO
channel.

- Save the last command that got executed while the proxy was crashing,
and save the result. The idea is that when the proxy is gonna reconnect,
it's gonna send again the last command because it didn't get the result
(this command is really gonna be triggered by the shim or the runtime
when they reconnect). For that reason, the agent should analyze the
command sent by the proxy after it reconnects, and not execute it (in
case that matches the last command), but send the saved result, to avoid
running the same command a second time. That way, the proxy will receive
the result of this command.

- Always save the last outputs that we are sending. This would allow the
agent to resend the result when the same command is submitted from the
shim or runtime, but that we don't want to re-run the command for real,
because it could have different results.

- Modify the agent service to restart on failure once it is able to
  reconnect to the workload and proxy.

#### Workload

#### Hypervisor

Identify how the current behaviour can be improved; if the proxy
currently stops, the hypervisor will be left running consuming a large
amount of CPU due to the agent attempting to reconnect to the proxy.
The reconnect behaviour is correct, but there is no timeout in the case
where the proxy needs to be manually stopped by an administrator for example.

### Required test changes

#### Mock components

Mock components and frameworks will need to be updated to match the new behaviour above.

#### Scenario testing

To ensure every failure scenario is covered, we need to provide a way to
stimulate all types of major failure. This needs further exploration but
we need to consider whether:

- it may be possible provoke some/most scenarios using our mock systems.

- we can use other tooling to for example guarantee no available disk
  space / memory.

- potentially we need to change the actual components to provide a way
  to make them fail "on demand".


### Scenarios that need testing

#### Disconnects

Ability to test all failure scenarios caused by any component being
disconnected from any other.

#### `ENOSPC`

Ensure all components handle a lack of disk space in a sane manner (by
reporting an error back to the caller).

#### `ENOMEM`

Ensure all components handle a lack of memory in a sane manner (by
reporting an error back to the caller).

#### Limits

Test what happens when:

- no more processes can be created.
- no more network connections can be created.
- no more file descriptors can be used.
- no more locks can be created.
- no more files can be created.
- no more inodes can be created.

#### Logging

- Ensure all components log full error details to ensure problem
determination is possible.

- Ensure all restartable components log a message when a restart occurs.

[proxy state PR]: https://github.com/clearcontainers/proxy/pull/107
[shim reconnect PR]: https://github.com/clearcontainers/shim/pull/54
