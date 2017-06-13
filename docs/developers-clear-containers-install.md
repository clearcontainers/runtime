# Developers Clear Containers 3.0 Install

This guide is not intended for end-users. Instead, this guide provides
instructions for any developers eager to try Clear Containers 3.0 and who
want to build Clear Containers from the source code and are familiar with the
process.

## Requirements

  * [go 1.8.3](https://golang.org/)
  * [glibc-static](https://www.gnu.org/software/libc/libc.html)

## Clear Containers 3.0 components

  * [Runtime](https://github.com/clearcontainers/runtime)
  * [Proxy](https://github.com/clearcontainers/proxy)
  * [Shim](https://github.com/clearcontainers/shim)
  * [Virtcontainers](https://github.com/containers/virtcontainers)

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
$ go get github.com/clearcontainers/runtime
$ go get github.com/clearcontainers/proxy
$ git clone https://github.com/clearcontainers/shim $GOPATH/src/github.com/clearcontainers
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
$ make
$ sudo make install
```

4. Virtcontainers

This step will only install the ``pause`` binary included in [https://github.com/containers/virtcontainers](https://github.com/containers/virtcontainers)

The ``pause`` binary is required to allow the creation of an "empty" pod.
The pod does not contain any containers; it simply provides the environment
to allow their creation.

```bash
$ sudo yum install -y glibc-static
$ cd $GOPATH/src/github.com/clearcontainers/runtime/.ci/
$ ./install_virtcontainers.sh
```

## Enable Clear Containers 3.0 for Docker

1. Create a link of Clear Containers configuration file

This step is needed due to issue [#206](https://github.com/clearcontainers/runtime/issues/206)


```bash
$ mkdir -p /etc/clear-containers
$ ln -s /usr/local/etc/clear-containers/configuration.toml /etc/clear-containers/configuration.toml
```

Edit the file according to your needs.

Refer to [https://github.com/clearcontainers/runtime#global-logging](https://github.com/clearcontainers/runtime#global-logging)
for additional information.

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
