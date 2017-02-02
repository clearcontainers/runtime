[![Build Status](https://travis-ci.org/containers/virtcontainers.svg?branch=master)](https://travis-ci.org/containers/virtcontainers)
[![Go Report Card](https://goreportcard.com/badge/github.com/containers/virtcontainers)](https://goreportcard.com/report/github.com/containers/virtcontainers)
[![Coverage Status](https://coveralls.io/repos/github/containers/virtcontainers/badge.svg?branch=master)](https://coveralls.io/github/containers/virtcontainers?branch=master)
[![GoDoc](https://godoc.org/github.com/containers/virtcontainers?status.svg)](https://godoc.org/github.com/containers/virtcontainers)

# virtcontainers

`virtcontainers` is a Go library that can be used to build hardware-virtualized container
runtimes.

## Background

The few existing VM-based container runtimes (Clear Containers, runv, rkt's
kvm stage 1) all share the same hardware virtualization semantics but use different
code bases to implement them. `virtcontainers`'s goal is to factorize this code into
a common Go library.

Ideally, VM-based container runtime implementations would become translation
layers from the runtime specification they implement (e.g. the [OCI runtime-spec][oci]
or the [Kubernetes CRI][cri]) to the `virtcontainers` API.

[oci]: https://github.com/opencontainers/runtime-spec
[cri]: https://github.com/kubernetes/kubernetes/blob/master/docs/proposals/container-runtime-interface-v1.md

## Out of scope

Implementing a container runtime tool is out of scope for this project. Any
tools or executables in this repository are only provided for demonstration or
testing purposes.

### virtcontainers and CRI

`virtcontainers`'s API is loosely inspired by the Kubernetes [CRI][cri] because
we believe it provides the right level of abstractions for containerized pods.
However, despite the API similarities between the two projects, the goal of
`virtcontainers` is _not_ to build a CRI implementation, but instead to provide a
generic, runtime-specification agnostic, hardware-virtualized containers
library that other projects could leverage to implement CRI themselves.

## Design

### Pods

The `virtcontainers` execution unit is a _pod_, i.e. `virtcontainers` users start pods where
containers will be running.

`virtcontainers` creates a pod by starting a virtual machine and setting the pod
up within that environment. Starting a pod means launching all containers with
the VM pod runtime environment.

### Hypervisors

The `virtcontainers` package relies on hypervisors to start and stop virtual machine where
pods will be running. An hypervisor is defined by an Hypervisor interface implementation,
and the default implementation is the QEMU one.

### Agents

During the lifecycle of a container, the runtime running on the host needs to interact with
the virtual machine guest OS in order to start new commands to be executed as part of a given
container workload, set new networking routes or interfaces, fetch a container standard or
error output, and so on.
There are many existing and potential solutions to resolve that problem and `virtcontainers` abstracts
this through the Agent interface.

## API

The high level `virtcontainers` API is the following one:

### Pod API

* `CreatePod(podConfig PodConfig)` creates a Pod.
The Pod is prepared and will run into a virtual machine. It is not started, i.e. the VM is not running after `CreatePod()` is called.

* `DeletePod(podID string)` deletes a Pod.
The function will fail if the Pod is running. In that case `StopPod()` needs to be called first.

* `StartPod(podID string)` starts an already created Pod.

* `StopPod(podID string)` stops an already running Pod.

* `ListPod()` lists all running Pods on the host.

* `EnterPod(cmd Cmd)` enters a Pod root filesystem and runs a given command.

* `PodStatus(podID string)` returns a detailed Pod status.

### Container API

* `CreateContainer(podID string, container ContainerConfig)` creates a Container on a given Pod.

* `DeleteContainer(podID, containerID string)` deletes a Container from a Pod. If the container is running it needs to be stopped first.

* `StartContainer(podID, containerID string)` starts an already created container.

* `StopContainer(podID, containerID string)` stops an already running container.

* `EnterContainer(podID, containerID string, cmd Cmd)` enters an already running container and runs a given command.

* `ContainerStatus(podID, containerID string)` returns a detailed container status.


An example tool using the `virtcontainers` API is provided in the `hack/virtc` package.
