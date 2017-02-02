# Ansible roles for CIAO
This is an example of a playbook to deploy CIAO using ansible.

---
## Prework

### Access
Ansible requires that the user running the playbook has passwordless ssh access
from the deployment machine to the managed nodes and passwordless sudo privileges
on both the managed nodes and deployment machine.

### Requirements

#### CIAO nodes
* Ansible requirements can be found
[here](http://docs.ansible.com/ansible/intro_installation.html#managed-node-requirements),
also check requirements for [fedora](doc/requirements.md#fedora).
* CIAO can be installed in ClearLinux, Fedora 24 and Ubuntu 16.04.
CIAO dependencies will be installed automatically
* If running behind a proxy server read [this](doc/requirements.md#proxies)
* For firewall settings read [this](doc/firewall.md)

#### Deployment machine
The deployment machine can be any Linux OS as long as it has docker installed.

###### Pull the ciao-deploy container
We provide a ready-to-use
[Docker container](https://hub.docker.com/r/clearlinux/ciao-deploy/).
Simply download it and run your setup:

    $ docker pull clarlinux/ciao-deploy

---

## Configuration

### Edit the [hosts](hosts) file according to your cluster setup
```ini
[controllers]
controller.example.com

[networks]
network.example.com

[computes]
compute1.example.com
compute2.example.com
compute3.example.com
```

It's also encouraged to edit [group_vars/all](group_vars/all) file
to change default passwords and other settings.

### Gather ceph config files
Ciao storage is implemented to use ceph as its storage backend. For this reason all ciao nodes
require a copy of the ceph configuration file and authentication token which can be found on
/etc/ceph/ceph.conf and /etc/ceph/ceph.client.admin.keyring files in the ceph monitor node.

In the working directoy, create a `ceph` folder and copy the ceph files mentioned above
before proceeding to the next step.

---

## Start the deployment container with your setup
Once you have edited the `hosts` file, you must map the ciao repository
to the container, so if your current working directory is `./ciao`,
you would start the container as follows:

```
$ docker run --privileged -v /path/to/your/.ssh/key:/root/.ssh/id_rsa \
             -v $(pwd):/root/ciao \
             -v /dev/:/dev/ \
             -it clearlinux/ciao-deploy
```

### Run the playbook

```
# cd /root/ciao/_DeploymentAndDistroPackaging/ansible
# ansible-playbook -i hosts ciao.yml
```

---

## NOTES:

### Running container in privileged mode mapping /dev/
Container is called in *privileged* mode in order to install your certificates
in the CNCI image by using the `losetup` command. Because we need to access
`/dev/loop*` devices, we also need to mount `/dev/` into the container.
To learn more about the Docker options used, please refer to the
[Docker documentation](https://docs.docker.com/engine/reference/commandline/run/).

### Default guest ssh public key
the `group_vars/all` file contains a default public ssh-key which is used
to access the guest VMs ciao launches. We strongly encourage the user to
change the content of the `ciao_guest_key` variable for a known ssh-key,
otherwise the guests won't be accessible using the default value.

### A note on docker hostname resolution
This playbook uses docker containers to start the [identity service](https://hub.docker.com/r/clearlinux/keystone/) and [ciao-webui](https://hub.docker.com/r/clearlinux/ciao-webui/).

Docker containers uses /etc/resolv.conf on the host machine filtering any localhost
address since 'localhost' is not accesible from the container. If after this filtering
there is no nameserver entries in the containers /etc/resolv.conf the daemon adds
public Google DNS Servers (8.8.8.8 and 8.8.4.4) to the containers DNS configuration.

This situation can be caused by NetworkManager which automatically populates /etc/resolv.conf
and has an option to configure a local caching nameserver. If this is your case you can comment
the line "dns=dnsmasq" from /etc/NetworkManager/NetworkManager.conf

Make sure the hosts running docker (controller and compute nodes) have a correctly
configured dns server that can resolve the cluster nodes names.

## Installing ciao from sources
If you are interested in deploying ciao from the master branch, read [this](doc/development.md)
