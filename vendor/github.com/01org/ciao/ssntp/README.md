# Simple and Secure Node Transfer Protocol #

## Overview ##

The Simple and Secure Node Transfer Protocol (SSNTP) is a custom, fully
asynchronous and TLS based application layer protocol. All Cloud Integrated
Advanced Orchestrator (CIAO) components communicate with each others over SSNTP.

SSNTP is designed with simplicity, efficiency and security in mind:

* All SSNTP entities are identified by a Universal Unique IDentifier (UUID).
* All SSNTP frame headers are identical for easy parsing.
* SSNTP payloads are optional.
* SSNTP payloads are YAML formatted.
* SSNTP is a one way protocol where senders do not receive a synchronous
  answer from the receivers.
* Any SSNTP entity can asynchronously send a command, status or event to
  one of its peers.

## SSNTP clients and servers ##

The SSNTP protocol defines 2 entities: SSNTP clients and SSNTP servers.

A SSNTP server listens for and may accept connections from many SSNTP
clients. It never initiates a connection to another SSNTP entity.

A SSNTP client initiates a connection to a SSNTP server and can
only connect to one single server at a time. It does not accept
incoming connections from another SSNTP entity.

Once connected, both clients and servers can initiate SSNTP transfers
at any point in time without having to wait for any kind of SSNTP
acknowledgement from the other end of the connection. SSNTP is a fully
asynchronous protocol.

### Roles ###

All SSNTP entities must declare their role at connection time, as part
of their signed certificate extended key usage attributes.

SSNTP roles allow for:

1. SSNTP frames filtering: Depending on the declared role of the sending entity,
   the receiving party can choose to discard frames and optionally send a
   frame rejection error back.
2. SSNTP frames routing: A SSNTP server implementation can configure frame
   forwarding rules for multicasting specific received SSNTP frame types to
   all connected SSNTP clients with a given role.

There are currently 6 SSNTP different roles:

* SERVER (0x1): A generic SSNTP server.
* Controller (0x2): The CIAO Command and Status Reporting client.
* AGENT (0x4): The CIAO compute node Agent. It receives workload
  commands from the Scheduler and manages workload on a given compute
  node accordingly.
* SCHEDULER (0x8): The CIAO workload Scheduler. It receives workload
  related commands from the Controller and schedules them on the available compute
  nodes.
* NETAGENT (0x10): The CIAO networking compute node Agent. It receives
  networking workload commands from the Scheduler and manages workload on a
  given networking compute node accordingly.
* CNCIAGENT (0x20): A Compute Node Concentrator Instance Agent runs within
  the networking node workload and manages a specific tenant private network.
  All instances for this tenant will have a GRE tunnel established between
  them and the CNCI, and the CNCI acts as the tenant routing entity.

## SSNTP connection ##
Before a SSNTP client is allowed to send any frame to a SSNTP server,
or vice versa, both need to successfully go through the SSNTP
connection protocol.
The SSNTP connection is a mandatory step for the client and the
server to verify each other's roles and also to retrieve each other's
UUIDs.

1. SSNTP client sends a CONNECT command to the SSNTP server. This
   frame contains the advertised SSNTP client and this should match
   the client's certificate extended key usage attributes. The server
   will verify that both match and if they don't it will send a SSNTP
   error frame back with a ConnectionAborted (0x6) error code.
   The CONNECT frame destination UUID is the nil UUID as the client
   does not know the server UUID before getting its CONNECTED frame.

2. The server asynchronously sends a CONNECTED status frame to the
   client in order to notify him about a successful connection. The
   CONNECTED frame contains the server advertised role and the cluster
   configuration data in its payload.
   The client must verify that the server role matches its certificate
   extended key usage attributes. If that verification fails the client
   must send a SSNTP error frame to the server where the error code is
   ConnectionFailure (0x4), and then must close the TLS connection to
   the server.
   The client should also parse the cluster
   [configuration data] (https://github.com/01org/ciao/blob/master/payloads/configure.go)
   that comes in the CONNECTED payload and configure itself accordingly.

3. Connection is successfully established. Both ends of the connection
   can now asynchronously send SSNTP frames.

## SSNTP certificates ##

SSNTP uses ciao-cert to generate the certificates it needs to communicate. They
can be generated with instructions found in [ciao-cert] (https://github.com/01org/ciao/tree/master/ciao-cert).

## SSNTP frames ##

Each SSNTP frame is composed of a fixed length, 8 bytes long header and
an an optional YAML formatted payload.

### SSNTP header ###

```
+----------------------------------------------------------------+
|   Major  |   Minor  |   Type   | Operand  |  Payload Length    |
| (1 byte) | (1 byte) | (1 byte) | (1 byte) |  or Role (4 bytes) |
+----------------------------------------------------------------+
```

* Major is the SSNTP version major number. It is currently 0.
* Minor is the SSNTP version minor number. It is currently 1.
* Type is the SSNTP frame type. There are 4 different frame types:
  COMMAND, STATUS, EVENT and ERROR.
* Operand is the SSNTP frame sub-type.
* Payload length is the optional YAML formatted SSNTP payload length
  in bytes. It is set to zero for payload less frames.
* Role is the SSNTP entity role. Only the CONNECT command and
  CONNECTED status frames are using this field as a role descriptor.

### SSNTP COMMAND frames ###

There are 10 different SSNTP COMMAND frames:

#### CONNECT ####
CONNECT must be the first frame SSNTP clients send when trying to
connect to a SSNTP server. Any frame sent to a SSNTP server from a
client that did not initially sent a CONNECT frame will be discarded
and the TLS connection to the client will be closed.

The purpose of the CONNECT command frame is for the client to advertise
its role and for the server to verify that the advertised role matches
the client's certificate extended key usage attributes.

The CONNECT frame is payloadless and its Destination UUID is the nil
UUID:

```
+--------------------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |          Role             | Client UUID | Nil UUID |
|       |       | (0x0) |  (0x0)  | (bitmask of client roles) |             |          |
+--------------------------------------------------------------------------------------+
```

#### START ####
The CIAO Controller client sends the START command to the Scheduler in
order to schedule a new workload. The [START command YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/start.go)
is mandatory and contains a full workload description.

If the Scheduler finds a compute node (CN) with enough capacity to run
this workload, it will then send a START command to the given Agent
UUID managing this CN with the same payload.

If the Scheduler cannot find a suitable CN for this workload, it will
asynchronously send a SSNTP ERROR frame back to the Controller. The error
code should be StartFailure (0x2) and the payload must comply with the
[StartFailure YAML schema] (https://github.com/01org/ciao/blob/master/payloads/startfailure.go)
so that the Controller eventually knows that a given instance/workload UUID
could not start.

Once the Scheduler has sent the START command to an available CN Agent,
it is up to this Agent to actually initialize and start an instance
that matches the START YAML payload. If that fails the Agent should
asynchronously sends a SSNTP ERROR back to the Scheduler and the error
code should be StartFailure (0x2). The Scheduler must then forward that
error frame to the Controller.

The START command payload is mandatory:

```
+--------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted       |
|       |       | (0x0) |  (0x1)  |                 | workload description |
+--------------------------------------------------------------------------+
```

#### STOP ####
The CIAO Controller client sends the STOP command to the Scheduler in
order to stop a running instance on a given CN. The [STOP command
YAML payload] (https://github.com/01org/ciao/blob/master/payloads/stop.go)
is mandatory and contains the instance UUID to be stopped and the
agent UUID that manages this instance.

STOPping an instance means shutting it down. Non persistent
instances are deleted as well when being STOPped.
Persistent instances metadata and disks images are stored and
can be started again through the RESTART SSNTP command.

There are several error cases related to the STOP command:

1. If the Scheduler cannot find the Agent identified in the STOP
   command payload, it should send a SSNTP error with the
   StopFailure (0x3) error code back to the Controller.

2. If the Agent cannot actually stop the instance (Because e.g.
   it's already finished), it should also send a SSNTP error with
   the StopFailure (0x3) error code back to the Scheduler. It is
   then the Scheduler responsibility to notify the Controller about it
   by forwarding this error frame.

```
+--------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted |
|       |       | (0x0) |  (0x2)  |                 |     payload    |
+--------------------------------------------------------------------+
```

#### STATS ####
CIAO CN Agents periodically send the STATS command to the Scheduler
in order to provide a complete view of the compute node status. It is
up to the CN Agent implementation to define the STATS sending period.

Upon reception of Agent STATS commands, the Scheduler must forward it
to the Controller so that it can provide a complete cloud status report back to
the users.

The STATS command comes with a mandatory [YAML formatted payload]
(https://github.com/01org/ciao/blob/master/payloads/stats.go).

```
+----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted compute |
|       |       | (0x0) |  (0x3)  |                 | node statistics        |
+----------------------------------------------------------------------------+
```

#### EVACUATE ####
The CIAO Controller client sends EVACUATE commands to the Scheduler
to ask a specific CIAO Agent to evacuate its compute node, i.e.
stop and migrate all of the current workloads it is monitoring on
its node.

The [EVACUATE YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/evacuate.go)
is mandatory and describes the next state to reach after evacuation
is done. It could be 'shutdown' for shutting the node down, 'update'
for having it run a software update, 'reboot' for rebooting the node
or 'maintenance' for putting the node in maintenance mode:

```
+---------------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted compute      |
|       |       | (0x0) |  (0x4)  |                 | node next state description |
+---------------------------------------------------------------------------------+
```

#### DELETE ####
The CIAO Controller client may send DELETE commands in order to
completely remove an already STOPped instance from the cloud.
This command is only relevant for persistent workload based instances
as non persistent instances are implicitly deleted when being STOPed.

Deleting a persistent instance means completely removing it from
the cloud and thus it should no longer be reachable for e.g. a
RESTART command.

When asked to delete a non existing instance the CN Agent
must reply with a DeleteFailure error frame.

The [DELETE YAML payload schema]
(https://github.com/01org/ciao/blob/master/payloads/stop.go)
is the same as the STOP one.

```
+--------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted |
|       |       | (0x0) |  (0x5)  |                 |     payload    |
+--------------------------------------------------------------------+
```

#### RESTART ####
The CIAO Controller client may send RESTART commands in order to
restart previously STOPped persistent instances.
Non persistent instances cannot be RESTARTed as they are
implicitly deleted when being STOPped.

When asked to restart a non existing instance the CN Agent
must reply with a RestartFailure error frame.

The [RESTART YAML payload schema]
(https://github.com/01org/ciao/blob/master/payloads/start.go)
is the same as the STOP one.

```
+--------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted |
|       |       | (0x0) |  (0x6)  |                 |     payload    |
+--------------------------------------------------------------------+
```

#### AssignPublicIP ####
AssingPublicIP is a command sent by the Controller to assign
a publicly routable IP to a given instance. It is sent
to the Scheduler and must be forwarded to the right CNCI.

The public IP is fetched from a pre-allocated pool
managed by the Controller.

The [AssignPublicIP YAML payload schema]
(https://github.com/01org/ciao/blob/master/payloads/assignpublicIP.go)
is made of the CNC, the tenant and the instance UUIDs,
the allocated public IP and the instance private IP and MAC.

```
+--------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted |
|       |       | (0x0) |  (0x7)  |                 |     payload    |
+--------------------------------------------------------------------+
```

#### ReleasePublicIP ####
ReleasePublicIP is a command sent by the Controller to release
a publicly routable IP from a given instance. It is sent
to the Scheduler and must be forwarded to the right CNCI.

The released public IP is added back to the Controller managed
IP pool.

The [ReleasePublicIP YAML payload schema]
(https://github.com/01org/ciao/blob/master/payloads/assignpublicIP.go)
is made of the CNCI and a tenant UUIDs, the released
public IP, the instance private IP and MAC.

```
+--------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted |
|       |       | (0x0) |  (0x8)  |                 |     payload    |
+--------------------------------------------------------------------+
```

#### CONFIGURE ####
CONFIGURE commands are sent to request any SSNTP entity to
configure itself according to the CONFIGURE command payload.
Controller or any SSNTP client handling user interfaces defining any
cloud setting (image service, networking configuration, identity
management...) must send this command for any configuration
change and for broadcasting the initial cloud configuration to
all CN and NN agents.

CONFIGURE commands should be sent in the following cases:

* At cloud boot time, as a broadcast command.
* For every cloud configuration change.
* Every time a new agent joins the SSNTP network.

The [CONFIGURE YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/configure.go)
always includes the full cloud configuration and not only changes
compared to the last CONFIGURE command sent.

```
+-----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload  |
|       |       | (0x0) |  (0x9)  |                 |                         |
+-----------------------------------------------------------------------------+
```

#### AttachVolume ####
AttachVolume is a command sent to ciao-launcher for attaching a storage volume
to a specific running or paused instance.

The AttachVolume command payload includes a volume UUID and an instance UUID.

```
+-----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload  |
|       |       | (0x0) |  (0xa)  |                 |                         |
+-----------------------------------------------------------------------------+
```

#### DetachVolume ####
DetachVolume is a command sent to ciao-launcher for detaching a storage volume
from a specific running or paused instance.

The DetachVolume command payload includes a volume UUID and an instance UUID.

```
+-----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload  |
|       |       | (0x0) |  (0xb)  |                 |                         |
+-----------------------------------------------------------------------------+
```

#### Restore ####

Restore is used to ask a specific CIAO agent that had previously been placed into
maintenance mode by an EVACUATE command to start accepting new instances once more.

The payload for this command contains the UIID of the node to restore.

```
+---------------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload      |
|       |       | (0x0) |  (0x4)  |                 |                             |
+---------------------------------------------------------------------------------+
```

### SSNTP STATUS frames ###

There are 5 different SSNTP STATUS frames:

#### CONNECTED ####
CONNECTED is sent by SSNTP servers back to a client to notify it
that the connection successfully completed.

From the CONNECTED frame the client will gather 2 pieces of
information:

1. The server UUID. This UUID will be used as the destination UUID
   for every frame the client sends going forward.
2. The server Role. The client must verify that the server TLS
   certificate extended key usages attributes match the advertise
   server Role. If it does not, the client must discard and close
   the TLS connection to the server.

The CONNECTED frame payload is the same as the
[CONFIGURE one](https://github.com/01org/ciao/blob/master/payloads/configure.go)
and contains cluster configuration data.

```
+--------------------------------------------------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |         Role              | Server UUID | Client UUID | Payload | YAML formatted |
|       |       | (0x1) |  (0x0)  | (bitmask of server roles) |             |             |  Length |      payload   |
+--------------------------------------------------------------------------------------------------------------------+
```

#### READY ####
SSNTP compute node Agents send READY status frames to let the
Scheduler know that:

1. Their CN capacity has changed. The new capacity is described
   in the [READY YAML payload]
   (https://github.com/01org/ciao/blob/master/payloads/ready.go).
   This is the main piece of information the Scheduler uses to
   make its instances scheduling decisions.
2. They are ready to take further commands, and in particular to
   start new workloads on the CN they manage. It is important to
   note that a Scheduler should not send a new START commands to
   a given Agent until it receives the next READY status frame
   from it.
   Some Scheduler implementations may implement opportunistic
   heuristics and send several START commands after receiving a
   STATUS frame, by forecasting CN capacities based on the START
   command payloads they previously sent. This allow them to
   reach shorter average instances startup times at the risk of
   hitting higher than expected cloud overcommit ratios.

The READY status payload is almost a subset of the STATS command
one as it does describe the CN capacity status without providing
any details about the currently running instances. There are
several differences between READY and STATS:

* READY frames are asynchronous while STATS frames are periodic.
  Agent implementations will typically send READY status to the
  Scheduler after successfully starting a new instance on the CN
  while they send STATS command frames to the Controller every so often.
* READY frames are typically much smaller than STATS ones as their
  payload does not contain any instance related status. On CNs
  running thousands of instances, STATS payloads can be
  significantly larger than READY ones.
* Sending a STATS command does explicitly provide information
  about the Agent's readiness to process any further instance
  related commands. For example, an Agent may be busy starting an
  instance while at the same time sending a STATS command.

As a consequence SSNTP compute node Agents must use the READY and
FULL status frames, rather than STATs frames, to notify the scheduler
about their availability and capacity.

The READY status frame payload is mandatory:

```
+----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted compute |
|       |       | (0x1) |  (0x1)  |                 | node new capacity      |
+----------------------------------------------------------------------------+
```

#### FULL ####
Whenever the CN they manage runs out of capacity, SSNTP Agents
must send a FULL status frame to the Scheduler.

The Scheduler must not send any START command to an Agent whose latest status is
reported to be FULL. FULL Agents who receive such commands should reply
with an SSNTP error frame to the Scheduler.
The error code should be StartFailure (0x2)

The Scheduler may decide to resume sending START commands to a
FULL Agent after receiving the next READY status frame from it.
Any SSNTP command except for the START and CONNECT ones can be
sent to a FULL Agent.

The FULL status frame is payloadless:

```
+---------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length |
|       |       | (0x1) |  (0x2)  |       (0x0)     |
+---------------------------------------------------+
```

#### OFFLINE ####
OFFLINE is a compute node status frame sent by SSNTP Agents to let
the Scheduler know that although they're running and still
connected to the SSNTP network, they are not ready to process any
kind of SSNTP commands. Agents should reply with a SSNTP error
frame to any received frame while they are OFFLINE.

The Scheduler should forward OFFLINE status frames to the Controller
for it to immediately know about a CN not being able to process any
further commands.

SSNTP Agents in OFFLINE mode should continue sending periodic
STATS frame.

The OFFLINE status frame is payloadless:

```
+---------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length |
|       |       | (0x1) |  (0x3)  |       (0x0)     |
+---------------------------------------------------+
```

#### MAINTENANCE ###
TBD
```
+-----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload  |
|       |       | (0x1) |  (0x4)  |                 |                         |
+-----------------------------------------------------------------------------+
```

### SSNTP EVENT frames ###

Unlike STATUS frames, EVENT frames are not necessarily related to
a particular compute node's status.  They allow SSNTP entities to
notify each other about important events.

There are 6 different SSNTP EVENT frames: TenantAdded,
TenantRemoved, InstanceDeleted, ConcentratorInstanceAdded,
PublicIPAssigned and TraceReport.

#### TenantAdded ####
TenantAdded is used by CN Agents to notify Networking
Agents that the first workload for a given tenant has just started.
Networking agents need to be notified about this so that they can
forward the notification to the right CNCI (Compute Node Concentrator Instance),
i.e. the CNCI running the tenant workload.

A [TenantAdded event payload]
(https://github.com/01org/ciao/blob/master/payloads/tenantadded.go)
is a YAML formatted one containing the tenant, the agent
and the concentrator instance (CNCI) UUID, the tenant subnet,
the agent and the CNCI IPs, the subnet key and the CNCI MAC.

The Scheduler receives TenantAdded events from the CN Agent
and must forward them to the appropriate CNCI Agent.

```
+---------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted tenant |
|       |       | (0x3) |  (0x0)  |                 | information           |
+---------------------------------------------------------------------------+
```

#### TenantRemoved ####
TenantRemoved is used by CN Agents to notify Networking
Agents that the last workload for a given tenant has just
terminated. Networking agents need to be notified about
it so that they can forward it to the right CNCI (Compute
Node Concentrator Instance), i.e. the CNCI running the
tenant workload.

A [TenantRemoved event payload]
(https://github.com/01org/ciao/blob/master/payloads/tenantadded.go)
is a YAML formatted one containing the tenant, the agent
and the concentrator instance (CNCI) UUID, the tenant subnet,
the agent and the CNCI IPs, and the subnet key.

The Scheduler receives TenantRemoved events from the CN Agent
and must forward them to the appropriate CNCI Agent.

```
+---------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted tenant |
|       |       | (0x3) |  (0x1)  |                 | information           |
+---------------------------------------------------------------------------+
```

#### InstanceDeleted ####
InstanceDeleted is sent by workload agents to notify
the scheduler and the Controller that a previously running
instance has been deleted.
While the scheduler and the Controller could infer that piece
of information from the next STATS command (The deleted
instance would no longer be there) it is safer, simpler
and less error prone to explicitly send this event.

A [InstanceDeleted event payload]
(https://github.com/01org/ciao/blob/master/payloads/instancedeleted.go)
is a YAML formatted one containing the deleted instance UUID.

The Scheduler receives InstanceDeleted events from the
payload agents and must forward them to the Controller.

```
+---------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted tenant |
|       |       | (0x3) |  (0x2)  |                 | information           |
+---------------------------------------------------------------------------+
```

#### ConcentratorInstanceAdded ####
Networking node agents send this event to the Scheduler
to notify the SSNTP network that a networking concentrator
instance (CNCI) is now running on this node.
A CNCI handles the GRE tunnel concentrator for a given
tenant. Each instance started by this tenant will have a
GRE tunnel established between it and the CNCI allowing all
instances for a given tenant to be on the same private
network.

The Scheduler must forward that event to all Controllers. The Controller
needs to know about it as it will fetch the CNCI IP and the
tenant UUID from this event's payload and pass that through
the START payload when scheduling a new instance for this
tenant. A tenant instances can not be scheduled until Controller gets
a ConcentratorInstanceAdded event as instances will be
isolated as long as the CNCI for this tenant is not running.

A [ConcentratorInstanceAdded event payload]
(https://github.com/01org/ciao/blob/master/payloads/concentratorinstanceadded.go)
is a YAML formatted one containing the CNCI IP and the tenant
UUID on behalf of which the CNCI runs.

```
+---------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted CNCI   |
|       |       | (0x3) |  (0x3)  |                 | information           |
+---------------------------------------------------------------------------+
```

#### PublicIPAssigned ####
Networking concentrator instances (CNCI) send PublicIPAssigned
to the Scheduler when they successfully assigned a public IP
to a given instance.
The public IP can either come from a Controller pre-allocated pool,
or from a control network DHCP server.

The Scheduler must forward those events to the Controller.

The [PublicIPAssigned event payload]
(https://github.com/01org/ciao/blob/master/payloads/concentratorinstanceadded.go)
contains the newly assigned public IP, the instance private IP,
the instance UUID and the concentrator UUID.

```
+--------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted |
|       |       | (0x3) |  (0x4)  |                 | payload        |
+--------------------------------------------------------------------+
```

#### TraceReport ####
Any SSNTP entity can decide to send a TraceReport event in order
to let the CIAO controller know about any kind of frame traces.

It is then up to the Controller to interpret and store those traces.

The [TraveReport event payload]
(https://github.com/01org/ciao/blob/master/payloads/tracereport.go)
contains a set of frame traces.

```
+----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload |
|       |       | (0x3) |  (0x5)  |                 |                        |
+----------------------------------------------------------------------------+
```

#### NodeConnected ####
NodeConnected events are sent by the Scheduler to notify e.g. the Controllers about
a new compute or networking node being connected.
The [NodeConnected event payload]
(https://github.com/01org/ciao/blob/master/payloads/nodeconnected.go)
contains the connected node UUID and the node type (compute or networking)

```
+----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload |
|       |       | (0x3) |  (0x6)  |                 |                        |
+----------------------------------------------------------------------------+
```

#### NodeDisconnected ####
NodeDisconnected events are sent by the Scheduler to notify e.g. the Controllers about
a compute or networking node disconnection.
The [NodeDisconnected event payload]
(https://github.com/01org/ciao/blob/master/payloads/nodeconnected.go)
contains the disconnected node UUID and the node type (compute or networking)

```
+----------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted payload |
|       |       | (0x3) |  (0x7)  |                 |                        |
+----------------------------------------------------------------------------+
```

### SSNTP ERROR frames ###
SSNTP being a fully asynchronous protocol, SSNTP entities are
not expecting specific frames to be acknowledged or rejected.
Instead they must be ready to receive asynchronous error
frames notifying them about an application level error, not
a frame level one.

There are 7 different SSNTP ERROR frames:

#### InvalidFrameType ####
When a SSNTP entity receives a frame whose type it does not
support, it should send an InvalidFrameType error back
to the sender.

The [InvalidFrameType error payload]
(https://github.com/01org/ciao/blob/master/payloads/invalidframetype.go)
only contains the SSNTP frame type that the receiver could
not process:

```
+-----------------------------------------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | Source UUID | Destination UUID | YAML formatted frame |
|       |       | (0x4) |  (0x0)  |                 |             |                  | type information     |
+-----------------------------------------------------------------------------------------------------------+
```

#### StartFailure ####
The StartFailure SSNTP error frames must be sent when an
instance could not be started. For example:

* The Scheduler receives a START command from the Controller but
  all its CN Agents are busy or full. In that case the
  Scheduler must send a StartFailure error frame back to
  the Controller

* An Agent receives a START command from the Scheduler but
  it cannot start the instance. This could happen for many
  reasons:
  * Malformed START YAML payload
  * Compute node is full
  In that case the Agent must send a StartFailure error
  frame back to the Scheduler and the Scheduler must forward
  it to the Controller.

The [StartFailure YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/startfailure.go)
contains the instance UUID that failed to be started together
with an additional error string.

```
+--------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted frame |
|       |       | (0x4) |  (0x1)  |                 | error information    |
+--------------------------------------------------------------------------+
```

#### StopFailure ####
When the Controller client needs to stop a running instance on a given CN,
it sends a STOP SSNTP command to the Scheduler. The STOP command
payload contains the instance UUID and the CN Agent UUID where that
instance is running.

* If the Scheduler can no longer find the CN Agent, it must send
  a StopFailure error frame back to the Controller.

* If the CN Agent cannot stop the instance because, for example, it
  is no longer running, it must send a StopFailure error frame back
  to the Scheduler and the Scheduler must forward it to the Controller.

The [StopFailure YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/startfailure.go)
contains the instance UUID that failed to be stopped together
with an additional error string.

```
+--------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted frame |
|       |       | (0x4) |  (0x2)  |                 | error information    |
+--------------------------------------------------------------------------+
```

#### ConnectionFailure ####
Both SSNTP clients and servers can send a ConnectionFailure error
frame when the initial connection could not be completed but should
be retried. ConnectionFailure is not a fatal error but represents
a transient connection error.

The ConnectionFailure error frame is payloadless:

```
+---------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length |
|       |       | (0x4) |  (0x3)  |     (0x0)       |
+---------------------------------------------------+
```

#### DeleteFailure ####
When the Controller client wants to delete a stopped instance on a given CN,
it sends a DELETE SSNTP command to the Scheduler.

* If the Scheduler can no longer find the CN Agent, it must send
  a DeleteFailure error frame back to the Controller.

* If the CN Agent cannot delete the instance because, for example, it
  is no longer present, it must send a DeleteFailure error frame back
  to the Scheduler and the Scheduler must forward it to the Controller.

The [DeleteFailure YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/deletefailure.go)
contains the instance UUID that failed to be stopped together
with an additional error string.
```
+--------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted frame |
|       |       | (0x4) |  (0x4)  |                 | error information    |
+--------------------------------------------------------------------------+
```

#### RestartFailure ####
When the Controller client wants to restart a stopped instance on a given CN,
it sends a RESTART SSNTP command to the Scheduler.

* If the Scheduler can no longer find the CN Agent, it must send
  a RestartFailure error frame back to the Controller.

* If the CN Agent cannot restart the instance because, for example, it
  is no longer present, it must send a RestartFailure error frame back
  to the Scheduler and the Scheduler must forward it to the Controller.

The [RestartFailure YAML payload]
(https://github.com/01org/ciao/blob/master/payloads/startfailure.go)
contains the instance UUID that failed to be stopped together
with an additional error string.
```
+--------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted frame |
|       |       | (0x4) |  (0x5)  |                 | error information    |
+--------------------------------------------------------------------------+
```

#### ConnectionAborted ####
Both SSNTP clients and servers can send a ConnectionAborted error
frame when either the CONNECT command frame or the CONNECTED status
frame contain an advertised role that does not match the peer's
certificate extended key usage attribute.

Sending ConnectionAborted means that for security reasons the connection
will not be retried.

The ConnectionAborted error frame is payloadless:
```
+---------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length |
|       |       | (0x4) |  (0x6)  |     (0x0)       |
+---------------------------------------------------+
```

#### InvalidConfiguration ####
The InvalidConfiguration error is either sent by the Scheduler to report
an invalid CONFIGURE payload back to the sender, or by the clients to
which a CONFIGURE command has been forwarded to and that leads to
configuration errors on their side.
When the scheduler receives such error back from any client it should revert
back to the previous valid configuration.

The InvalidConfiguration error frame contain the invalid
[configuration data](https://github.com/01org/ciao/blob/master/payloads/configure.go) payload.
```
+------------------------------------------------------------------------+
| Major | Minor | Type  | Operand |  Payload Length | YAML formatted     |
|       |       | (0x4) |  (0x7)  |                 | configuration data |
+------------------------------------------------------------------------+
```
