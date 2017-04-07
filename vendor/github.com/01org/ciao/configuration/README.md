## Ciao configuration

One of ciao's goal is to run on top of a minimal and centralized cluster configuration layer.
The configuration package implements a small API for ciao components to fetch and store
cluster configuration data and use it to configure themselves.

## Overview

Ciao's cluster configuration is stored and fetched from a cluster specific storage backend.
The `ciao-scheduler` is the only component in a ciao cluster that interacts with the configuration
backend. As a consequence it needs to be given a configuration URI through its
`-configuration-uri` command line option. The default value for it is `file:///etc/ciao/configuration.yaml`

The scheduler initially fetches the configuration data by calling `configuration.ExtractBlob()` but
can also get configuration updates from the `ciao-controller` component after it receives a `CONFIGURE`
command from it.

The scheduler propagates configuration data in 2 ways:

1. Through the initial client connection: The scheduler adds a configuration binary payload
(e.g. a marshalled YAML) to the SSNTP `CONNECTED` frame when a client successfully connected.
The client can then fetch the configuration payload by calling `configuration.Payload()`.

2. Through the `CONFIGURE` command: The `ciao-controller` can get configuration updates through
its HTTPS endpoints and will then send the updated configuration data to the `ciao-scheduler`
by adding it to a SSNTP CONFIGURE command.
It is the `ciao-scheduler`'s duty to validate this new configuration data and then forward it
to all ciao SSNTP clients by multicasting a CONFIGURE command to all of them.

### Backends

The ciao configuration package only implements the logic for fetching, storing, validating
and manipulating configuration data. It does not implement configuration physical storage but
instead relies on the configuration storage backend.

Currently supported configuration storage backends is:

* Local file: The local file storage URI should follow the `file://[absolute path to the configuration file]` scheme.

## Configuration data

The ciao configuration data follows a [YAML scheme] (https://github.com/01org/ciao/blob/master/payloads/configure.go):

```
configure:
  scheduler:
    storage_uri: string [The storage URI path]
  storage:
    ceph_id: string [Name used for the Ceph identifier]
  controller:
    compute_port: int
    compute_ca: string [The HTTPS compute endpoint CA]
    compute_cert: string [The HTTPS compute endpoint private key]
    compute_fqdn: string [The HTTPS server fully qualified domain name, MUST match name for which compute_cert is valid]
    identity_user: string [The identity (e.g. Keystone) user]
    identity_password: string [The identity (e.g. Keystone) password]
  launcher:
    compute_net: list [The launcher compute network(s)]
    mgmt_net: list [The launcher management network(s)]
    disk_limit: bool
    mem_limit: bool
  image_service:
    type: string [The image service type, e.g. glance]
    url: string [The image service URL]
  identity_service:
    type: string [The identity service type, e.g. keystone]
    url: string [The identity service URL]
```

## Configuration Examples

### Minimal

```
configure:
  scheduler:
    storage_uri: /etc/ciao/configuration.yaml
  controller:
    compute_ca: /etc/pki/ciao/compute_ca.pem
    compute_cert: /etc/pki/ciao/compute_key.pem
    compute_fqdn: compute.example.com
    identity_user: controller
    identity_password: ciao
  launcher:
    compute_net:
    - 192.168.0.0/16
    mgmt_net:
    - 192.168.0.0/16
  image_service:
    url: http://glance.example.com:9292
  identity_service:
    url: http://keystone.example.com:35357
```

### Complete

```
configure:
  scheduler:
    storage_uri: /etc/ciao/configuration.yaml
  storage:
    ceph_id: ciao
  controller:
    compute_port: 8774
    compute_ca: /etc/pki/ciao/compute_ca.pem
    compute_cert: /etc/pki/ciao/compute_key.pem
    compute_fqdn: compute.example.com
    identity_user: controller
    identity_password: ciao
  launcher:
    compute_net:
    - 192.168.0.0/16
    mgmt_net:
    - 192.168.0.0/16
    disk_limit: true
    mem_limit: true
  image_service:
    type: glance
    url: http://glance.example.com:9292
  identity_service:
    type: keystone
    url: http://keystone.example.com:35357
```
