# CNCI Agent #

## Overview ##

Compute Node Concentrators or CNCIs are Virtual Machines running on
Network Nodes which handle subsets of traffic belonging to a single tenant.
A single network node can run multiple CNCI's limited by the Compute and 
Network needs of the CNCIs. All tenant level switching and routing for 
a given tenant is handled (isolated) from other tenants using the CNCIs.

The CNCIs also implement tenant specific firewall and NAT rules. In the future
they may be extended to perform traffic shaping.

## CNCI Agent ##

The CNCI Agent is the service running within a CNCI VM that communicates with 
the ciao-scheduler to create new bridges and tunnels in response to remote 
bridge and tunnel creation on a compute node.

## CNCI Agent Lifecyle ##

### CNCI Provisioning ###

A CNCI VM is provisioned by the ciao-controller to handle and isolate traffic
for subnets that belong to a specific tenant.

### CNCI Registration ###

When the CNCI VM boots up (at each boot up or restart) the CNCI Agent
notifies the ciao-scheduler using SSNTP that it is active and handles
tenant subnets for a specific tenant. The scheduler in turn notifies the 
controller of the CNCI IP address.

The ciao-controller associates this IP address with the appropriate 
tenant (subnets).

At this point the CNCI is registered with the ciao-controller to handle 
a specific set of tenant subnets

### Compute Node Subnet Creation and Registration ###

When the ciao-controller requests scheduling of a tenant workload it also
sends the associated CNCI IP address that handles the tenant traffic for this
workload as part of the payload definition to the cia-launcher.

The launcher creates the VNIC (Virtual Network Interface) on a compute 
node in response to a workload being launched by the ciao-launcher

When the VNIC is instantiated the networking library checks if it is the 
first (only) instance of that tenant subnet on that CN at that point in time.

If it is the first instance it creates a local bridge and a tunnel it to the 
CNCI associated with that workload.

It also requests the launcher to notify the CNCI (via the ciao-scheduler) about
the creation of this remote subnet.

The Launcher sends this request to the ciao-scheduler which sends it to the 
CNCI all via SSNTP.

### CNCI Subnet Creation ###

When the CNCI sees a Remote Subnet Registration message it links the remote
subnet to the appropriate subnet bridge on the CNCI. 

The CNCI agent manages the bridges, routing, NAT and traffic for all tenant
IPs and subnets it handles.

