# ciao-cert

ciao-cert is a command line tool for generating [SSNTP](https://github.com/01org/ciao/tree/master/ssntp)
specific certificates. In particular it includes SSNTP roles as part of
the certificate extended key attribute, which is an SSNTP requirement.

ciao-cert generates generates PEM files containing self signed certificates
and private keys for all SSNTP roles.

## Usage

```shell
Usage of ciao-cert:
  -alsologtostderr
        log to standard error as well as files
  -directory string
        Installation directory (default ".")
  -dump string
        Print details about provided certificate
  -elliptic-key
        Use elliptic curve algorithms
  -email string
        Certificate email address (default "ciao-devel@lists.clearlinux.org")
  -host string
        Comma-separated hostnames to generate a certificate for
  -ip string
        Comma-separated IPs to generate a certificate for
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -organization string
        Certificates organization
  -role value
        Comma separated list of SSNTP role [agent, scheduler, controller, netagent, server, cnciagent]
  -anchor
        Whether this cert should be the trust anchor
  -anchor-cert string
        Trust anchor certificate for signing
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -verify
        Verify client certificate
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

## Example

On our example cluster, the scheduler is running on ciao-ctl.intel.com.
The ciao-ctl.intel.com host is a multi homed machine connected to the
cluster control plane through 192.168.1.118.

At a minimum, need to generate 5 private keys for the Scheduler,
Controller, Networking Agent, CNCI Agent and the Compute Node Agents. We
also need to generate the CA certificate.

Depending on your security tolerances, you may choose one of two deployment
styles for SSNTP client certificates:
* a generic private key for each SSNTP client type: ```-host=localhost``` in the command examples below.  Each Compute Node agent machine would have a copy of the one compute node agent key with 'localhost' validity.  This may be easiest for a fleet of ephemeral test systems.
* a unique private key per instance of each SSNTP client type:  For each machine, use a ```-host``` argument containing the fully qualified domain name (FQDN) of that machine.  Each Compute Node agent machine would have a unique compute node agent key valid for that machine's FQDN.  This enables more fine grained access control and revocation.
Note though that the SSNTP server certificate (```-role scheduler```) MUST
have a FQDN specified.

The examples below create a key for the scheduler server specific
to the FQDN "ciao-ctl.intel.com", a key for the controller client
specific to the FQDN "ciao-ctl.intel.com", and generic client keys with
```--host=localhost``` for the remaining SSNTP client classes (compute
node agent, networking node agent, and CNCI agent).

* Scheduler private key and CA certificate

        $GOBIN/ciao-cert -anchor -role scheduler -email=ciao-devel@lists.clearlinux.org -organization=Intel -ip=192.168.1.118 -host=ciao-ctl.intel.com -verify
  That will generate `CAcert-ciao-ctl.intel.com.pem` and `cert-Scheduler-ciao.ctl.intel.com.pem`.
* Controller private key

        $GOBIN/ciao-cert -role controller -anchor-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=ciao-ctl.intel.com -verify
  That will generate `cert-Controller-ciao-ctl.intel.com.pem`.
* Compute Node Agent private key

        $GOBIN/ciao-cert -role agent -anchor-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify
  That will generate `cert-CNAgent-localhost.pem`.
* Networking Node Agent private key

        $GOBIN/ciao-cert -role netagent -anchor-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify
  That will generate `cert-NetworkingAgent-localhost.pem`.
* CNCI Agent private key

        $GOBIN/ciao-cert -role cnciagent -anchor-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify
  That will generate `cert-CNCIAgent-localhost.pem`.

## Multi roles support

In some cases SSNTP clients or servers want to support
several roles at the same time and SSNTP supports that feature.

But certificates need to be generated accordingly, by passing a comma
separated list of roles to ciao-cert.  For example a specific testing
focused launcher agent may want to expose both the CN and NN agent roles:

```shell
$GOBIN/ciao-cert -role agent,netagent -anchor-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify
```

## Inspecting certificates

It is possible to have `ciao-cert` provide some information about the generated
certificates; this is done by using the `-dump` command line flag. Here is some
sample output from the scheduler certificate generated above:

```shell
$GOBIN/ciao-cert -dump ./cert-Scheduler-ciao-ctl.intel.com.pem
Certificate:    ./cert-Scheduler-ciao-ctl.intel.com.pem
Organization:   Intel
Is CA:          true
Validity:       2016-10-12 15:19:39 +0000 UTC to 2017-10-12 15:19:39 +0000 UTC
For role:       Scheduler-
For host:       ciao-ctl.intel.com
For IP:         192.168.1.118
Private key:    RSA PRIVATE KEY
```

## Dealing with certificate issues

### Role mismatches

Ciao cluster certificates implement a role base access control (RBAC) system of
cluster membership.  If a ciao cluster client attempts access using a
certificate whose embedded role does not match the SSNTP client connection
indicated role, the access will be disallowed.  The cluster logs will show
a message, eg:
```
Wrong certificate or missing/mismatched role OID
```
If this is observed, insure your client certificates are created with the
correct roles as indicated above, and your client binaries are run with
configuration using the correct certificate.

### Certificate signed by unknown authority

Ciao cluster certificates are signed by a common certificate authority
(CA).  The above documentation example creates a trust anchor CA with
the ```-anchor```, but you can also use a pre-existing one via the
```-anchor-cert``` option.

Either way, for ciao components to correctly operate, the CA's
certificate must be in the system trust store on each host running a
ciao component.  If it is not, you will see cluster log messages, eg:
```
x509: certificate signed by unknown authority
```
and the cluster will not form.

Depending on your linux distribution, golang runtime, and local IT
policies, the correct way to add your cluster's CA certificate to
your cluster systems' trust stores will vary.  Consult your applicable
documentation.
