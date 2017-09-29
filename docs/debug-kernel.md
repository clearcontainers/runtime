# Get cc-runtime Kernel logs

The `cc-runtime` launches a virtual machine when a new pod is
created. This virtual machine (VM) uses a custom kernel inside the VM.

The kernel boot logs are disabled by default. However these logs may be desired
when the kernel is customized by the end user, or if the virtual machine boot
fails due to the kernel.

To provide a debug log of the kernel boot, Clear Containers supports an option
to enable kernel boot logs using the configuration file.

1. Enable debug in `configuration.toml`

   ```
   $ enable_debug = true
   ```

2. Run a container to generate the logs

   ```
   $ sudo docker run -ti busybox true
   ```

3. Filter the kernel boot logs from the `cc-proxy` logs

   The `cc-proxy` logs show the sources of its collated information. To see
   the kernel boot logs, filter `cc-proxy` logs by the QEMU serial console.
   The QEMU serial console is represented by `source=qemu`.

   ```
   $ journalctl -u cc-proxy | grep source=qemu
   ```

4. Obtain logs using standalone `cc-proxy`

   In some cases it may be easier to run the `cc-proxy` standalone by first stopping
   the `cc-proxy` service and running it standalone to capture the logs.

   ```
   $ sudo systemctl stop cc-proxy
   $ sudo /usr/libexec/clear-containers/cc-proxy -log debug
   ```

   This will result in the `cc-proxy` log being printed on the terminal. This method
   can also be used to capture the logs in a specific `cc-proxy` file.

5. Connect to the Virtual machine console

   If the kernel boots successfully then the virtual machine console can be accessed
   for further debug.

   ```
   socat /run/virtcontainers/pods/<pod id>/console.sock
   ```
