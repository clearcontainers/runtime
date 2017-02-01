#CNCI Command Line Interface

##Overview

The cncicli is a simple command line interface to test the APIs exercised by
the CNCI agent when running within the CNCI VM.

Invoking the cncicli results in a CNCI Agent instance serving a single tenant
subnet on the host machine. This can be used to test the inter-working of a
CN VNIC created using the cncli and the cnci-agent.

## Usage

```

Usage of ./cncicli:
  -alsologtostderr
        log to standard error as well as files
  -cnciSubnet string
        CNCI Physicla subnet on which the CN can be reached
  -cnciuuid string
        CNCI UUID (default "cnciuuid")
  -cnip string
        CNCI reachable CN IP address (default "127.0.0.1")
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -operation string
        operation <create|delete|reset> reset clears all CNCI setup (default "create")
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -tenantSubnet string
        Tenant subnet served by this CNCI (default "192.168.8.0/21")
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging

```

## Example

To run a CNCI Agent which is setup on a machine which has a physical interface
192.168.0.100/24 setup to serve a tenant subnet of 172.1.1.0/24 with
VM's connecting to the CNCI from a remote CN with IP 192.168.0.103/24

```
go build
sudo ./cncicli -cnciSubnet 192.168.0.0/24 -cnip 192.168.0.103 -tenantSubnet 172.1.1.0/24
```

Now you can connect to this CNCI (running on the bare metal host) from a VM
running on 192.168.0.103 created using the command

```
sudo cncli -cnci 192.168.0.100 -subnet 192.168.0.0/24 -vnicsubnet 172.1.1.0/24 -vnicIP 172.1.1.24 -operation create
```
