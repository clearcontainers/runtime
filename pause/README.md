## pause binary

This directory provides a ``pause`` binary with the same semantics as
the one included in [https://github.com/containers/virtcontainers](https://github.com/containers/virtcontainers).

The ``pause`` binary is required to allow the creation of an "empty" pod.
The pod does not contain any containers; it simply provides the environment
to allow their creation.

The build step for this file is included in the top-level Makefile.
