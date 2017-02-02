.. contents::
.. sectnum::

cc-oci-runtime functional testing
==================================

Functional testing provides a way to verify that ``cc-oci-runtime``
behaves as expected according to OCI Runtime Specification
[#oci-runtime-cli]_ using Clear Containers [#clear-containers]_ .

Requirements
------------

- A Qemu_ hypervisor that supports the ``pc-lite`` machine type.
- A Bats_ Bash Automated Testing System.
- A Clear `containers image`_.
- An uncompressed linux kernel configured for Clear Containers.

Configuration
-------------

Functional testing needs an `OCI bundle`_ directory, the
default bundle directory is ``tests/functional/bundle``.

To  modify the bundle path , run::

./autogen.sh --with-tests-bundle-path=<BUNDLE-PATH>.

The bundle directory must contain:

- rootfs : A directory with root filesystem of a container (linux distribution).

The tests are executed using different config files ` from
``tests/functional/data/config*.json``

cc-oci-runtime creates Clear Containers based  VMs, to do this
three componentes are needed:

- A Clear `containers image`_.
- A kernel configured for Clear Containers.
- Qemu with pc-lite support

If the installed qemu does not support pc-lite, some tests will 
be skipped. 

You can modify the full path to any of these components:

To modify qemu path , run::

  $ ./autogen.sh --with-qemu-path=<qemu-path-with-pc-lite-support>

To modify default Clear Container image path, run::

  $ ./autogen.sh --with-cc-image=<clear-container-image-path>

To modify default Clear Container kernel path, run::

  $ ./autogen.sh --with-cc-kernel=<clear-linux-kernel-path>


Running functional tests
------------------------

Run all the tests (unit tests, functional, valgrind)::

    $ make check

Only run functional tests::

    $ make functional-test

To Run a specific tests is needed to use run-bats.sh script 
to run with network namespace unshared (done by docker); 
otherwise cc-oci-runtime will set container network using host interaces::

    $ bash ./data/run-bats.sh tests/functional/test-name.bats

Links
-----

.. _`Qemu`: http://qemu.org

.. _`bats`: https://github.com/sstephenson/bats

.. _`OCI bundle`: https://github.com/opencontainers/runtime-spec/blob/master/bundle.md

.. _`Containers image`: https://download.clearlinux.org/image/

.. [#oci-runtime-cli]
   https://github.com/opencontainers/runtime-spec/blob/master/runtime.md

.. [#clear-containers]
   https://clearlinux.org/features/clear-containers
