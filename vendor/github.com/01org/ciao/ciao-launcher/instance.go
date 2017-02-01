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
	"path"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	storage "github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
)

type instanceData struct {
	cmdCh          chan interface{}
	instance       string
	cfg            *vmConfig
	wg             *sync.WaitGroup
	doneCh         chan struct{}
	ac             *agentClient
	ovsCh          chan<- interface{}
	instanceWg     sync.WaitGroup
	monitorCh      chan interface{}
	connectedCh    chan struct{}
	monitorCloseCh chan struct{}
	statsTimer     <-chan time.Time
	vm             virtualizer
	instanceDir    string
	shuttingDown   bool
	rcvStamp       time.Time
	st             *startTimes
	storageDriver  storage.BlockDriver
}

type insStartCmd struct {
	userData []byte
	metaData []byte
	frame    *ssntp.Frame
	cfg      *vmConfig
	rcvStamp time.Time
}
type insRestartCmd struct{}
type insDeleteCmd struct {
	suicide bool
	running ovsRunningState
}
type insStopCmd struct{}
type insMonitorCmd struct{}

type insAttachVolumeCmd struct {
	volumeUUID string
}
type insDetachVolumeCmd struct {
	volumeUUID string
}

/*
This functions asks the server loop to kill the instance.  An instance
needs to request that the server loop kill it if Start fails completly.
As the serverLoop does not wait for the start command to complete, we wouldn't
want to do this, as it would mean all start commands execute in serial,
the serverLoop cannot detect this situation.  Thus the instance loop needs
to request it's own death.

The server loop is the only go routine that can kill the instance.  If the
instance kills itself, the serverLoop would lockup if a command came in for
that instance while it was shutting down.  The instance go routine cannot
send a command to the serverLoop directly as this could lead to deadlock.
So we must spawn a separate go routine to do this.  We also need to handle
the case that this go routine blocks for ever if the serverLoop is quit
by CTRL-C.  That's why we select on doneCh as well.  In this case,
the command will never be written to the serverLoop, our go routine will
exit, the instance will exit and then finally the overseer will quit.

There's always the possibility new commands will be received for the
instance while it is waiting to be killed.  We'll just fail those.
*/

func killMe(instance string, doneCh chan struct{}, ac *agentClient, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		cmd := &cmdWrapper{instance, &insDeleteCmd{suicide: true}}
		select {
		case ac.cmdCh <- cmd:
		case <-doneCh:
		}
		wg.Done()
	}()
}

func (id *instanceData) startCommand(cmd *insStartCmd) {
	glog.Info("Found start command")
	if id.monitorCh != nil {
		startErr := &startError{nil, payloads.AlreadyRunning}
		glog.Errorf("Unable to start instance[%s]", string(startErr.code))
		startErr.send(id.ac.conn, id.instance)
		return
	}
	st, startErr := processStart(cmd, id.instanceDir, id.vm, id.ac.conn)
	if startErr != nil {
		glog.Errorf("Unable to start instance[%s]: %v", string(startErr.code), startErr.err)
		startErr.send(id.ac.conn, id.instance)

		if startErr.code == payloads.LaunchFailure {
			id.ovsCh <- &ovsStateChange{id.instance, ovsStopped}
		} else if startErr.code != payloads.InstanceExists {
			glog.Warningf("Unable to create VM instance: %s.  Killing it", id.instance)
			killMe(id.instance, id.doneCh, id.ac, &id.instanceWg)
			id.shuttingDown = true
		}
		return
	}
	id.st = st

	id.connectedCh = make(chan struct{})
	id.monitorCloseCh = make(chan struct{})
	id.monitorCh = id.vm.monitorVM(id.monitorCloseCh, id.connectedCh, &id.instanceWg, false)
	id.ovsCh <- &ovsStatusCmd{}
	if cmd.frame != nil && cmd.frame.PathTrace() {
		id.ovsCh <- &ovsTraceFrame{cmd.frame}
	}
}

func (id *instanceData) restartCommand(cmd *insRestartCmd) {
	glog.Info("Found restart command")

	if id.shuttingDown {
		restartErr := &restartError{nil, payloads.RestartNoInstance}
		glog.Errorf("Unable to restart instance[%s]", string(restartErr.code))
		restartErr.send(id.ac.conn, id.instance)
		return
	}

	if id.monitorCh != nil {
		restartErr := &restartError{nil, payloads.RestartAlreadyRunning}
		glog.Errorf("Unable to restart instance[%s]", string(restartErr.code))
		restartErr.send(id.ac.conn, id.instance)
		return
	}

	restartErr := processRestart(id.instanceDir, id.vm, id.ac.conn, id.cfg)

	if restartErr != nil {
		glog.Errorf("Unable to restart instance[%s]: %v", string(restartErr.code),
			restartErr.err)
		restartErr.send(id.ac.conn, id.instance)
		return
	}

	id.connectedCh = make(chan struct{})
	id.monitorCloseCh = make(chan struct{})
	id.monitorCh = id.vm.monitorVM(id.monitorCloseCh, id.connectedCh, &id.instanceWg, false)
}

func (id *instanceData) monitorCommand(cmd *insMonitorCmd) {
	id.connectedCh = make(chan struct{})
	id.monitorCloseCh = make(chan struct{})
	id.monitorCh = id.vm.monitorVM(id.monitorCloseCh, id.connectedCh, &id.instanceWg, true)
}

func (id *instanceData) stopCommand(cmd *insStopCmd) {
	if id.shuttingDown {
		stopErr := &stopError{nil, payloads.StopNoInstance}
		glog.Errorf("Unable to stop instance[%s]", string(stopErr.code))
		stopErr.send(id.ac.conn, id.instance)
		return
	}

	if id.monitorCh == nil {
		stopErr := &stopError{nil, payloads.StopAlreadyStopped}
		glog.Errorf("Unable to stop instance[%s]", string(stopErr.code))
		stopErr.send(id.ac.conn, id.instance)
		return
	}
	glog.Infof("Powerdown %s", id.instance)
	id.monitorCh <- virtualizerStopCmd{}
}

func (id *instanceData) sendInstanceDeletedEvent() {
	var event payloads.EventInstanceDeleted

	event.InstanceDeleted.InstanceUUID = id.instance

	payload, err := yaml.Marshal(&event)
	if err != nil {
		glog.Errorf("Unable to Marshall STATS %v", err)
		return
	}

	_, err = id.ac.conn.SendEvent(ssntp.InstanceDeleted, payload)
	if err != nil {
		glog.Errorf("Failed to send event command %v", err)
		return
	}
}

func (id *instanceData) deleteCommand(cmd *insDeleteCmd) bool {
	if id.shuttingDown && !cmd.suicide {
		deleteErr := &deleteError{nil, payloads.DeleteNoInstance}
		glog.Errorf("Unable to delete instance[%s]", string(deleteErr.code))
		deleteErr.send(id.ac.conn, id.instance)
		return false
	}

	if id.monitorCh != nil {
		glog.Infof("Powerdown %s before deleting", id.instance)
		id.monitorCh <- virtualizerStopCmd{}
		select {
		case <-id.monitorCloseCh:
		case <-time.After(time.Second * 10):
			glog.Warningf("Timeout (10s) waiting for virtualizer to terminate")
		}
		id.vm.lostVM()
	}

	_ = processDelete(id.vm, id.instanceDir, id.ac.conn, cmd.running)

	id.unmapVolumes()

	if !cmd.suicide {
		id.sendInstanceDeletedEvent()
		id.ovsCh <- &ovsStatusCmd{}
	}
	return true
}

func (id *instanceData) attachVolumeCommand(cmd *insAttachVolumeCmd) {
	if id.shuttingDown {
		attachErr := &attachVolumeError{nil, payloads.AttachVolumeInstanceFailure}
		glog.Errorf("Unable to attach instance[%s]", string(attachErr.code))
		attachErr.send(id.ac.conn, id.instance, cmd.volumeUUID)
		return
	}

	attachErr := processAttachVolume(id.storageDriver, id.monitorCh, id.cfg, id.instance, id.instanceDir,
		cmd.volumeUUID, id.ac.conn)
	if attachErr != nil {
		attachErr.send(id.ac.conn, id.instance, cmd.volumeUUID)
		return
	}
	d, m, c := id.vm.stats()
	id.ovsCh <- &ovsStatsUpdateCmd{id.instance, m, d, c, id.getVolumes()}

	glog.Infof("Volume %s attached to instance %s", cmd.volumeUUID, id.instance)
}

func (id *instanceData) detachVolumeCommand(cmd *insDetachVolumeCmd) {
	if id.shuttingDown {
		detachErr := &detachVolumeError{nil, payloads.DetachVolumeInstanceFailure}
		glog.Errorf("Unable to detach instance[%s]", string(detachErr.code))
		detachErr.send(id.ac.conn, id.instance, cmd.volumeUUID)
		return
	}

	detachErr := processDetachVolume(id.storageDriver, id.monitorCh, id.cfg, id.instance, id.instanceDir,
		cmd.volumeUUID, id.ac.conn)
	if detachErr != nil {
		detachErr.send(id.ac.conn, cmd.volumeUUID, id.instance)
		return
	}
	d, m, c := id.vm.stats()
	id.ovsCh <- &ovsStatsUpdateCmd{id.instance, m, d, c, id.getVolumes()}

	glog.Infof("Volume %s detched from instance %s", cmd.volumeUUID, id.instance)
}

func (id *instanceData) logStartTrace() {
	if id.st == nil {
		return
	}

	runningStamp := time.Now()
	glog.Info("================ START TRACE ============")
	glog.Infof("Total time to start instance: %d ms", (runningStamp.Sub(id.rcvStamp))/time.Millisecond)
	glog.Infof("Launcher routing time: %d ms", (id.st.startStamp.Sub(id.rcvStamp))/time.Millisecond)
	glog.Infof("Creating time: %d ms", (id.st.runStamp.Sub(id.st.startStamp))/time.Millisecond)
	glog.Infof("Time to running: %d ms", (runningStamp.Sub(id.st.startStamp))/time.Millisecond)
	glog.Infof("Running detection time: %d ms", (runningStamp.Sub(id.st.runStamp))/time.Millisecond)
	glog.Info("")
	glog.Info("Detailed creation times")
	glog.Info("-----------------------")
	glog.Infof("Backing Image Check: %d", id.st.backingImageCheck.Sub(id.st.startStamp)/time.Millisecond)
	glog.Infof("Network creation: %d", id.st.networkStamp.Sub(id.st.backingImageCheck)/time.Millisecond)
	glog.Infof("VM/Container creation: %d", id.st.creationStamp.Sub(id.st.networkStamp)/time.Millisecond)
	glog.Infof("Time to start: %d", id.st.runStamp.Sub(id.st.creationStamp)/time.Millisecond)
	glog.Info("=========================================")
}

func (id *instanceData) instanceCommand(cmd interface{}) bool {
	select {
	case <-id.doneCh:
		return false
	default:
	}

	switch cmd := cmd.(type) {
	case *insStartCmd:
		id.rcvStamp = cmd.rcvStamp
		id.startCommand(cmd)
	case *insRestartCmd:
		id.restartCommand(cmd)
	case *insMonitorCmd:
		id.monitorCommand(cmd)
	case *insStopCmd:
		id.stopCommand(cmd)
	case *insAttachVolumeCmd:
		id.attachVolumeCommand(cmd)
	case *insDetachVolumeCmd:
		id.detachVolumeCommand(cmd)
	case *insDeleteCmd:
		if id.deleteCommand(cmd) {
			return false
		}
	default:
		glog.Warning("Unknown command")
	}

	return true
}

func (id *instanceData) getVolumes() []string {
	volumes := make([]string, 0, len(id.cfg.Volumes))
	for _, v := range id.cfg.Volumes {
		volumes = append(volumes, v.UUID)
	}
	return volumes
}

func (id *instanceData) unmapVolumes() {
	glog.Infof("Unmapping volumes for %s", id.instance)

	for _, v := range id.cfg.Volumes {

		// UnmapVolumeFromNode might fail if it's mapped to multiple
		// instances on the same node.  We don't treat this as an
		// error for now.

		if err := id.storageDriver.UnmapVolumeFromNode(v.UUID); err == nil {
			glog.Infof("Unmapping volume %s", v.UUID)
		}
	}
}

func (id *instanceData) instanceLoop() {

	id.vm.init(id.cfg, id.instanceDir)

	d, m, c := id.vm.stats()
	id.ovsCh <- &ovsStatsUpdateCmd{id.instance, m, d, c, id.getVolumes()}

DONE:
	for {
		select {
		case <-id.doneCh:
			break DONE
		case <-id.statsTimer:
			d, m, c := id.vm.stats()
			id.ovsCh <- &ovsStatsUpdateCmd{id.instance, m, d, c, id.getVolumes()}
			id.statsTimer = time.After(time.Second * resourcePeriod)
		case cmd := <-id.cmdCh:
			if !id.instanceCommand(cmd) {
				break DONE
			}
		case <-id.monitorCloseCh:
			// Means we've lost VM for now
			id.vm.lostVM()
			d, m, c := id.vm.stats()
			id.ovsCh <- &ovsStatsUpdateCmd{id.instance, m, d, c, id.getVolumes()}

			glog.Infof("Lost VM instance: %s", id.instance)
			id.monitorCloseCh = nil
			id.connectedCh = nil
			close(id.monitorCh)
			id.monitorCh = nil
			id.statsTimer = nil
			id.ovsCh <- &ovsStateChange{id.instance, ovsStopped}
			id.st = nil
			id.unmapVolumes()
		case <-id.connectedCh:
			id.logStartTrace()
			id.connectedCh = nil
			id.vm.connected()
			id.ovsCh <- &ovsStateChange{id.instance, ovsRunning}
			d, m, c := id.vm.stats()
			id.ovsCh <- &ovsStatsUpdateCmd{id.instance, m, d, c, id.getVolumes()}
			id.statsTimer = time.After(time.Second * resourcePeriod)
		}
	}

	if id.monitorCh != nil {
		close(id.monitorCh)
	}

	glog.Infof("Instance goroutine %s waiting for monitor to exit", id.instance)
	id.instanceWg.Wait()
	glog.Infof("Instance goroutine %s exitted", id.instance)
	id.wg.Done()
}

func startInstanceWithVM(instance string, cfg *vmConfig, wg *sync.WaitGroup, doneCh chan struct{},
	ac *agentClient, ovsCh chan<- interface{}, vm virtualizer, storageDriver storage.BlockDriver,
	instancesDir string) chan<- interface{} {
	id := &instanceData{
		cmdCh:         make(chan interface{}),
		instance:      instance,
		cfg:           cfg,
		wg:            wg,
		doneCh:        doneCh,
		ac:            ac,
		ovsCh:         ovsCh,
		vm:            vm,
		instanceDir:   path.Join(instancesDir, instance),
		storageDriver: storageDriver,
	}

	wg.Add(1)
	go id.instanceLoop()
	return id.cmdCh
}

func startInstance(instance string, cfg *vmConfig, wg *sync.WaitGroup, doneCh chan struct{},
	ac *agentClient, ovsCh chan<- interface{}) chan<- interface{} {

	storageDriver := storage.CephDriver{
		ID: cephID,
	}

	var vm virtualizer
	if simulate == true {
		vm = &simulation{}
	} else if cfg.Container {
		vm = &docker{storageDriver: storageDriver}
	} else {
		vm = &qemuV{}
	}
	return startInstanceWithVM(instance, cfg, wg, doneCh, ac, ovsCh, vm, storageDriver,
		instancesDir)
}
