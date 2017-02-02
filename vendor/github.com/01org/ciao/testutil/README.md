Ciao Testutil
=============

A given ciao component will have unit tests which cover its internals in a
component specific way.  But this is limited in scope and cannot cover all
of the component's functionality, because that functionality is ultimately
about creating and servicing SSNTP command/event/error/status flows.

To enable testing of those SSNTP flows, the testutil package provides
a set of common test tools for ciao.  Included are:

* shared payload constants
* shared test certificates
* test agent implementation for ssntp.AGENT and ssntp.NETAGENT roles
* test controller implementation for ssntp.Controller role
* test server implementation for ssntp.SERVER role
* example client-server test spanning the above SSNTP actors
* channels for tracking command/event/error/status flows across the SSNTP
  test actors

This allows a ciao component to be tested more meaninfully, but in partial
isolation.  The ciao component under test would be a real implementation,
but its SSNTP peers are the shared synthetic implementations from the
testutil package.

For an example of how to enable basic SSNTP test flows
and track the results through the channel helpers
review the internals of the [example client-server
test](https://github.com/01org/ciao/blob/master/testutil/client_server_test.go).

Additional test options
=======================

Virtual
-------

The next level of test breadth comes from actually running
a test cluster with real implementations of each ciao
component.  This is provided in a controlled fashion via the
[singlevm](https://github.com/01org/ciao/tree/master/testutil/singlevm)
CI script, which has detailed documentation
on the wiki at [Single VM Development
Environment](https://github.com/01org/ciao/wiki/HOWTO:-Single-VM-Development-Environment).

Physical
--------

Testing is of course also possible on a real hardware cluster which has
been set up according to the [cluster setup
guide](https://clearlinux.org/documentation/ciao-cluster-setup.html).
A minimal Build Acceptance Test (BAT) framework outputting TAP
(Test Anything Protocol) results is published [in our release
tools](https://github.com/01org/ciao/tree/master/_release/bat).
The python script drives ciao-cli to query and manipulate the state of
a cluster.
