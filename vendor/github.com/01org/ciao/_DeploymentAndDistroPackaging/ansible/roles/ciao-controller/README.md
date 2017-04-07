# clearlinux.ciao-controller
Ansible role to install the controller node for a CIAO cluster

This role configures the following components

* ciao scheduler
* ciao controller

## Requirements
None

## Role Variables
The available variables for this roles are the variables from [ciao-common](../ciao-common) plus the following:

Note: Mandatory variables are shown in **bold**

Variable  | Default Value | Description
--------  | ------------- | -----------
ciao_controller_ip | `{{ ansible_default_ipv4['address'] }}` | IP Address for CIAO controller node
ciao_mgmt_subnets | `{{ ansible_default_ipv4['network'] }}` | CIAO management subnets
ciao_compute_subnets | `{{ ciao_mgmt_subnet }}` | CIAO compute subnets
ciao_service_user | ciao | OpenStack user for CIAO services
ciao_service_password | ciaoUserPassword | Password for `ciao_service_user`
ciao_admin_email | admin@example.com | CIAO administrator email address
ciao_cert_organization | Example Inc. | Name of the organization running the CIAO cluster
ceph_id | admin | Cephx user to authenticate
secret_path | /etc/ceph/ceph.client.admin.keyring| Path to ceph user keyring
cnci_image_url | [clear-8260-ciao-networking.img.xz](https://download.clearlinux.org/demos/ciao/clear-8260-ciao-networking.img.xz) | URL for the latest ciao networking image
clear_cloud_image_url | [clear-11960-cloud.img.xz](https://download.clearlinux.org/releases/11960/clear/clear-11960-cloud.img.xz) | URL for the latest clearlinux cloud image
fedora_cloud_image_url | [Fedora-Cloud-Base-24-1.2.x86_64.qcow2](https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2) | URL for the latest fedora cloud image
firmware_image_url | [OVMF.fd](https://download.clearlinux.org/image/OVMF.fd) | URL for OVMF firmware

## Dependencies
* [ciao-common](../ciao-common)

## Example Playbook
file *ciao.yml*
```
- hosts: controllers
  roles:
    - clearlinux.ciao-controller
```

file *group_vars/all*
```
keystone_fqdn: identity.example.com
keystone_admin_password: secret

ciao_service_user: csr
ciao_service_password: secret
```

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
