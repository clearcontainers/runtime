# clearlinux.keystone
This role installs clearlinux/keystone docker container

## Requirements
* docker

## Role variables

Variable | Default Value | Description
-------- | ------------- | -----------
keystone_fqdn | `{{ ansible_fqdn }}` | Fully Qualified Domain Name for Keystone server
keystone_admin_password | adminUserPassword | Password for the admin user in keystone
mysql_data | /var/lib/mysql | Path to hold mysql database file

#### The following variables can be used to specify custom services, projects, users and roles

<table>
<tr>
<td><b>Variable</b></td>
<td><b>Example</b></td>
<td><b>Description</b></td>
</tr>

<tr>
  <td>keystone_services</td>
  <td><pre><code>
    keystone_services:
      - service: nova
        type: compute
        description: OpenStack Compute Service
  </code></pre></td>
  <td>A list of services to be created</td>
</tr>

<tr>
  <td>keystone_projects</td>
  <td><pre><code>
    keystone_projects:
      - project: demo
        description: Demo Project
  </code></pre></td>
  <td>A list of projects to be created</td>
</tr>

<tr>
  <td>keystone_users</td>
  <td><pre><code>
    keystone_users:
      - user: demo
        password: secret
        project: demo
        email: demo@example.com
  </code></pre></td>
  <td>A list of users to be created</td>
</tr>

<tr>
  <td>keystone_roles</td>
  <td><pre><code>
    keystone_roles:
      - demo
      - admin
  </code></pre></td>
  <td>A list of roles to be created</td>
</tr>

<tr>
  <td>keystone_user_roles</td>
  <td><pre><code>
    keystone_user_roles:
      - user: demo
        project: demo
        role: demo
  </code></pre></td>
  <td>A list of user, role mappings</td>
</tr>

</table>

## Dependencies
None

## Example playbook
file *ciao.yml*
```
- hosts: controllers
  roles:
    - keystone
```

file *group_vars/all*
```
keystone_fqdn: identity.example.com
keystone_admin_password: adminUserPassword
mysql_data: /var/lib/mysql

keystone_projects:
  - project: demo
    description: Demo Project

keystone_users:
  - user: demo
    password: demoUserPassword
    project: demo

keystone_roles:
  - demo

keystone_user_roles:
  - user: demo
    project: demo
    role: demo
```

## License
Apache-2.0

## Author Information
This role was created by [Leoswaldo Macias](leoswaldo.macias@intel.com) and [Obed Munoz](obed.n.munoz@intel.com)

[library/keystone](https://github.com/openstack/openstack-ansible-plugins/blob/master/library/keystone)
taken from [openstack-ansible-plugins](https://github.com/openstack/openstack-ansible-plugins)
