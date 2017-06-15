# clearlinux.ciao-launcher
Ansible role to install the compute or network node for a CIAO cluster

## Requirements
* docker
* qemu-kvm
* xorriso

## Role Variables
The available variables for this roles are the variables from [ciao-common](../ciao-common)

## Dependencies
* [ciao-common](../ciao-common)

## Example Playbook
file *ciao.yml*
```
- hosts: computes
  roles:
    - clearlinux.ciao-launcher

- hosts: networks
  roles:
    - clearlinux.ciao-launcher
```

file *group_vars/all*
```
ciao_controller_fqdn: controller.example.com
```

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
