## Ceph configuration
Ciao requires a working ceph cluster already in place.

If you already have configured all ciao nodes to be clients of the ceph cluster,
you can tell ansible to skip the configuration of ceph by setting `ceph_config = none`
in [group_vars/all](../group_vars/all) file or pass the command line argument
`--extra-vars "ceph_config=none"`

If you don't have a working ceph cluster, ansible can deploy a ceph container in the
deployment node by setting `ceph_config = container` in [group_vars/all](../group_vars/all)
file or pass the command line argument `--extra-vars "ceph_config=container"`

Note that this container is for demo/development purposes and should NEVER be used in production.
