# Installing Intel® Clear Containers 3.X on SLES* version 12 SP1

This document _only_ covers installing on a
[SLES](https://www.suse.com/products/server/) 12 SP1 system. You can use
this guide with more recent Service Packs, but the procedures have not
been tested.

Note:

If you are installing Clear Containers 3.X on a system that already has \
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
$ sudo -E zypper update
```
2. Install Git:
```
$ sudo -E zypper -n install git

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
$ script -efc ./installation/sles-setup.sh
```

Note:

- The installation script might take a long time to run because it
  must download and compile source packages.

- It is not strictly necessary to run the installation script using the
  `script(1)` command, but using this command writes a log of the
  installation to the file `typescript`. This is useful for administrators
  to see the changes made and for use when debugging issues.

## The following steps show how to verify the Clear Containers 3.X installation is successful:

1. Check the `cc-runtime` version:

```
$ cc-runtime --version
```

2. Test that a Clear Container can be created:

```
$ sudo -E docker run -ti busybox sh
```
