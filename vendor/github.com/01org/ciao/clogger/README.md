Ciao Logger
===========

Given the variety of different requirements for logging in Ciao, there is
a need to provide a custom logging interface that different modules can
share.

The clogger package provides a such a logging interface, adding no
additional imports to avoid issues such as glog getting its flags added
to usage messages. While at the same time rather than provide no default
logging interface other than a CiaoNullLogger implementation, which does
not write messages, a separate package, gloginterface, where a glog
backend is used is also available.
