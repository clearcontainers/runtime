# clearlinux.ciao-common
This role is a requirement for other ciao-* roles

## Requirements
None

## Role Variables
The following variables are available for all ciao roles

Variable  | Default Value | Description
--------  | ------------- | -----------
gopath | /tmp/go | golang GOPATH
ciao_controller_fqdn | `{{ ansible_fqdn }}` | FQDN for CIAO controller node

## Dependencies
None

## Example Playbook
None

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
