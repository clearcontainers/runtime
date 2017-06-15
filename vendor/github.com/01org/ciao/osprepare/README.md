# osprepare

osprepare is the Operating System Preparation facility of Ciao, enabling
very simple automated configuration of dependencies between various
Linux distributions in a sane fashion.

It provides a simple mechanism for expressing system dependencies and then
handles the OS-specific plumbing transparently.  In this way a ciao
component can in one place articulate its dependencies and with a single
call out get runtime confirmation that the system has the needed code,
without having to care about the details of whether that happens via a call
to apt or rpm or any other package/update manager software.

For more detailed technical information see the [osprepare package godoc
page](https://godoc.org/github.com/01org/ciao/osprepare).
