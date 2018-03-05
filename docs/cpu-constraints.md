# CPU Constraints in Clear Containers

## Default number of virtual CPUs

Before starting a container, the runtime reads the option `default_vcpus` from the configuration file to determine the number of virtual CPUs (vCPUs) needed to start the virtual machine. By default, `default_vcpus` is equal to 1 for fast boot time and a small memory footprint per virtual machine. Be aware that increasing this value negatively impacts the virtual machine's boot time and memory footprint.
In general, we recommend that you do not edit this variable, unless you know what are you doing. If your container needs more than one vCPU, use [docker `--cpus`][1], [docker update][4] or [kubernetes `cpu` limits][2] to assign more resources.

## Virtual CPUs and Kubernetes PODs

A kubernetes POD is a group of one or more containers, with shared storage and network, and a specification for how to run the containers [[1][3]]. In Clear Containers this group of containers runs inside the same virtual machine. If you do not specify a CPU constraint, the runtime does not hot add more vCPUs and the container is not placed inside a CPU cgroup.  Instead, the container uses the number of vCPUs specified by `default_vcpus` and shares these resources with other containers in the same situation (without a CPU constraint).

## Containers life cycle

When you create a container with a CPU constraint, the runtime hot adds the number of vCPUs required by the container. Similarly, when the container stops, the runtime hot removes these resources.

## Containers without CPU constraint

A container without a CPU constraint uses the default number of vCPUs specified in the configuration file. In the case of Kubernetes PODs, containers without a CPU constraint use and share between them the default number vCPUs. For example, if default_vcpus is equal to 1 and you have 2 containers without CPU constraints with each container trying to consume 100% of vCPU, the resources divide in two parts, 50% of vCPU for each container because your virtual machine does not have enough resources to satisfy containers needs. If you wish to give access to a greater or lesser portion of vCPUs to a specific container, you can use [docker --cpu-shares][1] or [kubernetes `cpu` requests][2].
Before running containers without CPU constraint, consider that your containers are not running alone. Since your containers run inside a virtual machine other processes try to use the vCPUs as well (e.g. `systemd` and the Clear Container [agent][5]). In general, we recommend setting `default_vcpus` equal to 1 to allow non-container processes to run on this vCPU and to specify a CPU constraint for each container. If your container is already running and needs more vCPUs, you can add more using [docker update][4].

## Containers with CPU constraint

The runtime calculates the number of vCPUs required by containers with CPU constraint using the following formula: `(quota + (period -1)) / period`. The result determines the number of vCPU to hot plug into the virtual machine. Once the vCPUs have been hot added, the [agent][5] places the container inside a CPU cgroup. This placement allows the container to use only its assigned resources.

## Do not waste resources

If you already know the number of vCPUs needed for each container and POD, or just want to run them with the same number of vCPUs, you can specify that number using the `default_vcpus` option in the configuration file, each virtual machine starts with that number of vCPUs. One limitation of this approach is that these vCPUs cannot be hot removed later and you might be wasting resources. For example, if you set `default_vcpus` to 8 and run only one container with a CPU constrain of 1 vCPUs, you might be wasting 7 vCPUs since the virtual machine starts with 8 vCPUs and 1 vCPUs is hot added and assigned to the container. Non-container processes might be able to use 8 vCPUs but they use a maximum 1 vCPU, hence 7 vCPUs might not be used.

## Limitations

- [docker `--cpuset-cpus`][1], [docker --cpu-shares][1], [docker update][4] and [kubernetes `cpu` requests][2] are not supported.



[1]: https://docs.docker.com/config/containers/resource_constraints/#cpu
[2]: https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource
[3]: https://kubernetes.io/docs/concepts/workloads/pods/pod/
[4]: https://docs.docker.com/engine/reference/commandline/update/
[5]: https://github.com/clearcontainers/agent

