# Get cc-runtime Kernel logs

The `cc-runtime` launches a virtual machine when a new pod is
created. This virtual machine (VM) uses a custom kernel inside the VM.

The kernel boot logs are disabled by default. However these logs may be desired
when the kernel is customized by the end user, or if the virtual machine boot
fails due to the kernel.

To provide a debug log of the kernel boot, Clear Containers supports an option
to enable kernel boot logs using the configuration file.

1. Enable kernel boot messages

   Set the `enable_debug=` option in the `[proxy.cc]` section to `true`, which
   assumes a standard configuration file path:

   ```
   $ sudo awk '{if (/^\[proxy\.cc\]/) {got=1}; if (got == 1 && /^#enable_debug/) {print "enable_debug = true"; got=0; next; } else {print}}' /usr/share/defaults/clear-containers/configuration.toml
   ```

1. Run a container to generate the logs

   ```
   $ sudo docker run -ti busybox true
   ```

1. Filter the kernel boot logs from the `cc-proxy` logs

   The `cc-proxy` logs show the sources of its collated information. To see
   the kernel boot logs, filter `cc-proxy` logs by the QEMU serial console,
   excluding the agent messages.
   The QEMU serial console is represented by `source=qemu`.

   ```
   $ sudo journalctl -t cc-proxy | grep source=qemu | egrep -v '\<cc-agent\>'
   ```

   To extract all logs entries for a particular container:

   ```
   $ sudo sudo journalctl -t cc-proxy | grep source=qemu | grep vm=CONTAINER_ID | egrep -v '\<cc-agent\>'
   ```
