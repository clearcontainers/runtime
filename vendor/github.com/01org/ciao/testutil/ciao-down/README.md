# ciao-down

ciao-down is a small utility for setting up a VM that contains
everything you need to run ciao's Single VM. All you need to have
installed on your machine is:

- Go 1.7 or greater

Then simply type

```
go get github.com/01org/ciao/testutil/ciao-down
$GOPATH/bin/ciao-down prepare
```

ciao-down will install some needed dependencies on your local PC such
as qemu and xorriso. It will then download an Ubuntu Cloud Image and
create a VM based on this image. It will boot the VM and install in that
VM everything you need to run ciao Single VM, including docker, ceph,
go, gcc, etc. When ciao-down prepare has finished you can connect to the
newly created VM with

```
$GOPATH/bin/ciao-down connect
```

Your host's GOPATH is mounted inside the VM. Thus you can edit your
the ciao code on your host machine and test in Single VM.

Follow the instructions at the prompt to start Single VM.

The nice thing about ciao-down is that it is built using existing ciao
packages. It uses osprepare to install dependencies on the host, some
launcher code to create the images needed by the VMs, and the qemu
package to launch and manage the VMs. All the rest is done via cloud-init.

## ciao-down commands

### prepare

ciao-down prepare creates and configures a new ciao-down VM.  Currently,
only one VM can be configured at any one time.  All the files associated
with the VM are stored in ~/.ciao-down.  The first time you run ciao-down
it will download the Ubuntu 16.04 cloud image which takes some time.  This
image is cached in ~/.ciao-down for future use.

An example of ciao-down prepare is given below:

```
$ ./ciao-down prepare
Booting VM with 7 GB RAM and 4 cpus
Booting VM : [OK]
Downloading Go : [OK]
Unpacking Go : [OK]
Installing apt-transport-https and ca-certificates : [OK]
Add docker GPG key : [OK]
Retrieving updated list of packages : [OK]
Upgrading : [OK]
Installing Docker : [OK]
Installing GCC : [OK]
Installing Make : [OK]
Installing QEMU : [OK]
Installing xorriso : [OK]
Installing ceph-common : [OK]
Installing Openstack client : [OK]
Updating NodeJS sources : [OK]
Installing NodeJS : [OK]
Auto removing unused components : [OK]
Building ciao : [OK]
Installing Go development utils : [OK]
Retrieving ciao-webui  : [OK]
Pulling ceph/demo : [OK]
Pulling clearlinux/keystone : [OK]
Downloading Fedora-Cloud-Base-24-1.2.x86_64.qcow2 : [OK]
Downloading CNCI image : [OK]
Downloading latest clear cloud image : [OK]
Setting git user.name : [OK]
Setting git user.email : [OK]
VM successfully created!
Type ciao-down connect to start using it.
```

By default, ciao-down will assign half of your host's resources to the VM
that it creates and launches.  If you have 8 CPUs and 8 GB of RAM on your
host, ciao-down will assign 4GB of RAM and 4 VCPUs to the guest VM.  You
can control this behaviour by using the --mem and --cpu options.  For
example,

ciao-down prepare --cpus 2 -mem 2

Creates and boots a VM with 2 VCPUs and 2 GB of RAM.

In addition to downloading and building the ciao source code, ciao-down also
downloads the code for ciao's ui, ciao-webui.  By default it will place the
source in a local directory in $HOME/ciao-webui.  However, if you are modifying
the code of the webui it is convenient to have the code shared between your host
machine and the ciao-down VM.  This can be achieved using the --ui-path.  This
option takes a path to the location on your host machine where the UI code is
stored.  This path gets mounted to the same location inside the VM, allowing you to
modify the sources on your host and compile inside the VM.  For example,

ciao-down prepare --with-ui $HOME/src/ciao-webui

will create a new ciao-down VM in which the $HOME/src/ciao-webui folder on
your host will be mounted at $HOME/src/ciao-webui.

Note that if the directory passed to --with-ui option already contains a
git repo, ciao-down will perform no action other than to mount the directory
in the guest VM.  It will not try to clone the repo or update it.  It assumes
that the directory already contains the webui code.

ciao-down by default creates a VM suitable for ciao development. It also supports
setting up development enviornments for other projects.

ciao-down prepare -vmtype clearcontainers

will create a new ciao-down VM for development of clearcontainers.

### delete

ciao-down delete, shuts down and deletes all the files associated with the VM.

### status

ciao-down status provides information about the current ciao-down VM, e.g., whether
it is running, and how to connect to it.  For example,

```
$ ciao-down status
Status	:	ciao up
SSH	:	ssh -q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i /home/user/.ciao-down/id_rsa 127.0.0.1 -p 10022
```

### stop

ciao-down stop is used to power down the ciao-down VM cleanly.

### start

ciao-down start boots a previously prepared but not running ciao-down VM.
The start command also supports the --mem and --cpu options.  So it's
possible to change the resources assigned to the guest VM by stopping it
and restarting it, specifying --mem and --cpu.

### quit

ciao-down quit terminates the VM immediately.  It does not shut down the OS
running in the VM cleanly.

## Git config

Ciao-down will try to copy the git config variables, user.name and user.email,
from the host to the newly created guest during the execution of the prepare
command.  This is done to allow you to create git commits from inside the guest.
Normally, you would modify the ciao source code and submit any changes on
the host.  However, it is sometimes useful to be able to create commits from
inside the guest, for example, if you were to ssh into the guest from a machine
that is not the host.


