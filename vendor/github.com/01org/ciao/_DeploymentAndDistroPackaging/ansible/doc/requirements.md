# Fedora
## Ansible requirements
Ansible requires the following packages to be present on fedora managed nodes:
* python-dnf
* netaddr
* libselinux-python

You can install them with the following command
```
sudo dnf install python-dnf netaddr libselinux-python
```

# Proxies
A Proxy server needs to be configured in different parts of the system when deploying
ciao behind a proxy server like in a corporate environment.

## Docker
Controller nodes and Computes nodes runs docker containers. Follow docker documentation
to configure a [HTTP proxy](https://docs.docker.com/engine/admin/systemd/#/http-proxy) on
the docker daemon.

## apt
Ciao automatically install the missing dependencies on launch by calling apt-get
in ubuntu nodes. Configure a HTTP Proxy in apt as shown in [AptConf](https://wiki.debian.org/AptConf)

## dnf
Ciao automatically install the missing dependencies on launch by calling dnf
in fedora nodes. Configure a HTTP Proxy by setting the `proxy` option in [dnf.conf](http://dnf.readthedocs.io/en/latest/conf_ref.html#options-for-both-main-and-repo)
