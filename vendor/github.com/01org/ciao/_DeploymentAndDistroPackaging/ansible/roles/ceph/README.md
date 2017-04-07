# ceph
* Optionally installs ceph/demo container on ciao controller node
* Install and configure ceph client on ciao nodes

## Requirements
* Docker (When Installing ceph/demo container on ciao controller node)

## Role Variables

Variable  | Default Value | Description
--------  | ------------- | -----------
cephx_user | admin | cephx user to login into the ceph cluster
ceph_config | files | Method to setup ceph. See defaults/main.yml for more info.
ceph_config_dir | ./ceph | Location for ceph configuration and authentication files (Used with ceph_config=files)
ceph_ip | default ip | IP Address of ceph container (Used with ceph_config=container)
ceph_subnet | default subnet | Subnet of ceph network (Used with ceph_config=container)

## Dependencies
None

## Example Playbook
  - hosts: controllers
    become: yes
    roles:
      - docker
      - ceph

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
