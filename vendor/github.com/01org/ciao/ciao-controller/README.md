Ciao Controller
===============

Ciao controller is responsible for policy choices around tenant workloads.
It provides [compute API
endpoints](https://github.com/01org/ciao/blob/master/ciao-controller/compute.go)
for access from
[ciao-cli](https://github.com/01org/ciao/tree/master/ciao-cli) and
[ciao-webui](https://github.com/01org/ciao-webui) over HTTPS.



Overview
--------

The ciao controller implements an
[SSNTP](https://github.com/01org/ciao/tree/master/ssntp)
client which generates commands sent to
[ciao-scheduler](https://github.com/01org/ciao/tree/master/ciao-scheduler)
and receives node and workload statistics from
[ciao-launcher](https://github.com/01org/ciao/tree/master/ciao-launcher).

Controller integrates with Keystone to allow isolation both between
tenants of a cloud and the administrators of that cloud.  Users within
distinct tenants are also isolated from each other.  Tenant users can
access usage statistics for their tenant workloads and issue commands
to manage their workloads.  Admin users can access usage statistics for
the overall cloud infrastructure and issue commands to manage it.

When a first workload is launched for a tenant,
ciao-controller automatically prepares a [CNCI
appliance](https://github.com/01org/ciao/tree/master/networking/ciao-cnci-agent)
for the tenant.  This provides a virtual network which spans the tenant's
workloads.  Tenant workloads have access only to their tenant private
network and not any other tenant networks.  New workload instances within
the tenant are automatically assigned network connectivity within that
tenant's private network.

Ciao-controller currently has early, developer oriented workload definition
files and a cloud-init template which demonstrate launching virtual
machines and docker workloads (see \*.csv and \*.yaml).


Running Controller
------------------

Controller has many configuration options and depends on connectivity
to a keystone server as well as ciao network node, ciao-scheduler, and
ciao compute nodes configured for ciao-launcher.

The key ciao-controller configuration options describe your keys (-cacert,
-cert, -httpscert, -httpskey), your keystone connection information
(-identity, -username, -password), and the location of your ciao-scheduler
SSNTP server (-url).

### Keystone Configuration

For demonstration purposes, your keystone server needs a the following
minimal configuration for controller:

```shell
$ openstack service create --name ciao compute
$ openstack user create --password hello csr
$ openstack role add --project service --user csr admin
$ openstack user create --password giveciaoatry demo
$ openstack role add --project demo --user demo user
```

This adds a ciao compute service, a keystone user and project for the
controller (aka csr) node, and a demo user with the password
"giveciaoatry".


### Certificates

Certificates are assumed to be in /etc/pki/ciao, or can be
specified on the command line via the "-cert" and "-cacert"
command line options.  Certificates are created with the
[ciao-cert](https://github.com/01org/ciao/tree/master/ssntp/ciao-cert)
tool.

You must also generate SSL certificates for use with the controller’s
HTTPS service, eg:

```shell
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout controller_key.pem -out controller_cert.pem
```

Copy the controller_cert.pem and controller_key.pem files to your
controller node. You can use the same location where you will be
building/running your controller binary (ciao-controller).

### Usage

```shell
Usage of ciao-controller/ciao-controller:
  -alsologtostderr
    	log to standard error as well as files
  -cacert string
    	CA certificate (default "/etc/pki/ciao/CAcert-server-localhost.pem")
  -cert string
    	Client certificate (default "/etc/pki/ciao/cert-client-localhost.pem")
  -database_path string
        path to persistent database (default "/var/lib/ciao/data/controller/ciao-controller.db")
  -image_database_path string
        path to image persistent database (default "/var/lib/ciao/data/image/ciao-image.db")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -nonetwork
    	Debug with no networking
  -stats_path string
    	path to stats database (default "/var/lib/ciao/data/controller/ciao-controller-stats.db")
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -tables_init_path string
	path to csv files (default "/var/lib/ciao/data/controller/tables")
  -url string
    	Server URL (default "localhost")
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -workloads_path string
	path to yaml files (default "/var/lib/ciao/data/controller/workloads")
```

### Example

```shell
sudo ./ciao-controller --cacert=/etc/pki/ciao/CAcert-ciao-ctl.intel.com.pem --cert=/etc/pki/ciao/cert-Controller-localhost.pem --url ciao.ctl.intel.com
```

# OpenStack Compatibility

In order to gain compatibility with common projects/tools as OpenStack Client, Rally Benchmarking and others you need to create the compute service and its corresponding endpoint for keystone. Run the following commands according to your environment as follows:

```
$ source <your-openrc>
$ openstack service create --name ciao --description "CIAO compute" compute
$ openstack endpoint create  compute --region RegionOne public https://<controller>:8774/v2.1/%\(tenant_id\)s
$ openstack endpoint create  compute --region RegionOne admin https://<controller>:8774/v2.1/%\(tenant_id\)s
$ openstack endpoint create  compute --region RegionOne internal https://<controller>:8774/v2.1/%\(tenant_id\)s
```
