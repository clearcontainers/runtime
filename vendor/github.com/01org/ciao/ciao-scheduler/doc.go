//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

/*

Ciao scheduler is the SSNTP server that runs in the control plane to
broker a ciao based cloud.

In addressing the broader problem of dispatching a workload for a
user, ciao splits the problem:

     +--------------+  +------------+
     |  ciao-webui  |  |  ciao-cli  |
     +--------------+  +------------+
                 |        |
              +--------------+
              |  controller  |
              +--------------+
                     |
              +--------------+
              |  scheduler   |
              +--------------+
                |    |    |
       +----------+  |  +----------+
       | launcher |  |  | launcher |
       +----------+  |  +----------+
                     |
                +----------+
                | launcher |
                +----------+

At the top level, ciao-webui
(https://github.com/01org/ciao-webui), ciao-cli
(https://github.com/01org/ciao/tree/master/ciao-cli) and ciao-controller
(https://github.com/01org/ciao/tree/master/ciao-controller) are
responsible for interacting with the user.  Ciao-controller enforces
policy, checking that the users' actions are allowed.  For allowed
actions, ciao-controller sends SSNTP command frames down to
ciao-scheduler.

At the lowest level, ciao-launcher
(https://github.com/01org/ciao/tree/master/ciao-launcher) is running
on each compute node.  It connects to the ciao-scheduler and sends node
level statistics regularly so that the scheduler always knows the current
resource state of the cluster.  The launchers also send up statistics
for each running workload, but scheduler does not pay attention to these
and merely forwards them up the stack to ciao-controller.

This layered design leaves a very lean, scalable scheduler in the middle,
where ciao-scheduler's primary task is to take a new workload description
and find a fit for it in the cluster.  Performing this task entails a
search across only in-memory, known up-to-date data, and is done VERY VERY
QUICKLY.

A Fit vs Best Fit

Ciao-scheduler explicitly does not attempt to find the best fit for
a workload.

We bias towards speed of dispatching and simplicity of implementation
over absolute optimality.

Aiming for optimality puts us on a slippery slope which at the extreme
could mean locking all state in the entire cluster, collecting and
analyzing the locked state, making a decision and then unlocking the
state.  This will have bad performance, both in terms of latency to
start an individual workload and for overall throughput when launching
many workloads.

We also assume that while a cloud administrator surely has cost
constraints, they are unlikely to always run a general compute cloud at
the extreme edge of capacity.  If they are providing a service for users,
their users will expect a reasonable response time for new work orders
and that in turn implies there is indeed capacity for new work.

Finding the best fit is more important if resources are highly constrained
and you want to make an attempt to give future workloads (whose specific
nature is yet unknown) a better chance of succeeding.  Again though,
attempting to address future unknowns adds complexity to the code,
incurs latencies and hinders scalability.

Today a compute node that has no remaining capacity (modulo a buffer
amount for the launcher and host OS's stability) will report that
it is full and the scheduler will not dispatch work to that node.
As a last resort, ciao-scheduler will return a "cloud full" status to
ciao-controller if no compute nodes have capacity to do work.

Data Structures and Scale

In the initial implementation, the scheduling choice
focuses primarily on RAM, disk and CPU availability (see
the "Resource" enumeration type in the START payload at
https://github.com/01org/ciao/blob/master/payloads/start.go for more
details) on compute nodes relative to the requested workload start.
This list of tracked resource types will grow over time to encompass
many more compute node and workload characteristics.  We don't expect
that to significantly impact the time needed to make a scheduling choice.
We have designed throughout ciao to scale.

Our goal is to make scheduling choices in the order of microseconds.
While we haven't yet tested on extremely large clusters, conceptually one
should expect that searching an in-memory data structure containing many
thousands of nodes' resource data should not take more than milliseconds.
Even if each node is a structure of a thousand unique resource statistics.
And even if the top structure is only a simple linked list.  Walking a
list of thousands of elements and doing thousands of string compares
for each element of the list is not a deeply computationally complex act.

For typical clouds today and in the foreseeable future, we expect our
implementation will scale.

Robustness and Update-ability

The nature of the launcher agents checking in with scheduler to update
their node statistics and request work means that the scheduler always
has an up-to-date, in-memory representation of the cluster.  No explicit
persistence of this data is required.  The scheduler can crash and
restart, or be stopped and updated and restarted, and launcher agents
will simply reconnect and keep on continually updating the scheduler of
any changes in their node statistics.

Fairness

Ciao-scheduler currently implements an extremely trivial algorithm to
prefer not using the most-recently-used compute node.  This is inexpensive
and leads to sufficient spread of new workloads across a cluster.

*/
package main
