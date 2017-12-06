# Get Clear Containers Agent debug logs

Clear Containers runtime launch a special virtual machine when a new pod is
created. This VM uses a VM-agent to spawn processes on behalf of the
pod/container(s) running inside the VM.

From Clear Containers 3.0-beta release a new agent (`cc-agent`) is used,
[Clear Containers Agent](https://github.com/clearcontainers/agent).

The Clear Containers agent relies heavily on [`libcontainer`](https://github.com/opencontainers/runc/tree/master/libcontainer) used by [`runc`](https://github.com/opencontainers/runc/) (standard program when running containers on bare metal).

To provide a debug log of any agent activity on a guest, the Clear Containers
agent sends logs through a QEMU serial console that are collected by [cc-proxy](https://github.com/clearcontainers/proxy)
and shown in its logs. By default, the Clear Containers agent logs are not collected by 
`cc-proxy` but can be enabled by enabling the proxy debug option.

1. Enable proxy debug

   Set the `enable_debug=` option in the `[proxy.cc]` section to `true` (assumes a standard configuration file path):

   ```
   $ sudo awk '{if (/^\[proxy\.cc\]/) {got=1}; if (got == 1 && /^#enable_debug/) {print "enable_debug = true"; got=0; next; } else {print}}' /usr/share/defaults/clear-containers/configuration.toml
   ```

1. Run a container to generate the logs:

   ```
   $ sudo docker run -ti busybox true
   ```

1. Filter the agent debug logs from the `cc-proxy` logs

   The `cc-proxy` logs show the sources of its collated information. To only see
   the agent debug logs, filter `cc-proxy` logs by the QEMU serial console (the
   agent logs were sent through it). The QEMU serial console is represented by
   `source=qemu`.

   ```
   $ sudo journalctl -t cc-proxy | grep source=qemu | egrep '\<cc-agent\>'
   ```

   To extract all logs entries for a particular container:

   ```
   $ sudo sudo journalctl -t cc-proxy | grep source=qemu | grep vm=CONTAINER_ID | egrep '\<cc-agent\>'
   ```
