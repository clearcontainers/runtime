# Installing Clear Containers 3.0 Alpha on Ubuntu

Clear Containers **3.0-alpha** is available for Ubuntu\* **16.04**.

This step is only required in case Docker is not installed on the system.
1. Install the latest version of Docker with the following commands:

```
$ sudo apt-get install \
	apt-transport-https \
	ca-certificates \
	curl \
	software-properties-common
$ curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
$ sudo add-apt-repository \
	"deb [arch=amd64] https://download.docker.com/linux/ubuntu \
	$(lsb_release -cs) \
	stable"
$ sudo apt-get update
$ sudo apt-get install docker-ce
```

For more information on installing Docker please refer to the
[Docker Guide](https://docs.docker.com/engine/installation/linux/ubuntu)

2. Install the Clear Containers 3.0 components with the following commands:

```
$ sudo sh -c "echo 'deb http://download.opensuse.org/repositories/home:/clearcontainers:/clear-containers-3-staging/xUbuntu_16.04/ /' >> /etc/apt/sources.list.d/cc-runtime.list"
$ curl -fsSL http://download.opensuse.org/repositories/home:/clearcontainers:/clear-containers-3-staging/xUbuntu_16.04/Release.key | sudo apt-key add -
$ sudo apt-get update
$ sudo apt-get install -y cc-runtime cc-proxy cc-shim
```

3. Configure Docker to use Clear Containers as the default with the following commands:

```
$ sudo mkdir -p /etc/systemd/system/docker.service.d/
$ cat << EOF | sudo tee /etc/systemd/system/docker.service.d/clr-containers.conf
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -D --add-runtime cc3=/usr/bin/cc-runtime --default-runtime=cc3
EOF
```

4. Restart the Docker and Clear Containers systemd services with the following commands:

```
$ sudo systemctl daemon-reload
$ sudo systemctl restart docker
$ sudo systemctl enable cc-proxy.socket
$ sudo systemctl start cc-proxy.socket
```

5. Run Clear Containers 3.0.

You are now ready to run Clear Containers 3.0. For example:

```
$ sudo docker run -ti fedora bash
```
