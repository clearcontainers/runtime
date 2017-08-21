# Ciao Single Machine Development and Test Environment

Developing cluster software is complicated if you must actually run a whole 
cluster on a set of physical machines.  This begs for a development environment
that is self contained and can be run without any setup.  

The goals for the Ciao development environment are that it:

- Requires very minimal setup by the user
- Does not affect the user's development system in any manner (i.e. the user
  can keep the firewall rules, selinux setup,... intact)
- Supports modes that allow it to run on a range of devices from powerful
  workstations to less powerful laptops
- Provides the ability to validate all code changes the user makes against
  the Ciao release criterion

This page documents a way to set up an entire Ciao cluster inside a single 
machine.  This cluster-in-a-machine mode is ideal for developers that desire 
the ability to build Ciao from sources, make changes and perform quick end to 
end functional integration testing without requiring multiple machines/VM's, 
creating a custom networking environment or maintaining a bevy of physical 
machines and a physical network.

We support two modes of operation:

- ciao-down mode: Where a virtual machine is automatically created and 
  launched, and the virtual cluster is setup and tested within the virtual 
  machine
- bare metal mode: Where the virtual cluster is setup on the host machine 
  itself

The ciao-down mode is the preferred mode of development on systems that have 
the resources and CPU capabilities needed, as it fully isolates the Ciao 
virtual cluster and sets up an environment in which Ciao is known to work 
seamlessly. In addition, the ciao-down mode does not require any changes to 
the user's network firewall setup. However, ciao-down mode does require 
VT-x nesting to be supported by the host.

The bare metal mode is the highest performance mode, but may require some 
network firewall modification. It also uses less resources and can run on 
machines whose CPUs do not support VT-x nesting.

In both modes Ciao is configured in a special all in one development mode 
where cluster nodes have dual roles (i.e launcher can be a Network Node and 
a Compute Node at the same time)

In the text below **machine** refers to the ciao-down VM in the case of the 
ciao-down mode, it refers to the host system in the case of the bare metal mode.

## Components running on the Machine
      1. Controller 	
      2. Scheduler 	
      3. Compute+Network Node Agent (i.e. CN + NN Launcher)
      4. Workloads (Containers and VMs)
      5. WebUI
      6. Mock Openstack Services
      7. Machine Local DHCP Server
      ...

The machine acts as the Ciao compute node, network node, ciao-controller, 
ciao-scheduler and also hosts the ciao-webui as well as other openstack 
and dhcp services.

## Graphical Overview

When the system is functioning the overall setup manifests as follows:

As you can see below the Cluster runs on a isolated virtual network resident 
inside the machine. Hence the cluster is invisible outside the machine and
completely self contained.

```
   
   ____________________________________________________________________________________________ 
  |                                                                                            |
  |                                                                                            |
  |                                                                                            |
  |                                                                  [Tenant VMs]  [CNCI VMs]  |
  |                                                                     |  |  |       ||       |
  |                                                     Tenant Bridges ----------     ||       |
  |                                                                         |         ||       |
  |                                                                         |         ||       |
  | [ciao-webui] [scheduler] [controller] [keystone] [CN+NN Launcher]       |         ||       |
  |    ||             ||       ||          ||           ||                  |         ||       |
  |    ||             ||       ||          ||           ||                  |         ||       |
  |    ||             ||       ||          ||           ||                  |         ||       |
  |    ||             ||       ||          ||           ||                  |         ||       |
  |    ||             ||       ||          ||           ||                  |         ||       |
  |    ||             ||       ||          ||           ||      [DHCP/DNS   |         ||       |
  |    ||             ||       ||          ||           ||        Server]   |         ||       |
  |    ||             ||       ||          ||           ||           ||     |         ||       |
  |  --------------------------------------------------------------------------------------    |              
  |           Host Local Network Bridge + macvlan (ciao_br, ciaovlan)                          |
  |                                                                                            |
  |                                                                                            |
  |____________________________________________________________________________________________|
                                                                                                       
                                Development Machine

```

----

# Install Go
On the host install the latest release of go for your distribution
[Installing Go](https://golang.org/doc/install).

> NOTE: Go version 1.8 or later is required for Ciao. Ciao will not work with 
older version of Go. Hence it is best you download and install the latest 
version of Go if you distro is not on Go 1.8.

You should also ensure that your GOPATH environment variable is set.


# Getting Started with ciao-down
ciao-down is a small utility for setting up a VM that contains
everything you need to run ciao's Single VM. All you need to have
installed on your machine is:

- Go 1.8 or greater

Once Go is installed you simply need to type

```
go get github.com/01org/ciao/testutil/ciao-down
$GOPATH/bin/ciao-down create ciao
```

ciao-down will install some needed dependencies on your local PC such
as qemu and xorriso. It will then download an Ubuntu Cloud Image and
create a VM based on this image. It will boot the VM and install in that
VM everything you need to run ciao Single VM, including docker, ceph,
go, gcc, etc. When ciao-down create has finished you can connect to the
newly created VM with

```
$GOPATH/bin/ciao-down connect
```

Your host's GOPATH is mounted inside the VM. Thus you can edit your
the ciao code on your host machine and test in Single VM.


## Proxies

One of the nice things about using ciao-down is that it is proxy aware.
When you run ciao-down create, ciao-down looks in its environment for
proxy variables such as http_proxy, https_proxy and no_proxy.  If it
finds them it ensures that these proxies are correctly configured for
all the software that it installs and uses inside the VM, e.g., apt, docker,
npm, wget, ciao-cli.  So if your development machine is sitting
behind a proxy, ensure you have your proxy environment variables set
before running ciao-down.

## Ciao-webui

Ciao has a webui called ciao-webui (https://github.com/01org/ciao-webui).
When ciao-down create is run, ciao-down downloads the source code for
the ciao-webui and all the development tools needed to build it.  By
default, ciao-down stores the code for the web-ui in ~/ciao-webui.

If you wish to actively work on the sources of ciao-webui you can ask
ciao-down create to mount a host directory containing the webui sources
into the VM.  This is done using the -ui-path option of ciao-down create.
For example

```
ciao-down create --ui-path $HOME/src/ciao-webui ciao
```

would mount the $HOME/src/ciao-webui directory inside the VM at the
same location as the directory appears on your host.  You can then
modify the ciao-webui code on your host and build inside the VM.


For more details and full set of capabilities of ciao-down see the full 
[ciao-down documentation ](https://github.com/01org/ciao/blob/master/testutil/ciao-down/README.md) 


# Getting Started with Bare Metal

## Install Docker
Install latest docker for your distribution based on the instructions from 
Docker
[Installing Docker](https://docs.docker.com/engine/installation/).

## Install ciao dependencies

Install the following packages which are required:
  1. qemu-system-x86_64 and qemu-img, to launch the VMs and create qcow images
  2. gcc, required to build some of the ciao dependencies
  3. dnsmasq, required to setup a test DHCP server

On clearlinux all of these dependencies can be satisfied by installing the following bundles:
```
swupd bundle-add cloud-control go-basic os-core-dev kvm-host os-installer
```
## Setup password less sudo

Setup passwordless sudo for the user who will be running the script below.

## Cluster External Network Access

If you desire to provide external network connectivity to the workloads then 
the host needs to act as gateway to the Internet. The host needs to enable 
ipv4 forwarding and ensure all traffic exiting the cluster via the host is 
NATed.

This assumes the host has a single network interface. For multi homed systems, 
the setup is more complicated and needs appropriate routing setup which is 
outside the scope of this document. If you have a custom firewall 
configuration, you will need set things up appropriately.

Very simplistically this can be done by
```
#$device is the network interface on the host
iptables -t nat -A POSTROUTING -o $device -j MASQUERADE 


echo 1 > /proc/sys/net/ipv4/ip_forward
```


## Download and build the sources

Download and build the ciao sources: 
```
cd $GOPATH/src
go get -v -u -tags debug github.com/01org/ciao/...
```

You should see no errors.

# Verify that Ciao is fully functional  using the **machine**

Now that you have the machine setup (either a bare metal setup or a 
ciao-down VM setup).

You can now quickly verify that all aspects of Ciao including VM launch, 
container launch, and networking.  

These steps are performed inside the machine.

To do this simply run the following:
```
cd $GOPATH/src/github.com/01org/ciao/testutil/singlevm
. ~/local/demo.sh
#Cleanup any previous setup
./cleanup.sh
#Set up the test environment
./setup.sh
. ~/local/demo.sh
#Perform a full cluster test
./verify.sh
```

The ```verify.sh``` script will:
- Create multiple Instances of Tenant VMs and Containers
- Test network connectivity between containers
- Test for ssh reach ability into VMs with private and external IPs
- Delete all the VM's and Container that were created

If the script reports success, it indicates to the developer that any changes 
made have not broken any functionality across all the Ciao components.

To quickly test any changes you make run verify.sh and observe no failures. 

Prior to sumitting a change request to ciao, please run the BAT tests below
in addition to verify.sh to ensure your changes meet the ciao acceptance
criterion.

Meeting the goal originally outlined at the top of the page, build/setup/running 
your cluster all-in-one all transpires quickly and easily from the single 
script.  The time needed for ./setup.sh and ./verify.sh to build ciao from 
source, configure it components into a virtual cluster, then launch and 
teardown containers and VMs is on the order of one minute total elapsed time.

# Ongoing Usage

Once it's finished, the ```setup.sh``` script leaves behind a virtual cluster 
which can be used to perform manual tests.  These tests are performed using 
the [ciao-cli](https://github.com/01org/ciao/blob/master/ciao-cli/README.md) tool.  

The ciao-cli tool requires that some environment variables be set up before it 
will work properly.  These variables contain the URLs of the various ciao 
services and the credentials needed to access these services.  The setup.sh 
script creates a shell source that contains valid values for the newly set up 
cluster.  To initialise these variables you just need to source that file, e.g,

```
. ~/local/demo.sh
```

To check everything is working try the following command

```
ciao-cli workload list
```

# Running ciao-webui

The easiest way to develop and run the ciao-webui is inside a VM built by 
ciao-down.  Not only does ciao-down download the web-ui code for you but it 
also downloads and configures all of the dependencies needed to develop the 
ciao-webui, such as npm.

To run the webui in a ciao-down VM simply execute the following commands

```
cd ~/ciao-webui
./deploy.sh production --config_file=$HOME/local/webui_config.json
```

And then point your host's browser at https://localhost:3000.  The
certificate used by the web-ui inside ciao-down is self signed so you
will need to accept the certificate in your browser before you can view
the web-ui.

Ciao-down configures an account for you to use.  The user name is 'csr'
and the password is 'hello'.  Enter these credentials in the login screen
to start managing your cluster.

# Running the BAT tests

The ciao project includes a set of acceptance tests that must pass before each 
release is made.  The tests perform various tasks such as listing workloads, 
creating and deleting instances, etc.  These tests can be run inside the 
machine

```
# Source the demo.sh file if you have not already done so
. ~/local/demo.sh
cd $GOPATH/src/github.com/01org/ciao/_release/bat
test-cases -v ./...
```

For more information on the BAT tests please see the [README](https://github.com/01org/ciao/blob/master/_release/bat/README.md).

# Cleanup / Teardown

To cleanup and tear down the cluster:
```
cd $GOPATH/src/github.com/01org/ciao/testutil/singlevm
#Cleanup any previous setup
. ~/local/demo.sh
./cleanup.sh
```

# Known Issues with Bare Metal

- Does not work on Fedora due to default firewall rules. 
https://github.com/01org/ciao/issues/526

In order to allow the traffic required by the test cases you can add temporary 
rules like the ones show below

```
#!/bin/bash
iptables -I INPUT   1 -p tcp -m tcp --dport 8888 -j ACCEPT
iptables -I INPUT   1 -p 47 -j ACCEPT
iptables -I OUTPUT  1 -p 47 -j ACCEPT
iptables -I INPUT   1 -p tcp --dport 22 -m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT
iptables -I OUTPUT  1 -p tcp --sport 22 -m conntrack --ctstate ESTABLISHED -j ACCEPT
iptables -I FORWARD 1 -p tcp --dport 22 -m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT
iptables -I FORWARD 1 -p tcp --sport 22 -m conntrack --ctstate ESTABLISHED -j ACCEPT
iptables -I FORWARD 1 -p udp -m udp --dport 67:68 -j ACCEPT
iptables -I FORWARD 1 -p udp -m udp --dport 123 -j ACCEPT
iptables -I FORWARD 1 -p udp -m udp --dport 53 -j ACCEPT
iptables -I FORWARD 1 -p udp -m udp --dport 5355 -j ACCEPT
iptables -I FORWARD 1 -p icmp -j ACCEPT
```

And delete them after the tests using
```
#!/bin/bash
iptables -D INPUT   -p tcp -m tcp --dport 8888 -j ACCEPT
iptables -D INPUT   -p 47 -j ACCEPT
iptables -D OUTPUT  -p 47 -j ACCEPT
iptables -D INPUT   -p tcp --dport 22 -m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT
iptables -D OUTPUT  -p tcp --sport 22 -m conntrack --ctstate ESTABLISHED -j ACCEPT
iptables -D FORWARD -p tcp --dport 22 -m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT
iptables -D FORWARD -p tcp --sport 22 -m conntrack --ctstate ESTABLISHED -j ACCEPT
iptables -D FORWARD -p udp -m udp --dport 67:68 -j ACCEPT
iptables -D FORWARD -p udp -m udp --dport 123 -j ACCEPT
iptables -D FORWARD -p udp -m udp --dport 53 -j ACCEPT
iptables -D FORWARD -p udp -m udp --dport 5355 -j ACCEPT
iptables -D FORWARD -p icmp -j ACCEPT
```

