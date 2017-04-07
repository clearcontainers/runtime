/*
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
*/

// ciao-launcher is an ssntp agent that runs on compute and network nodes.
// Its primary purpose is to launch and manage containers and virtual
// machines.  For more information on installing and running ciao-launcher
// see https://github.com/01org/ciao/blob/master/ciao-launcher/README.md
// for more information.
//
// Introduction
//
// ciao-launcher tries to take advantage of Go's concurrency support as much as
// possible.  The intention here is that most of the work involved in launching
// and manipulating VMs is mostly self contained and IO bound and so should in
// theory lend itself well to a concurrent design.  As a consequence, ciao-launcher
// is highly concurrent and performant.  The concurrent nature
// of ciao-launcher can make it a little difficult to understand for new comers,
// so here are a few notes on its design.  ciao-launcher can be thought of as a
// collection of distinct go routines.  These notes explain what these go
// routines are for and how they communicate.
//
// Main
//
// Main is the go routine that starts when ciao-launcher is itself launched.  The code
// for this is in main.go.  It parses the command line parameters, initialises
// networking, ensures that no other instance of ciao-launcher are running and then
// starts the server go routine.  Having done all this, the main go routine waits
// for a signal, e.g., SIGTERM, from the OS to quit.  When this signal is retrieved
// it instructs all child go routines to quit and waits for their exit.  Note that
// it only waits for 1 second.  If all child go routines have failed to exit in 1
// second, ciao-launcher panics.  The panic is useful as it prints the stack trace of
// all the running go routines, so you can see which ones are blocked.
//
// The Server go routine
//
// Manages the connection to the SSNTP server and pre-processes all commands
// received from this server. The code for this go routine is also in main.go, at
// least for the time being.  ciao-Launcher
// establishes a connection to the ssntp server via the ssntp.Dial command.  This
// creates a separate go routine, managed by the ssntp library.  Any subsequent
// ssntp events that occur are handled by the ciao-launcher function CommandNotify.
// CommandNotify is called in the context of the ssntp go routine.  To avoid
// blocking this go routine, ciao-launcher parses the YAML associated with the command,
// sends a newly created internal command down a channel to the server go routine
// and returns.  The command is then processed further in the server go routine.
//
// Most commands are operations on instances, e.g., create a new instance, delete
// an instance, and are ultimately processed by a go routine dedicated to the
// particular instance to which they pertain.  These instance go routines are
// managed by another go routine called the overseer which will be discussed in
// more detail below.  Before the server go routine can forward a command to the
// appropriate go routine it needs to ask the overseer for a channel which can be
// used to communicate with the relevant instance go routine.  This is done by
// sending an ovsAddCmd or an ovsGetCmd to the overseer via the overseer channel,
// ovsCh.  ovsAddCmd is used when starting a new instance.  ovsGetCmd is used to
// process all other commands.
//
// The overseer go routine is started by the server go routine.  When the server
// go routine is asked to exit by the main go routine, i.e., main go routine
// closes the doneCh channel, the server go routine closes the channel it uses
// to communicate with the overseer.  This instructs the overseer to close,
// which it does after all the instance go routines it manages have in turn
// exited.  The server go routine waits for the overseer to exit before
// terminating.
//
// The Overseer
//
// The overseer is a go routine that serves three main purposes.
//
//  1.  It manages instance go routines that themselves manage individual vms.
//  2.  It collects statistics about the node and the VMs it hosts and
//      tranmits these periodically to the ssntp server via the STATS and
//      STATUS commands.
//  3.  It Rediscovers and reconnects to existing instances when ciao-launcher is started.
// Overseer launches new instances via the startInstance function from instance.go.
// This function starts a new go routine for that instance and returns a channel
// through which commands can be sent to the instance.  The overseer itself does
// not send commands down this channel.  It cannot as this would lead to deadlock.
// Instead it makes this channel available to the server go routine when it is
// requested via the ovsAddCmd or the ovsGetCmd commands.
//
// The overseer passes each instance go routine a reference to the a single
// channel, childDoneCh.  This channel is closed when the overseer starts shutting
// down.  Closing this channel serves as a broadcast notification to each instance
// go routine, indicating that they need to shutdown.  The overseer waits until all
// instance go routines have shut down before exiting.  This is achieved via a wait
// group called chilWg.
//
// The overseer maintains a map of information about each instance called
// instances.  The map is indexed by the uuid of instances.  It contains
// information about the instances, namely their running state and their resource
// usage.  This information is used when sending STATs and STATUS commands.
//
// Information about the instances ultimately comes from the instance go routines.
// However, these go routines cannot access the overseer's instance map directly.
// To update it they send commands down a channel, which the overseer passes to
// startInstance, for example, ovsStatsUpdateCmd or ovsStateChange.  The overseer
// processes these commands in the processCommand function.
//
// The Instance Go routines
//
// ciao-launcher maintains one go routine per instance it manages.  These go routines
// exist regardless of the state of the underlying instance, i.e., there is
// generally one go routine running per instance, regardless of whether that
// instance is pending, exited or running.
//
// The instance go routines serve 3 main purposes:
//
//  1. They accept and process commands from the server go routine down their
//     command channel.  These commands typically come from the ssntp server,
//     although there are some occasions where the commands originate from inside
//     ciao-launcher itself.
//  2. They monitor the running state of VMs.
//  3. They manage the collection of instance statistics, which they report
//     to the overseer.
//
// The nice thing about this design is that almost all instance related work can be
// performed in parallel.  Stats can be computed for one instance at the same time
// as a separate instance is being powered down.  ciao-launcher can process any number
// of commands to start new instances in parallel.  There is no locking required
// apart from a synchronised access to the overseer map made by the server go
// routine when the command is first received and a small check related to the
// image from which the instance will be launched, in the case where instances are
// being started.  An additional synchronisation point is required for docker
// instances to ensure that the relevant docker network has been created
// before the container.
//
// Note that although commands submitted to different instances are in executed in
// parallel, the instance go routines serialise commands issue against a single
// instance.  This is necessary to avoid instance corruption.
//
// Right now the command channels that the server go routine uses to send command
// to instances are not buffered.  This might need to change as currently it could
// be possible for a SSNTP server to kill ciao-launcher's parallelism by repetively
// sending commands to the same instance over and over again.
//
// The code for the instance go routines is in instance.go.  However, the code that
// executes most of the commands has been placed in separate files named after the
// commands themselves, e.g., start_instance.go, delete_instance.go.  It should be noted
// that the code in these files runs in the context of an instance go routine.
// Finally, some of the code used to process instance commands is in payloads.go.
// This is for legacy reasons and in the future this file will probably go away and
// its contents will be redistributed.  It should not be assumed as of the time of
// writing that all the code in payloads.go runs in an instance go routine.
// payloads.go needs cleaning up (https://github.com/01org/ciao/issues/10).
//
// The virtualizer
//
// The instance go routines need to talk to qemu and docker to manage their VMs
// and containers.  However, they do not do so directly.  Rather they do so via
// a virtualizer interface.
//
// The virtualizer interface is designed to isolate ciao-launcher, and in particular,
// functions that run in the instance go routine, from the underlying virtualisation
// technologies used to launch and manage VMs and containers.
// All the methods on the virtualizer interface will be called serially by the instance
// go routine.  Therefore there is no need to synchronised data between the virtualizers
// methods.
//
// qemu.go contains functions to start and stop a VM, to create a qcow2 image, to
// collect statistics about the instance running in the hypervisor, such as its
// memory and cpu usage, and to monitor the instance, i.e., to determine whether
// the VM is actually running or not.
//
// docker.go contains methods to manage docker containers.
//
// For more information about the virtualizer API, please see the comments
// in https://github.com/01org/ciao/blob/master/ciao-launcher/virtualizer.go
//
package main
