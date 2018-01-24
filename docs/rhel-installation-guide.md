# Install Intel® Clear Containers 3.0 on RHEL

## Pre-requisites

Install Docker-EE or Docker-CE on RHEL.

## Required setup

You are required to run the `sudo` command without specifying a password
to set up the Clear Containers 3.X installation. Verify this requirement
with the following commands:
```
$ su -
# echo "$some_user ALL=(ALL:ALL) NOPASSWD: ALL" | (EDITOR="tee -a" visudo)
$ exit

```

The following steps show you how to install Clear Containers 3.X

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
$ script -efc ./installation/rhel-setup.sh

```

Note:

- The `script(1)` command is not strictly necessary to run the installation script but
using this command writes a log of the installation to the `typescript`. Administrators might
find the log useful to see the changes made and when debugging issues.

Verify the Clear Containers 3.X installation is successful:

1. Check the `cc-runtime` version:

```
$ cc-runtime --version

```

2. Test that a Clear Container can be created:

```
$ sudo docker run -ti busybox sh

```
