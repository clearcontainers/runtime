# Installing Intel® Clear Containers 3.0 on CentOS* version 7

This document _only_ covers installing on a
[CentOS](https://www.centos.org/) system. It explicitly does **not**
cover installation on Red Hat* Enterprise Linux (RHEL).

Note:

If you are installing Clear Containers 3.X on a system that already has
Clear Containers 2.X installed, first read [the upgrading document](upgrading.md).

## Required setup

You are required to run the `sudo` command without specifying a password
to set up the Clear Containers 3.X installation. Verify this requirement
with the following commands:
```
$ su -
# echo "$some_user ALL=(ALL:ALL) NOPASSWD: ALL" | (EDITOR="tee -a" visudo)
$ exit

```

## The following steps show you how to install Clear Containers 3.X

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

- The installation script might take a long time to run because it
  must download and compile source packages.

- It is not strictly necessary to run the installation script using the
  `script(1)` command, but using this command writes a log of the
  installation to the file `typescript`. This is useful for administrators
  to see the changes made and for use when debugging issues.

- The installation/centos-setup.sh script works for CentOS and selects other variants. To obtain a list of supported variants, please run:
```
$ installation/centos-setup.sh -h
```

## The following steps show how to verify the Clear Containers 3.X installation is successful:

1. Check the `cc-runtime` version:

```
$ cc-runtime --version

```

2. Test that a Clear Container can be created:

```
$ sudo docker run -ti busybox sh

```
