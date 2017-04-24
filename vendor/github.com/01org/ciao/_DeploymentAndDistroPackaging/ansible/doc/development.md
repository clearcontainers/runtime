## Skip ceph configuration
If you plan to manually setup ceph in your ciao nodes set `skip_ceph = True`
in [group_vars/all](../group_vars/all) file or pass the command line argument
`--extra-vars "skip_ceph=true"`
