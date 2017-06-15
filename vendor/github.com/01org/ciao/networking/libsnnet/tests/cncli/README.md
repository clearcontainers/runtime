#Compute Node Command Line Interface

##cncli

The cncli is a simple command line interface that can be used to create
ciao VNICs for Workload VMs and CNCI VMs.

The cli can be invoked with the following options

```
Usage of ./cncli:
  -alsologtostderr
        log to standard error as well as files
  -cnci string
        CNCI IP (default "127.0.0.1")
  -cnciuuid string
        CNCI UUID (default "cnciuuid")
  -cnuuid string
        CN UUID (default "cnuuid")
  -iuuid string
        instance UUID (default "iuuid")
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -mac string
        VNIC MAC Address (default "DE:AD:BE:EF:02:03")
  -nwNode
        true if Network Node
  -operation string
        operation <create|delete> (default "create")
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -subnet string
        subnet of the compute network
  -suuid string
        subnet UUID (default "suuid")
  -tuuid string
        tunnel UUID (default "tuuid")
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
  -vnicIP string
        VNIC IP (default "127.0.0.1")
  -vnicsubnet string
        subnet of vnic network (default "127.0.0.1/24")
  -vuuid string
        VNIC UUID (default "vuuid")
```

A simple use of this cli would be to create a VNIC and attach a VM to it.

This can be achieved as follows

1. Create a workload VNIC

   Here we assume that one of the interfaces on your host system has an IP
   address in the 192.168.1.0/24 subnet.

   ```
   go build
   sudo ./cncli -subnet 192.168.1.0/24 -operation create
   ```
   The output of this is along the lines of

   ```
   Creating VNIC for Workload
   SSNTP Event := &{2 127.0.0.1 192.168.1.15 127.0.0.0/24 tuuid suuid cnciuuid cnuuid 127 }
   tap interface := svn_fd6db780
   ```

  This indicates that a tap interface has been created with the name svn_fd6db780

2. Launch a VM

   Now you can launch a VM that attaches to the VNIC

   ```
  sudo ./run_vm.sh clear.img svn_fd6db780
   ```

3. Delete the VNIC

   ```
   sudo ./cncli -subnet 192.168.1.0/24 -operation delete
   ```
