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

package main

import (
	"errors"
	"sync"
)

type virtualizerStopCmd struct{}
type virtualizerAttachCmd struct {
	responseCh chan error
	volumeUUID string
	device     string
}
type virtualizerDetachCmd struct {
	responseCh chan error
	volumeUUID string
}

var errImageNotFound = errors.New("Image Not Found")

//BUG(markus): These methods need to be cancellable

// The virtualizer interface is designed to isolate launcher, and in particular,
// functions that run in the instance go routine, from the underlying virtualisation
// technologies used to launch and manage VMs and containers.
// All the methods on the virtualizer interface will be called serially by the instance
// go routine.  Therefore there is no need to synchronised data between the virtualizers
// methods.
type virtualizer interface {
	// Initialise the virtualizer.  cfg contains the configuration information from
	// the START payload that originally created the instance.  InstanceDir is the
	// directory launcher has assigned to this instance.  The virtualizer is free
	// to store any data it likes in this directory.
	init(cfg *vmConfig, instanceDir string)

	// Ensure that an image is available to this instance.  This may involve downloading
	// the image.
	ensureBackingImage() error

	// Creates the rootfs, and or any supporting images, for a new instance
	// bridge: name of the bridge if any.  Needed for docker containers
	// userData: cloudinit userdata payload
	// metaData: cloudinit metaData payload
	createImage(bridge string, userData, metaData []byte) error

	// Deletes any state related to the instance that is not stored in the
	// instance directory.  State stored in the instance directory will be automatically,
	// deleted by the instance go routine.
	deleteImage() error

	// Boots a VM.  This method is called by START
	startVM(vnicName, ipAddress, cephID string) error

	//BUG(markus): Need to use context rather than the monitor channel to
	//detect when we need to quit.

	// Monitors a newly started VM.  It is intended to start one of more
	// go routines to monitor the instance and return immediately with a
	// channel that the instance go routine can use to communicate with the
	// monitored instance.
	//
	// closedCh: should be closed by this method, or a go routine that it spawns,
	// when it is determined that the VM or container that is being monitored is
	// not running.
	// connectedCh: Should be closed by this method, or a go routine that it spawns,
	// when it is determined that the VM or container that is being monitored is
	// running.
	// wg: wg.Add should be called before any go routines started by this method
	// are launched.  wg.Done should be called by these go routines before they
	// exit.  The instance go routine will use this wg to wait until all go routines
	// launched by this method have closed down before it itself exits.
	// boot: indicates whether monitorVM has been called during launcher startup,
	// in which case it's true.  I'm not sure this is needed.  It might get removed
	// shortly.
	//
	// Returns a channel.  The instance go routine uses this channel for two purposes:
	// 1. It sends commands down the channel, e.g., stop VM.
	// 2. It closes the channel when it is itself asked to shutdown.  When the channel is
	//    closed, any go routines returned by monitor vm should shutdown.
	monitorVM(closedCh chan struct{}, connectedCh chan struct{},
		wg *sync.WaitGroup, boot bool) chan interface{}

	// Returns current statistics for the instance.
	// disk: Size of the VM/container rootfs in GB or -1 if not known.
	// memory: Amount of memory used by the VM or container process, in MB
	// cpu: Normalized CPU time of VM or container process
	stats() (disk, memory, cpu int)

	// connected is called by the instance go routine to inform the virtualizer that
	// the VM is running.  The virtualizer can used this notification to perform some
	// bookkeeping, for example determine the pid of the underlying process.  It may
	// seem slightly odd that this function exists.  After all, it's a goroutine
	// spawned by the monitorVM function that initially informs the instance go
	// routine that the VM is connected. The problem is that all virtualizer methods
	// need to called by the instance go routine.  If the virtualizer were to modify
	// it's own state directly from a go routine spawned by monitorVM, mutexes would
	// be needed.  Perhaps this would be a better design.  However, for the time being
	// connected exists and is called.
	connected()

	// Similar to connected.  This function is called by the instance go routine when
	// it detects that the VM or container has stopped running.  As with connected, it
	// is a go routine spawned by monitorVM that originally detects that the VM has gone
	// down and signals the instance go routine of this fact by closing the closedCh.
	// The instance go routine then calls lostVM so that the virtualizer can update
	// its internal state.
	lostVM()
}
