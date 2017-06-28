# Developers Clear Containers 3.0 Install

This guide is not intended for end-users. Instead, this guide provides
instructions for any developers eager to try Clear Containers 3.0 and who
want to build Clear Containers from the source code and are familiar with the
process.

## Requirements

  * [go 1.8.3](https://golang.org/)
  * [glibc-static](https://www.gnu.org/software/libc/libc.html)
  * [gcc](https://gcc.gnu.org/)

## Clear Containers 3.0 components

  * [Runtime](https://github.com/clearcontainers/runtime)
  * [Proxy](https://github.com/clearcontainers/proxy)
  * [Shim](https://github.com/clearcontainers/shim)

**IMPORTANT:** Do not combine [Clear Containers 2.1](https://github.com/01org/cc-oci-runtime) and [Clear Containers 3.0](https://github.com/clearcontainers).
Both projects ship ``cc-proxy`` and they are not compatible with each other.

## Setup the environment

1. Define GOPATH

```bash
$ export GOPATH=$HOME/go
```

2. Create GOPATH Directory

```bash
$ mkdir -p $GOPATH
```

3. Get the code

```bash
$ go get -d github.com/clearcontainers/runtime
$ go get -d github.com/clearcontainers/proxy
$ git clone https://github.com/clearcontainers/shim $GOPATH/src/github.com/clearcontainers/shim
$ go get -d github.com/clearcontainers/tests
```

## Build and install components

1. Proxy

```bash
$ cd $GOPATH/src/github.com/clearcontainers/proxy
$ make
$ sudo make install
```

2. Shim

```bash
$ cd $GOPATH/src/github.com/clearcontainers/shim
$ ./autogen.sh
$ make
$ sudo make install
```

3. Runtime

```bash
$ cd $GOPATH/src/github.com/clearcontainers/runtime
$ export QEMUBINDIR=/usr/bin
$ export SYSCONFDIR=/etc
$ export SHAREDIR=/usr/share
$ export PKGLIBEXECDIR=/usr/libexec/clear-containers
$ export LOCALSTATEDIR=/var
$ make -e
$ sudo -E make -e install
```

For more details on the runtime's build system, run:

```bash
$ make help
```

4. Qemu lite

In Fedora:
```bash
$ source /etc/os-release
$ sudo dnf config-manager --add-repo \
http://download.opensuse.org/repositories/home:/clearlinux:/preview:/clear-containers-2.1/Fedora\_$VERSION_ID/home:clearlinux:preview:clear-containers-2.1.repo
$ sudo dnf install qemu-lite
```

5. Rootfs and kernel images

TODO: kernel version might be old here. Check the latest version in $GOPATH/src/github.com/clearcontainers/tests/.ci/setup_env_ubuntu.sh
```bash
$ export clear_release=$(curl -sL https://download.clearlinux.org/latest)
$ export cc_img_path="/usr/share/clear-containers"
$ export kernel_clear_release=12760
$ export kernel_version="4.5-50"
$ $GOPATH/src/github.com/clearcontainers/tests/.ci/install_clear_image.sh $clear_release $cc_img_path
$ $GOPATH/src/github.com/clearcontainers/tests/.ci/install_clear_kernel.sh $kernel_clear_release $kernel_version $cc_img_path
```

## Enable Clear Containers 3.0 for Docker

1. Clear Containers configuration file

Edit $SYSCONFDIR/clear-containers/configuration.toml according to your needs.

Refer to [https://github.com/clearcontainers/runtime#debugging](https://github.com/clearcontainers/runtime#debugging)
for additional information how to debug the runtime.

2. Configure Docker for Clear Containers 3.0

```bash
$ sudo mkdir -p /etc/systemd/system/docker.service.d/
$ cat << EOF | sudo tee /etc/systemd/system/docker.service.d/clr-containers.conf
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -D --add-runtime clearcontainers=/usr/local/bin/cc-runtime --default-runtime=runc

[Service]
# Allow maximum number of containers to run.
TasksMax=infinity

EOF
```

3. Restart Docker and Clear Containers systemd services

```bash
$ sudo systemctl daemon-reload
$ sudo systemctl restart docker
$ sudo systemctl enable cc-proxy.socket
$ sudo systemctl start cc-proxy.socket
```

## Run Clear Containers 3.0

```bash
$ docker run --runtime clearcontainers -it ubuntu bash
root@6adfa8386497732d78468a19da6365602e96e95c401bec2c74ea1af14c672635:/#
```
