# clearlinux.ciao-webui
This role installs clearlinux/ciao-webui docker container

## Requirements
Docker

## Role variables

Variable  | Default Value | Description
--------  | ------------- | -----------
keystone_fqdn | `{{ ansible_fqdn }}` | Hostname of the identity host
ciao_webui_fqdn | `{{ ansible_fqdn }}` | Hostname of the webui host

## Dependencies
None

## Example playbook
file *ciao.yml*
```
- hosts: controllers
  roles:
    - ciao-webui
```

## License
Apache-2.0

## Author Information
This role was created by [Erick Cardona](erick.cardona.ruiz@intel.com)
