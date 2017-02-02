# Development Mode
By default, these playbooks install ciao on production environments from
the latest packages available for each supported distro.

However, there are some options that can be specified when running the playbooks
to alter this default behaviour. These options are useful during development and
are described in more detail below:

## Deploy from source code
In order to deploy ciao from the master branch in github set `ciao_dev = True` in [group_vars/all](../group_vars/all) file or pass the
command line argument `--extra-vars "ciao_dev=true"`

## Skip ceph configuration
If you plan to manually setup ceph in your ciao nodes set `skip_ceph = True`
in [group_vars/all](../group_vars/all) file or pass the command line argument
`--extra-vars "skip_ceph=true"`
