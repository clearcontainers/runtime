# CNCI Test Server #

## Overview ##

A simple SSNTP server that can be used to perform unit testing of the CNCI
agent. The CNCI Test Server sends a stream of event and command to any CNCI 
agent that registers with it.

The CNCI Agent is expected to handle all the requests appropriately.

### Warning ###

If the server is used to test a CNCI running with a CNCI VM then the server
has to be run a machine that is different from the one that is hosting the 
CNCI VM. 

This is required as the CNCI VM uses macvatap for networking and macvtap 
traffic cannot be received by the host whichout complex network plumbing


