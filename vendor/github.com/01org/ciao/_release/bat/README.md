# BAT tests

This folder contains a set of BAT tests.  Each set of tests validates a specific
part of ciao, such as storage, and is implemented by a separate go package in one of
the following sub folders.  

```
.
├── base_bat     - Basic tests that verify that the cluster is functional
├── image_bat    - BAT tests for the image service
├── storage_bat  - Storage related BAT tests
```

The tests are implemented using the go testing framework.  This is convenient
as this framework is used for ciao's unit tests and so is already familiar
to ciao developers, it requires no additional dependencies and it works with ciao's
existing test case runner, test-cases.

# A Short Guide to Running the BAT Tests

## Set up

The BAT tests require a running ciao cluster to execute.  This can be a
full ciao cluster running on hundreds of nodes or a Single VM ciao cluster
running on a single machine.  For more information about Single VM see
[here](https://github.com/01org/ciao/wiki/Single-Machine-Development-Environment).

The BAT tests have some dependencies. The device on which they are run must have
qemu-img installed and must also have access to the ceph cluster. The controller
node in a ciao cluster fulfills both of these requirements so it is often easiest
to run the BAT tests from this node. There is only one node in Single VM clusters,
so when using Single VM, simply run the BAT tests from the device on which you ran
setup.sh.

The BAT tests require that certain environment variables have been set before they
can be run:

* "CIAO_IDENTITY" - the URL and port number of your identity service
* "CIAO_CONTROLLER" - the URL and port number of the ciao compute service
* "CIAO_USERNAME" - a test user with user level access to a test tenant
* "CIAO_PASSWORD" - your test user's password
* "CIAO_ADMIN_USERNAME" - your cluster admin user name
* "CIAO_ADMIN_PASSWORD" - your cluster admin password.

Note if you are using Single VM a script will be created for you called
~/local/demo.sh that initialises these variables to their correct
values for the Single VM cluster.  You just need to source this file
before running the tests, e.g.,

```
. ~/local/demo.sh
```

## Run the BAT Tests and Generate a Pretty Report

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# test-cases ./...
```

## DON'T use go test to run the BAT tests!

You might be forgiven for thinking that the easiest way to run all the
BAT tests would be to do the following.

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# go test -v ./...
```

This currently does not work.  The reason is that when go test is run
with a wildcard and that wildcard matches multiple packages the tests
for all of these packages are run in parallel.  As all tests are run in
the same tenant and some tests call ciao-cli instance delete -all, the
tests from different packages can interfere with each other.  This means
we cannot safely use go test to run all the BAT tests.  Once
we have ciao-cli support for tenant creation, it will be possible to
have each set of tests create their own tenants.  It will then be
possible to run the BAT tests concurrently.  For the time being, use
test-cases which runs the tests for all packages serially.  go test
can be safely used to run the BAT tests for a specific package.


## Run the BAT Tests and Generate TAP report

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# test-cases -format tap ./...
```

## Run the BAT Tests and Generate a Test Plan

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# test-cases -format html ./...
```

## Run a Single Set of Tests

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# go test -v github.com/01org/ciao/_release/bat/base_bat
```

## Run a Single Test

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# go test -v -run TestGetAllInstances ./...
```
