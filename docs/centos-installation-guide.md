# Installing Intel® Clear Containers 3.0 on CentOS* version 7

This document _only_ covers installing on a
[CentOS](https://www.centos.org/) system. It explicitly does **not**
cover installation on Red Hat* Enterprise Linux (RHEL).

Note:

If you are installing on a system that already has Clear Containers 2.x
installed, first read [the upgrading document](upgrading.md).

## Required Setup

The installation requires the current user to run the `sudo` command
without specifying a password. Verify this with the following commands:

```
$ su -
# echo "$some_user ALL=(ALL:ALL) NOPASSWD: ALL" | (EDITOR="tee -a" visudo)
$ exit

```

## Installation steps

1. Ensure the system packages are up-to-date:

```
$ sudo yum -y update

```
2. Install Git:

```
$ sudo yum install -y git

```
3. Create the installation directory and clone the repository:

```
$ mkdir -p $HOME/go/src/github/clearcontainers
$ cd $HOME/go/src/github/clearcontainers
$ git clone https://github.com/clearcontainers/runtime
$ cd runtime

```
4. Run the installation script:

```
$ script -efc ./installation/centos-setup.sh

```

Note:

- Running the installation script can take a long time as it needs to
  download source packages and compile them.

- Although it is not strictly necessary to run the installation
  script using the `script(1)` command, doing so ensures that a log of the
  installation is written to the file `typescript`. This is useful for
  administrators to see what changes were made and can also be used to
  debug any issues.

## Verify the installation was successful

1. Check the `cc-runtime` version:

```
$ cc-runtime --version

```

2. Test that a Clear Container can be created:

```
$ sudo docker run -ti busybox sh

```
