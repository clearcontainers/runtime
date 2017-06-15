Ciao Scheduler
==============

The ciao scheduler implements an
[SSNTP](https://github.com/01org/ciao/tree/master/ssntp) server to
receive workloads from the ciao controller system(s), to receive status
changes regarding ciao compute node (CN) resources and launched workload
instances, and to reply to nodes who've checked in by giving them work.


Running Scheduler
-----------------

Scheduler does not need to run as root, unlike other ciao components.

Certificates are assumed to be in /etc/pki/ciao, or can be
specified on the command line via the "-cert" and "-cacert"
command line options.  Certificates are created with the
[ciao-cert](https://github.com/01org/ciao/tree/master/ssntp/ciao-cert)
tool.

For debugging or informational purposes glog options are useful.
The "-heartbeat" option emits a simple textual status update of connected
controller(s) and compute node(s).

Of course nothing much interesting happens until you connect at least
a ciao-controller and ciao-launchers also.  See the [ciao cluster setup
guide]() for more information.

### Usage

```shell
Usage of ./ciao-scheduler:
  -alsologtostderr
    	log to standard error as well as files
  -cacert string
    	CA certificate (default "/etc/pki/ciao/CAcert-server-localhost.pem")
  -cert string
    	Server certificate (default "/etc/pki/ciao/cert-server-localhost.pem")
  -cpuprofile string
    	Write cpu profile to file
  -heartbeat
    	Emit status heartbeat text
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

### Example

```shell
$GOBIN/ciao-scheduler --cacert=/etc/pki/ciao/CAcert-ciao-ctl.intel.com.pem --cert=/etc/pki/ciao/cert-Scheduler-ciao-ctl.intel.com.pem --heartbeat
```

More Information
----------------

See
[ciao-scheduler godoc](https://godoc.org/github.com/01org/ciao/ciao-scheduler)
for more information on the design thinking behind the implementation.
