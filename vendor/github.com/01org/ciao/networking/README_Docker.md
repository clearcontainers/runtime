#Docker Networking Background

Docker over the last few releases has added the ability for third parties to replace parts of the docker engine using plugins.

Currently for networking Docker supports replacement of the built in networking functionality using the following two plugins

- [Network Plugin](https://github.com/docker/libnetwork/blob/master/docs/remote.md "Network Plugin"): Responsible for creation of the infrastructure (bridges, interfaces, veth pairs etc)
  

- [IPAM Plugin](https://github.com/docker/libnetwork/blob/master/docs/ipam.md "IPAM Plugin") (IP Address Management Plugin): Responsible for managing the network subnet, IP address, gateway and static route assignment...
  
The plugin's initially managed containers at a node level (local) vs at a cluster (global) level.

Docker has recently added support for global management of Networking using the [overlay driver plugin](https://github.com/docker/libnetwork/blob/master/docs/overlay.md "overlay network plugin"). However this is work in progress. Docker 1.11 seems to have most of the framework required to integrate with an alternative SDN framework.



#ciao Docker Networking
In ciao we already have a simple SDN controller implementation that manages the following

- Tenant Network Isolation
- Tenant Subnets
- Tenant specific bridges, switching, routing, interconnection and aggregation
- Tenant IP and MAC address management

We already create node local bridges for each tenant subnet and link them to the broader tenant network. We also have the capability to add VM and container compatible interfaces to the tenant bridge. 

- In addition the ciao-launcher also controls the lifecyle of workloads (i.e. how containers, VMs and other workloads are spawned).
- Also the typical user only cares that the workload (container or VM) is launched and that it has network connectivity and other networking properties and restrictions the user desires

We leverage these two facts to create Network and IPAM Plugins that serve as simple pass-thro drivers. This means that they are only needed to map the ciao networking framework and identifiers to the docker generated identifiers. (Basically docker uuids to ciao uuids, bridges and interfaces).

Hence we chose to use the Network and IPAM plugins in pass thro mode vs using the Overlay Plugin. The details of the pass thro mode are 

One of the reasons for doing so is that ciao does not try to rely on centralized state as far as possible. Also a goal of ciao is to treat all workloads VMs, Containers,... as equal with the same degree of connectivity and controls, managed via a unified scheduler.

#How does it all come together

##Registration
ciao Networking Agent (i.e. ciao networking library within launcher) registers as a Plugin for Docker using a JSON spec file

###Docker Plugin Configuration File

    /etc/docker/plugins/ciao.json
    {
     "Name": "ciao",
     "Addr": "http://127.0.0.1:9999"
    }
    
Note: This will be secured using TLS for additional security


##Activation
When Docker Engine starts it queries the agent's capabilities using HTTP REST APIs

    /Plugin.Activate

The Networking Agent reports that it is both a Network Driver and a IPAM Driver


    resp := `{ "Implements": ["NetworkDriver", "IpamDriver"] }`

The Networking Agent reports that it is both a Network Driver and a IPAM Driver

Docker then queries for additional capabilities using

    /NetworkDriver.GetCapabilities
    /IpamDriver.GetCapabilities
    
Agent Responds with


    {Scope: "local"}

indicating that it is not a overlay driver.

NOTE: This was done intentionally as mentioned before.

##Subnet Management

A logical docker visible subnet has to be created before a container can be attached to a subnet. Even though ciao is creating and managing its own subnets, prior to launching a container on node a logical subnet has to be created on the node. This is done through the following sequence

When the first container belonging to a particular tenant subnet is launched on a given node, the network subnet is logically created using the docker API. The equivalent CLI based sequence is explained below


    docker network create --driver=ciao --ipam-driver=ciao --subnet=<a.b.c.d/s> --gateway=<a.b.c.x> --opt "bridge"="<ciao bridge linkname>"   <ciao bridge linkname>
    
example

    docker network create --driver=ciao --ipam-driver=ciao --subnet=192.168.111.0/24 --gateway=192.168.111.1 --opt "bridge"="br1" br1
    
Some things to note
- We are fully specifying the subnet we want to create, its default gateway and on what ciao bridge this subnet already accessible on this node.
- We are also specifying that for this subnet we want to use the ciao network plugin as the network driver and the IPAM driver.
- By specifying the networking information beforehand, Docker also informs the IPAM Network Plugin (informing it vs asking it to allocate a pool)
- We are sending in ciao specific networking parameters using the optional parameters which Docker relays to the plugin


Some of the key interactions between Docker and the plugins's are shown below based on the CLI example specified 

###Network Address Pool Management
    [IpamDriver.RequestPool] Body [{"AddressSpace":"","Pool":"192.168.111.0/24","SubPool":"","Options":{},"V6":false}

Here Docker specifies to the IPAM plugin the desired network pool to be assigned to a given subnet (vs requesting a pool for the subnet that is being created.

One key thing to note here is that this subnet is not really local. The ciao overlay network framework ensures that the subnet can be distributed across multiple compute nodes. Hence docker has visibility into the local state of the subnet.
    
Docker also informs the IPAM Plugin of the gateway IP address

    [IpamDriver.RequestAddress] Body [{"PoolID":"8e2fa5ef-adcd-4c7f-8d07-74e4a95cde06","Address":"192.168.111.1","Options":{"RequestAddressType":"com.docker.network.gateway"}}

Hence the ciao IPAM plugin is just used to map the ciao network properties to the docker logical network.    

###Network Management

Docker requests the Network Plugin to create a network with the properties it obtained from the IPAM driver.


    [NetworkDriver.CreateNetwork] Body [{"NetworkID":"2f234f730b2791f8230bef63086e4a5deb895b53ca286580bcb943ddc729c230","Options":{"com.docker.netw|ork.enable_ipv6":false,"com.docker.network.generic":{"bridge":"br1"}},"IPv4Data":[{"AddressSpace":"","Gateway":"192.168.111.1/24","Pool":"192.168.111.0/24"}],"IPv6Data":[]}

The Network Plugin uses the information (specifically the bridge) to map the docker logical network to the ciao network framework.


**From the steps above, you will notice that the plugins are controlled by the ciao-launcher through the docker API (hence we call them pass-through drivers)**

At this point the plugin records the docker logical network id (UUID) and what ciao bridge maps to (in the example here br1 <> 2f234f730b2791f8230bef63086e4a5deb895b53ca286580bcb943ddc729c230)


###Container Launch and IP Address Assignment

Prior to the launch of a container the ciao-launcher creates the ciao specific interface that should be used by the workload to attach to the specific tenant network.

When we need to launch a container the ciao-launcher again fully specifies the network, IP address and MAC address to be used which is illustrated using the CLI as follows

    docker run -it --net=<subnet name>  --ip=<a.b.c.d> --mac-address=<aa:bb:cc:dd:ee:ff> <dockerimage>

example:


    docker run -it --net=br1 --ip=192.168.111.2 --mac-address=CA:FE:00:00:10:02 busybox

Docker sends this information to the network plugin. The network and IPAM plugin map this information to the interface that has already been created for this tenant by the launcher and sends its name back to docker. 

Docker then attaches the interface to the container, moves it into the container namespace, programs its MAC and IP address, gateway and default and static routes

The typical sequence of operations is shown below
    
    [IpamDriver.RequestAddress] Body [{"PoolID":"82fa5ef-adcd-4c7f-8d07-74e4a95cde06","Address":"192.168.111.2","Options":{"com.docker.network.endpoint.macaddress":"ca:fe:00:00:10:02"}}

    [NetworkDriver.CreateEndpoint] Body [{"NetworkID":"2f234f730b2791f8230bef63086e4a5deb895b53ca286580bcb943ddc729c230","EndpointID":"2688ccdb101bb5f3b199880041162ae3ec1b299053d703e07d61330292c380d3","Interface":{"Address":"192.168.111.2/24","AddressIPv6":"","MacAddress":"ca:fe:00:00:10:02"},"Options":{"com.docker.network.endpoint.exposedports":[],"com.docker.network.endpoint.macaddress":"yv4AABAC","com.docker.network.portmap":[]}}

    [NetworkDriver.Join] Body [{"NetworkID":"2f234f730b2791f8230bef63086e4a5deb895b53ca286580bcb943ddc729c230","EndpointID":"2688ccdb101bb5f3b199880041162ae3ec1b299053d703e07d61330292c380d3","SandboxKey":"/var/run/docker/netns/837d34a2ff2b","Options":{"com.docker.network.endpoint.exposedports":[],"com.docker.network.portmap":[]}}
    
    [NetworkDriver.ProgramExternalConnectivity] Body [{"NetworkID":"2f234f730b2791f8230bef63086e4a5deb895b53ca286580bcb943ddc729c230","EndpointID":"2688ccdb101bb5f3b199880041162ae3ec1b299053d703e07d61330292c380d3","Options":{"com.docker.networ|k.endpoint.exposedports":[],"com.docker.network.portmap":[]}}
    
    [NetworkDriver.EndpointOperInfo] Body [{"NetworkID":"2f234f730b2791f8230bef63086e4a5deb895b53ca286580bcb943ddc729c230","EndpointID":"2688ccdb101bb5f3b199880041162ae3ec1b299053d703e07d61330292c380d3"}
    

With this approach all tenant workloads (VM, Containers, ...) have equal network connectivity and can be managed uniformly from a network point of view.
