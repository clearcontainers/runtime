# Ciao Networking

## Overview

The goal of the Ciao Networking framework is to provide each tenant with a
secure isolated overlay network without the need for any configuration by the
tenant and minimal configuration by the data center operator.

The only configuration that is required (by the data center operator) is to
assign roles to each physical node. Subsequent discovery and configuration of
the network is performed automatically using the
[ciao-controller](https://github.com/01org/ciao/tree/master/ciao-controller)
and [ciao-launcher](https://github.com/01org/ciao/tree/master/ciao-launcher)

Compute Nodes and Network Nodes can be added dynamically and will be auto
discovered and utilized without any additional network configuration.  Each
physical node in Ciao is assigned one of the following roles by the
ciao-launcher when the node is booted.

### Compute Node (CN)

A compute node typically runs VM and Container workloads for multiple tenants.

### Network Node (NN)

A Network Node is used to aggregate network traffic for all tenants while still
keeping individual tenant traffic isolated from all other the tenants using
special virtual machines called Compute Node Concentrators (or CNCIs)


### Compute Node Concentrator (CNCI)

CNCIs are Virtual Machines automatically configured by the ciao-controller,
scheduled by the ciao-scheduler on a need basis, when tenant workloads are
created.

## Goals

The primary design goals of Ciao Networking are to provide
- Fully isolated tenant overlay networks
- Auto discovery and configuration of physical nodes
- Auto configuration of basic tenant network properties
- Support large number of tenants with large or small number of workloads
- Operate on any Linux distribution by limiting the number of dependencies on
  user-space tools and leveraging Linux kernel interfaces whenever possible.
- Provide the ability to migrate workloads from a Compute Node on demand or
  when a CN crashes without tenant intervention
- Provide the ability to migrate CNCIs on demand or when a Network Node crashes
- Provide the ability to transparently encrypt all tenant traffic traffic even
  within the data center network
- Support for multiple types of tunneling protocols (GRE, VXLAN, Geneve,...)
- Support for tenant and workload level security rules
- Support for tenant and workload level NAT rules (inbound and outbound)
	- Public IP assignment support for tenant workloads
	- Inbound directed port forwarding (SSH, HTTP, ...)
	- Configurable ability to reach any tenant workload via SSH

## Ciao Networking Functional Components

Ciao Networking consists of the following key components (none of which are
user or tenant visible)
- libsnnet: which provides networking APIs to the ciao-launcher to create
  tenant specific network interfaces on CNs and CNCI specific network
interfaces on a NN
- ciao-cnci-agent: a [SSNTP](https://github.com/01org/ciao/tree/master/ssntp) client
  which connects to the
[ciao-scheduler](https://github.com/01org/ciao/tree/master/ciao-scheduler) and
runs within a CNCI VM and configures tenant network connectivity by interacting
with the ciao-controller and ciao-launchers using the ciao-scheduler.
	- The ciao-cnci-agent can also be run on physical nodes if desired
- docker-plugin: built into the libsnnet which is used by ciao-launcher to
  provide unified networking between VM and Docker workloads

# Ciao Network Topology

The figure below illustrates a lowest level subset of the network topology
where tenant overlay networks have been setup. The ciao components (launcher,
scheduler, controller and cnci-agent) communicate securely using SSNTP. All
tenant traffic is encapsulated using tenant subnet specific tunnels and
terminates in tenant specific CNCIs. The figure also illustrates how the CNCI
creates a tenant bridge per tenant subnet and inter-subnet tenant traffic is
routing within the CNCI.  ![](./documentation/ciao-networking.png "ciao network
topology")
