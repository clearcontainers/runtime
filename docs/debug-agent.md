# Get Clear Containers Agent debug logs

Clear Containers runtime launch a special virtual machine when a new pod is
created. This VM uses a VM-agent to spawn processes on behalf of the
pod/container(s) running inside the VM.

From Clear Containers 3.0-beta release a new agent is used,
[Clear Containers Agent](https://github.com/clearcontainers/agent):

The Clear Containers Agent relies heavily on libcontainer used by runc 
(standard program when running containers on bare metal).

To provide a debug log of any agent activity on a guest, the Clear Containers
agent sends logs through a QEMU serial console that are collected by [cc-proxy](https://github.com/clearcontainers/proxy)
and shown in its logs. By default, the Clear Containers agent logs are not collected by 
`cc-proxy` but can be enabled by adding the `-log debug` option to `cc-proxy`.

1- Add the `-log debug` option to  `cc-proxy`  

```
mkdir -p /etc/systemd/system/cc-proxy.service.d/
cat << EOT >  /etc/systemd/system/cc-proxy.service.d/proxy-debug.conf
[Service]
ExecStart=
ExecStart=/usr/libexec/clear-containers/cc-proxy -log debug
EOT
# Restart cc-proxy to provide debug logs.
systemctl daemon-reload
systemctl restart cc-proxy
```
2- Run a container to generate the agent debug logs

```
sudo docker run -ti busybox true
```
3- Filter the agent debug logs from the `cc-proxy` logs
The `cc-proxy` logs show the sources of its collated information. To only see
the agent debug logs, filter `cc-proxy` logs by the QEMU serial console (the
agent logs were sent through it). The QEMU serial console is represented by
`source=qemu`.

```
journalctl -u cc-proxy | grep source=qemu
```

The debug log format is:
DEBU[0019] *AGENT MESSAGE* source=qemu vm=*CONTAINER_ID*
