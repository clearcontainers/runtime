# Clear Containers SELinux module

SELinux module to allow Clear Containers to run

## Install

Run the following commands as root

Create the module

```
dnf install selinux-policy-devel rpm-build
make
```

Fix  /run/cc-oci-runtime/proxy.sock

```
# restorecon -R -v /run/cc-oci-runtime/proxy.sock
```

Insert selinux module

```
# semodule -X 300 -i cc-proxy.pp.bz2
```

Start proxy-socket:

```
# systemctl start cc-proxy.socket
```

Check status on proxy-socket:

```
# systemctl status cc-proxy.socket
● cc-proxy.socket - Clear Containers Proxy Socket
   Loaded: loaded (/usr/lib/systemd/system/cc-proxy.socket; disabled; vendor preset: disabled)
      Active: active (listening) since Tue 2017-01-17 14:36:36 CST; 8min ago
           Docs: https://github.com/01org/cc-oci-runtime/proxy
              Listen: /var/run/cc-oci-runtime/proxy.sock (Stream)

              Jan 17 14:36:36 foo.bar systemd[1]: Listening on Clear Containers Proxy Socket.
              Jan 17 14:36:45 foo.bar systemd[1]: Listening on Clear Containers Proxy Socket.
              Jan 17 14:44:39 foo.bar systemd[1]: Listening on Clear Containers Proxy Socket.
```

References:
* [https://github.com/01org/cc-oci-runtime/issues/519#issuecomment-273294907](https://github.com/01org/cc-oci-runtime/issues/519#issuecomment-273294907)
* [https://lvrabec-selinux.rhcloud.com/2015/07/07/how-to-create-selinux-product-policy/](https://lvrabec-selinux.rhcloud.com/2015/07/07/how-to-create-selinux-product-policy/)
* [https://github.com/mgrepl/docker-selinux](https://github.com/mgrepl/docker-selinux)
