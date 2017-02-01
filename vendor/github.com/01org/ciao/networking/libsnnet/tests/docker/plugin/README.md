This is simple standalone Docker Plugin implementation designed to test against
changes made in the docker network and IPAM plugin framework.

It is designed to be run standalone so that any changes made in the docker
plugin framework can be tested against the evolution of the docker networking

In ciao the plugin acts as a slave to the ciao networking framework.

The goal here is to do no work in the plugin except inform the docker
daemon about the veth interface it needs to place inside the container

Hence the real flow will be as follows

0. Launcher starts the docker http server plugin thread

Note: The launcher should be launched prior to the docker daemon.
      Also we need to configure docker daemon to not create its default
	  bridge and host networks as they cause problems.

1. Launcher gets a request to launch a container
   The request from the Controller to launcher already has the following
   information (IP Address, MAC Address and subnet for the VNIC)
   Note: Based on the current ciao design the gateway for the
   subnet can be inferred.

2. Launcher invokes ciao networking to create a Container Vnic

3. ciao Networking
     a. Creates a veth pair
     b. Assigns the macaddress to the container side of veth pair
     c. Attaches the veth to the tenant bridge (creating it, if needed)
     d. Returns the fully configured docker side veth pair to Launcher
     e. Also notified launcher if the subnet needs to be created
        (Note: This is the docker logical subnet)

4. (Launcher) if a subnet creation request was returned. Uses docker API
   or command line to instantiate the network in the docker database

  docker network create -d=ciao
			--ipam-driver=ciao
			--subnet=<subnet.IPnet>
			--gateway=<gatewayIP>
			--opt "bridge"=<subnet.BridgeName>
			subnet.Name

	Note: Our custom IPAM driver is needed to support overlapping subnets
	between tenants. Otherwise the default IPAM driver meets our needs.

	Note: Fully specify the network creation and handing control to the
	ciao driver (-d) makes docker a pass-through for networking.

	In the future any additional information need by the plugin can also been
	sent as more options. e.g.
			--opt "cnci"=<subnet.cnciIP>

	- This in turn will result in a callback to the plugin.

	- The plugin will record this information and return success

5. (Launcher) will then request docker to create & launch the container,
   again fully specifying the networking configuration.

    docker run -it --net=<subnet.Name> --ip=<instance.IP> --mac-address=<instance.MacAddresss> busybox

	WARNING: There is a bug in the latest docker 1.10.03 (which has been fixed
	in the 1.11 dev version) which does not pass the --ip parameter to the
	remote IPAM plugin. Without this we cannot use our IPAM driver

6. The ciao docker plugin acts as both a network and IPAM remote plugin.
   It handles all the requests. Some of the more important ones are

	 a. EndPointCreate: If the container is being created for the first time
        As we have already created the VNIC, we only need to cache the endpoint
		id to instance map
	 b. Join: When the end point is being placed inside the container
	    On Join the plugin will return back to docker the following information
           - name of the veth pair to place within the container
	       - the ethernet device name prefix to be assigned to the logic
		     interface within the container (e.g. eth or eno)
		   - the default gw for the container
		   - any other static routes to be added within the container (if needed)

	  Note: We will delete only when the launcher tells us to tear down networking.
		    Not when docker logically tears down the network.

7. The docker daemon will use the values sent back by the plugin to launch the container
   Move the veth into the docker container and give it the logical name.
   Setup the IP address and gateway

