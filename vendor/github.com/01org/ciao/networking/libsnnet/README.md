# Simple Node Network Library #

## Overview ##

The Simple Node Network Library (libsnnet) implements a simple SDN controller.
The library implements all the networking setup primitives required in Ciao.

libsnnet currently provides the following capabilities
- Creation of multiple isolated tenant overlay L2 networks
- Auto assignment of IP Addresses for VM's and containers
- Support for Virtual Machine (QEMU)and Container workloads
    - Containers and VMs that belong to the same tenant are placed in the 
      same L2 overlay network and connect to each other seamlessly
- Ability to perform inbound and outbound NAT to/from the workloads
- A built in docker Network and IPAM plugin to allow interworking between 
  Docker logical networks and the ciao L2 overlay Network

It tries to rely on interfaces directly exposed by the kernel vs using user
space tools to ensure maximum portability. The implementation maintains local 
state on leaf nodes vs relying on centralized state. It also uses local state to
perform any network re-configuration in the event of a launcher crash or restart

Currently the library supports creation of bridges, GRE tunnels, VM and Container
compatible interfaces (VNICs) on nodes. It also provides and the ability to
attach tunnels and VNICs to bridges.

The implementation also provides the ability to interconnect these bridges
across nodes creating L2 Overlay networks.

## Roles ##

The library supports node specific networking initialization capabilities.
It currently supports setup of Compute Nodes (CN), Network Nodes (NN) and
Compute Node Concentrator Instances (CNCI)

### Compute Node ###

A compute node typically runs VM and Container workloads. The library provides
API's to perform network initialization as well as network interface creation
and overlay network linking.

### Network Node ###

The tenant overlay networks are linked together to Network Nodes. The Network
Node switch and route traffic between the tenant bridges and subnets distributed
across multiple Compute Nodes.

### CNCI ###

Compute Node Concentrators or CNCIs are Virtual Machines running on
Network Nodes which handle subsets of traffic belonging to a single tenant.
A single network node can run multiple CNCI's limited by the Compute and
Network needs of the CNCIs. All tenant level switching and routing for
a given tenant is handled isolated from other tenants using the CNCI's.
The CNCIs also implement tenant specific firewall and NAT rules. In the future
they may be extended to perform traffic shaping.

## Testing ##
The libsnnet library exposes API's that are used by the launcher and other
components of ciao. However the library also includes a reasonably comprehensive
unit test framework. The tests can be run as follows

```bash
sudo ip link add testdummy type dummy
sudo ip addr add 198.51.100.1/24 dev testdummy
export SNNET_ENV=198.51.100.0/24
export FWIFINT_ENV=testdummy

sudo ip link add extdummy type dummy
sudo ip addr add 203.0.113.1/24 dev extdummy
export FWIF_ENV=extdummy

sudo -E go test --tags travis -v --short
```

Note: Some of the API's require Docker 1.11+ to be installed on the test system.
Please install Docker to ensure that all the unit tests pass.

These tests also help identify any regressions due to changes in the netlink 
library or the docker Network and IPAM plugin frameworks.
