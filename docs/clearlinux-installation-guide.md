# Installing Clear Containers 3.0 on Clear Linux\*

Clear Containers 3.0 is available in Clear Linux since version **17890**.

Note:

If you already have a previous Clear Linux version with Clear Containers 2.x installed,
just update your OS to the latest version to get Clear Containers 3.0.

1. Run the `swupd` command to ensure the host is using the latest version of Clear Linux.

   ```
   $ sudo swupd update
   ```

2. Install the Clear Containers bundle.

   ```
   $ sudo swupd bundle-add containers-virt
   ```

3. Start the Docker\* and Clear Containers `systemd` services.

   ```
   $ sudo systemctl enable docker
   $ sudo systemctl start docker
   $ sudo systemctl enable cc3-proxy
   $ sudo systemctl start cc3-proxy
   ```

4. Run Clear Containers 3.0.

   You are now ready to run Clear Containers 3.0. For example:

   ```
   $ sudo docker run -ti busybox sh
   ```

## More information about Docker in Clear Linux.

Docker on Clear Linux provides a `docker.service` service file to start the `docker` daemon.
The daemon will use `runc` or `cc-runtime` depending on the environment:

If you are running Clear Linux on baremetal or on a VM with Nested Virtualization activated,
`docker` will use `cc-runtime` as the default runtime. If you are running Clear Linux
on a VM without Nested Virtualization, `docker` will use `runc` as the default runtime.
It is not necessary to configure Docker to use `cc-runtime` manually since Docker itself
will automatically use this runtime on systems that support it.

To check which runtime your system is using, run:
```
$ sudo docker info | grep Runtime
```
