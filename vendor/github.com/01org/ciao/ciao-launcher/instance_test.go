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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	storage "github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

var testInstancesDir string

var standardCfg = vmConfig{
	Cpus:       2,
	Mem:        370,
	Disk:       8000,
	Instance:   "testInstance",
	Image:      "testImage",
	Legacy:     true,
	VnicMAC:    "02:00:e6:f5:af:f9",
	VnicIP:     "192.168.8.2",
	ConcIP:     "192.168.42.21",
	SubnetIP:   "192.168.8.0/21",
	TenantUUID: "67d86208-000-4465-9018-fe14087d415f",
	ConcUUID:   "67d86208-b46c-4465-0000-fe14087d415f",
	VnicUUID:   "67d86208-b46c-0000-9018-fe14087d415f",
}

// instanceTestState implements virtualizer and serverConn
type instanceTestState struct {
	t               *testing.T
	instance        string
	statsArray      [3]int
	sf              payloads.ErrorStopFailure
	stf             payloads.ErrorStartFailure
	df              payloads.ErrorDeleteFailure
	rf              payloads.ErrorRestartFailure
	avf             payloads.ErrorAttachVolumeFailure
	dvf             payloads.ErrorDetachVolumeFailure
	connect         bool
	monitorCh       chan interface{}
	errorCh         chan struct{}
	monitorClosedCh chan struct{}
	failStartVM     bool
	ac              *agentClient
}

func (v *instanceTestState) init(cfg *vmConfig, instanceDir string) {

}

func (v *instanceTestState) checkBackingImage() error {
	return nil
}

func (v *instanceTestState) downloadBackingImage() error {
	return nil
}

func (v *instanceTestState) createImage(bridge string, userData, metaData []byte) error {
	return nil
}

func (v *instanceTestState) deleteImage() error {
	return nil
}

func (v *instanceTestState) startVM(vnicName, ipAddress, cephID string) error {
	if v.failStartVM {
		return fmt.Errorf("Failed to start VM")
	}
	return nil
}

func (v *instanceTestState) monitorVM(closedCh chan struct{}, connectedCh chan struct{},
	wg *sync.WaitGroup, boot bool) chan interface{} {

	// Need to be careful here not to modify any state inside v before
	// we've closed the channel.

	v.monitorClosedCh = closedCh

	monitorCh := make(chan interface{})
	v.monitorCh = monitorCh
	if v.connect {
		close(connectedCh)
	}
	return monitorCh
}

func (v *instanceTestState) stats() (disk, memory, cpu int) {
	return v.statsArray[0], v.statsArray[1], v.statsArray[2]
}

func (v *instanceTestState) connected() {

}

func (v *instanceTestState) lostVM() {
}

func (v *instanceTestState) SendError(error ssntp.Error, payload []byte) (int, error) {
	switch error {
	case ssntp.StopFailure:
		err := yaml.Unmarshal(payload, &v.sf)
		if err != nil {
			v.t.Fatalf("Failed to unmarshall stop error %v", err)
		}
	case ssntp.StartFailure:
		err := yaml.Unmarshal(payload, &v.stf)
		if err != nil {
			v.t.Fatalf("Failed to unmarshall start error %v", err)
		}
	case ssntp.DeleteFailure:
		err := yaml.Unmarshal(payload, &v.df)
		if err != nil {
			v.t.Fatalf("Failed to unmarshall delete error %v", err)
		}
	case ssntp.RestartFailure:
		err := yaml.Unmarshal(payload, &v.rf)
		if err != nil {
			v.t.Fatalf("Failed to unmarshall restart error %v", err)
		}
	case ssntp.AttachVolumeFailure:
		err := yaml.Unmarshal(payload, &v.avf)
		if err != nil {
			v.t.Fatalf("Failed to unmarshall attach volume error %v", err)
		}
	case ssntp.DetachVolumeFailure:
		err := yaml.Unmarshal(payload, &v.dvf)
		if err != nil {
			v.t.Fatalf("Failed to unmarshall detach volume error %v", err)
		}
	}

	if v.errorCh != nil {
		close(v.errorCh)
	}

	return 0, nil
}

func (v *instanceTestState) SendEvent(event ssntp.Event, payload []byte) (int, error) {
	return 0, nil
}

func (v *instanceTestState) Dial(config *ssntp.Config, ntf ssntp.ClientNotifier) error {
	return nil
}

func (v *instanceTestState) SendStatus(status ssntp.Status, payload []byte) (int, error) {
	return 0, nil
}

func (v *instanceTestState) SendCommand(cmd ssntp.Command, payload []byte) (int, error) {
	return 0, nil
}

func (v *instanceTestState) Role() ssntp.Role {
	return ssntp.AGENT | ssntp.NETAGENT
}

func (v *instanceTestState) UUID() string {
	return ""
}

func (v *instanceTestState) Close() {

}

func (v *instanceTestState) isConnected() bool {
	return true
}

func (v *instanceTestState) setStatus(status bool) {

}

func (v *instanceTestState) cleanUpInstance() {
	_ = os.RemoveAll(path.Join(testInstancesDir, v.instance))
}

func (v *instanceTestState) verifyStatsUpdate(t *testing.T, cmd interface{}) {
	stats := cmd.(*ovsStatsUpdateCmd)
	if stats.diskUsageMB != v.statsArray[0] || stats.memoryUsageMB != v.statsArray[1] ||
		stats.CPUUsage != v.statsArray[2] || stats.instance != v.instance {
		t.Fatal("Incorrect statistics received")
	}
}

func (v *instanceTestState) getStatsUpdate(t *testing.T, ovsCh <-chan interface{}) *ovsStatsUpdateCmd {
	var cmd interface{}
	select {
	case cmd = <-ovsCh:
	case <-time.After(time.Second):
		t.Error("Timed out waiting for ovsStatsUpdateCmd")
		return nil
	}
	stats, ok := cmd.(*ovsStatsUpdateCmd)
	if !ok {
		t.Error("Unexpected Command received on ovsCh")
	}
	return stats
}

func (v *instanceTestState) expectStatsUpdateWithVolumes(t *testing.T,
	ovsCh <-chan interface{}, volumes []string) bool {

	stats := v.getStatsUpdate(t, ovsCh)
	if stats == nil {
		return false
	}

	if len(volumes) != len(stats.volumes) {
		t.Errorf("Unxpected number of volumes.  Expected %d found %d",
			len(volumes), len(stats.volumes))
	}

	found := 0
	for _, vol := range volumes {
		for _, vol2 := range stats.volumes {
			if vol2 == vol {
				found++
				break
			}
		}
	}

	if found < len(volumes) {
		t.Errorf("Missing volumes.  Expected %d found %d", len(volumes), found)
		return false
	}

	return true
}

func (v *instanceTestState) expectStatsUpdate(t *testing.T, ovsCh <-chan interface{}) bool {
	stats := v.getStatsUpdate(t, ovsCh)
	if stats == nil {
		return false
	}
	if stats.diskUsageMB != v.statsArray[0] || stats.memoryUsageMB != v.statsArray[1] ||
		stats.CPUUsage != v.statsArray[2] || stats.instance != v.instance {
		t.Error("Incorrect statistics received")
		return false
	}
	return true
}

func (v *instanceTestState) deleteInstance(t *testing.T, ovsCh chan interface{},
	cmdCh chan<- interface{}) bool {

	v.errorCh = make(chan struct{})
	select {
	case cmdCh <- &insDeleteCmd{}:
	case <-time.After(time.Second):
		t.Error("Timed out sending Stop command")
		return false
	}

	for {
		select {
		case <-v.errorCh:
			v.errorCh = nil
			t.Error("Delete command Failed")
			return false
		case ovsCmd := <-ovsCh:
			switch ovsCmd.(type) {
			case *ovsStatusCmd:
				return true
			case *ovsStatsUpdateCmd:
			default:
				t.Error("Unexpected commands received on ovsCh")
				return false
			}
		case monCmd := <-v.monitorCh:
			if _, stopCmd := monCmd.(virtualizerStopCmd); !stopCmd {
				t.Errorf("Invalid monitor command found %t, expected virtualizerStopCmd", monCmd)
				return false
			}
			close(v.monitorClosedCh)
		case <-time.After(time.Second):
			t.Error("Timed out waiting for ovsStatsUpdateCmd")
			return false
		}
	}
}

func (v *instanceTestState) ClusterConfiguration() (payloads.Configure, error) {
	return payloads.Configure{}, nil
}

func cleanupShutdownFail(t *testing.T, instance string, doneCh chan struct{}, ovsCh chan interface{}, wg *sync.WaitGroup) {
	_ = os.RemoveAll(path.Join(testInstancesDir, instance))

	shutdownInstanceLoop(doneCh, ovsCh, wg, t)
	t.FailNow()
}

func waitForStateChange(t *testing.T, ovsState ovsRunningState, ovsCh chan interface{}) bool {
	for {
		select {
		case ovsCmd := <-ovsCh:
			switch stChange := ovsCmd.(type) {
			case *ovsStateChange:
				if stChange.state != ovsState {
					t.Errorf("ovs state %d expected.  Found state %d",
						ovsState, stChange.state)
					return false
				}
				return true
			case *ovsStatsUpdateCmd:
			default:
				t.Error("Unexpected commands received on ovsCh")
				return false
			}
		case <-time.After(time.Second):
			t.Error("Timed out waiting for overseer channel")
			return false
		}
	}
}

func (v *instanceTestState) startInstance(t *testing.T, ovsCh chan interface{},
	cmdCh chan<- interface{}, cfg *vmConfig, errorOk bool) bool {

	v.errorCh = make(chan struct{})
	select {
	case cmdCh <- &insStartCmd{cfg: cfg, rcvStamp: time.Now()}:
	case <-time.After(time.Second):
		t.Error("Timed out sending Stop command")
		return false
	}

DONE:
	for {
		select {
		case <-v.errorCh:
			v.errorCh = nil
			if !errorOk {
				t.Error("Start command Failed")
				return false
			}
			return true
		case ovsCmd := <-ovsCh:
			switch ovsCmd.(type) {
			case *ovsStatusCmd:
				break DONE
			case *ovsStatsUpdateCmd:
			default:
				t.Error("Unexpected commands received on ovsCh")
				return false
			}
		case <-time.After(time.Second):
			t.Error("Timed out waiting for ovsStatsUpdateCmd")
			return false
		}
	}

	if !v.connect {
		return true
	}

	if !waitForStateChange(t, ovsRunning, ovsCh) {
		return false
	}

	return v.expectStatsUpdate(t, ovsCh)
}

func (v *instanceTestState) restartInstance(t *testing.T, ovsCh chan interface{},
	cmdCh chan<- interface{}, errorOk bool) bool {

	v.errorCh = make(chan struct{})
	select {
	case cmdCh <- &insRestartCmd{}:
	case <-time.After(time.Second):
		t.Error("Timed out sending Restart command")
		return false
	}

	for {
		select {
		case <-v.errorCh:
			v.errorCh = nil
			if !errorOk {
				t.Error("Restart command Failed")
			}
			return false
		case ovsCmd := <-ovsCh:
			switch stChange := ovsCmd.(type) {
			case *ovsStateChange:
				if stChange.state != ovsRunning {
					t.Errorf("ovsRunning expected.  Found state %d", stChange.state)
					return false
				}
				return true
			case *ovsStatsUpdateCmd:
			default:
				t.Error("Unexpected commands received on ovsCh")
				return false
			}
		case <-time.After(time.Second):
			t.Error("Timed out waiting for ovsStatsUpdateCmd")
			return false
		}
	}
}

func shutdownInstanceLoop(doneCh chan struct{}, ovsCh chan interface{}, wg *sync.WaitGroup,
	t *testing.T) {
	close(doneCh)

	timeout := time.After(time.Second * 5)
DONE:
	for {
		select {
		case <-ovsCh:
		case <-timeout:
			t.Error("Timedout waiting for instance loop to exit")
			break DONE
		case <-func() chan struct{} {
			ch := make(chan struct{})
			go func() {
				wg.Wait()
				close(ch)
			}()
			return ch
		}():
			break DONE
		}
	}
}

// Checks that an instance loop can be started and shutdown
//
// We just check that the instanceLoop can be started and shutdown.  No commands are
// actually executed by the instance.
//
// It should be possible to start and stop the instanceLoop without any problems.
func TestStartInstanceLoop(t *testing.T) {
	var wg sync.WaitGroup
	doneCh := make(chan struct{})
	ovsCh := make(chan interface{})
	state := &instanceTestState{
		t:          t,
		instance:   "testInstance",
		statsArray: [3]int{10, 128, 10},
	}
	cfg := &vmConfig{}
	cmdWrapCh := make(chan *cmdWrapper)
	ac := &agentClient{conn: state, cmdCh: cmdWrapCh}
	_ = startInstanceWithVM(state.instance, cfg, &wg, doneCh, ac, ovsCh, state, &storage.NoopDriver{}, testInstancesDir)
	ok := state.expectStatsUpdate(t, ovsCh)
	shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
	if !ok {
		t.FailNow()
	}
}

// Checks an instance loop can be deleted before an instance is launched.
//
// We start the instance loop and then delete the instance straight away.
//
// The instanceLoop should start and should then terminate cleanly once the
// deleteCmd is received.  Note delete works here, even though we haven't
// actually started an instance.
func TestDeleteInstanceLoop(t *testing.T) {
	var wg sync.WaitGroup
	doneCh := make(chan struct{})
	ovsCh := make(chan interface{})
	state := &instanceTestState{
		t:          t,
		instance:   "testInstance",
		statsArray: [3]int{10, 128, 10},
		errorCh:    make(chan struct{}),
	}
	cfg := &vmConfig{}
	cmdWrapCh := make(chan *cmdWrapper)
	ac := &agentClient{conn: state, cmdCh: cmdWrapCh}
	cmdCh := startInstanceWithVM(state.instance, cfg, &wg, doneCh, ac, ovsCh, state,
		&storage.NoopDriver{}, testInstancesDir)

	ok := state.expectStatsUpdate(t, ovsCh)
	if !ok {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}
	wg.Wait()
}

// Check we cannot stop an instance that is not running.
//
// We start the instance loop and then try to stop the instance straight away.
// When this fails we delete the instance.
//
//  We should receive a SSNTP stopErr and the instance loop should close.
func TestStopNotRunning(t *testing.T) {
	var wg sync.WaitGroup
	doneCh := make(chan struct{})
	ovsCh := make(chan interface{})
	state := &instanceTestState{
		t:          t,
		instance:   "testInstance",
		statsArray: [3]int{10, 128, 10},
		errorCh:    make(chan struct{}),
	}
	cfg := &vmConfig{}
	cmdWrapCh := make(chan *cmdWrapper)
	ac := &agentClient{conn: state, cmdCh: cmdWrapCh}
	cmdCh := startInstanceWithVM(state.instance, cfg, &wg, doneCh, ac, ovsCh, state,
		&storage.NoopDriver{}, testInstancesDir)

	ok := state.expectStatsUpdate(t, ovsCh)
	if !ok {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}

	select {
	case cmdCh <- &insStopCmd{}:
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timed out sending Stop command")
	}

	select {
	case <-state.errorCh:
		state.errorCh = nil
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timed out waiting for error channel")
	}

	if state.sf.InstanceUUID != state.instance ||
		state.sf.Reason != payloads.StopAlreadyStopped {
		t.Error("Invalid Stop error returned")
	}

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}
	wg.Wait()
}

func startVMWithCFG(t *testing.T, wg *sync.WaitGroup, cfg *vmConfig, connect bool, errorOk bool) (*instanceTestState, chan interface{}, chan<- interface{}, chan struct{}) {
	networking = false
	doneCh := make(chan struct{})
	ovsCh := make(chan interface{})
	state := &instanceTestState{
		t:          t,
		instance:   "testInstance",
		statsArray: [3]int{10, 128, 10},
		connect:    connect,
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}
	cmdCh := startInstanceWithVM(state.instance, cfg, wg, doneCh, state.ac, ovsCh, state,
		&storage.NoopDriver{}, testInstancesDir)
	if !state.expectStatsUpdate(t, ovsCh) {
		shutdownInstanceLoop(doneCh, ovsCh, wg, t)
		t.FailNow()
	}

	if !state.startInstance(t, ovsCh, cmdCh, cfg, errorOk) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, wg)
	}
	return state, ovsCh, cmdCh, doneCh
}

// Check we can start an instance that is not running.
//
// We start the instance loop and then try to start an instance.  Our test virtualizer
// closes the connected channel to indicate that the instance is running.  We then
// check to see whether we receive the state change notification at which point we
// delete the instance.
//
// The instance is started and deleted correctly and the instanceLoop should close
// down cleanly.
func TestStartNotRunning(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

// Check we can delete an instance which has been started but has not yet connected.
//
// We start the instance loop and then try to start an instance.  The key point here
// is that we do not close the connected channel, simulating a qemu instance for
// example that has not yet started up.  We then delete the instance.
//
// The instance is started and deleted correctly and the instanceLoop should close
// down cleanly.
func TestDeleteNoConnect(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, false, false)

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		_ = os.RemoveAll(path.Join(testInstancesDir, cfg.Instance))
		close(doneCh)
		t.FailNow()
	}

	wg.Wait()
}

// Check we can shut down the instance loop cleanly when we have a running instance.
//
// We start the instance loop and then try to start an instance.  Our test virtualizer
// closes the connected channel to indicate that the instance is running.  We then
// close the doneCh channel simulating a launcher exit.  We need to explicitly delete
// the instance directory, so the subsequent tests don't fail.
//
// The instance is started correctly and the instanceLoop shuts down cleanly.
func TestLoopShutdownWithRunningInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	_, ovsCh, _, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	shutdownInstanceLoop(doneCh, ovsCh, &wg, t)

	// We need to remove the instance manually to have a clean setup for the
	// subsequent tests.

	_ = os.RemoveAll(path.Join(testInstancesDir, cfg.Instance))
}

// Check we can restart an instance
//
// We start the instance loop and then try to restart an instance.  Our test virtualizer
// closes the connected channel to indicate that the instance is running.  We then
// check to see whether we receive the state change notification at which point we
// close the doneCh.
//
// The instance should start correctly.  We should receive an error when attempting
// to restart the instance.  The instanceLoop should quit cleanly.
func TestRestart(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	doneCh := make(chan struct{})
	ovsCh := make(chan interface{})
	state := &instanceTestState{
		t:          t,
		instance:   "testInstance",
		statsArray: [3]int{10, 128, 10},
		connect:    true,
	}
	cmdWrapCh := make(chan *cmdWrapper)
	ac := &agentClient{conn: state, cmdCh: cmdWrapCh}
	cmdCh := startInstanceWithVM(state.instance, &cfg, &wg, doneCh, ac, ovsCh, state,
		&storage.NoopDriver{}, testInstancesDir)
	ok := state.expectStatsUpdate(t, ovsCh)
	if !ok {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}

	if !state.restartInstance(t, ovsCh, cmdCh, false) {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}

	shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
}

// Check we can handle a restart error
//
// We start the instanceLoop and then try to restart an instance.  This attempt
// will fail as we've configured startVm to return an error.  We then shutdown
// the instance loop.
//
// The instanceLoop should start correctly, the restartCommand should fail with
// the correct error and the instanceLoop should close down cleanly.
func TestRestartFail(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	doneCh := make(chan struct{})
	ovsCh := make(chan interface{})
	state := &instanceTestState{
		t:           t,
		instance:    "testInstance",
		statsArray:  [3]int{10, 128, 10},
		connect:     true,
		failStartVM: true,
	}
	cmdWrapCh := make(chan *cmdWrapper)
	ac := &agentClient{conn: state, cmdCh: cmdWrapCh}
	cmdCh := startInstanceWithVM(state.instance, &cfg, &wg, doneCh, ac, ovsCh, state,
		&storage.NoopDriver{}, testInstancesDir)
	ok := state.expectStatsUpdate(t, ovsCh)
	if !ok {
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.FailNow()
	}

	if state.restartInstance(t, ovsCh, cmdCh, true) {
		t.Error("Restart was expected to Fail")
	}

	if state.rf.Reason != payloads.RestartLaunchFailure {
		t.Errorf("Invalid restart error found %s, expected %s",
			state.rf.Reason, payloads.RestartLaunchFailure)
	}

	shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
}

// Check we get an error when starting an instance with an invalid image
//
// We start the instance loop and then try to start an instance with an invalid
// config. This should cause a sudicide command to get sent to the acCmd channel.
// We'll extract this command and send it back down our instance channel,
// which should kill the instanceLoop.
//
// The instanceLoop should start correctly but the start command should fail.
// The suicide command recevied from the acCmd channel should terminate the
// instanceLoop cleanly.
func TestStartBadImage(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	cfg.Image = ""

	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, true)
	if state.stf.Reason != payloads.InvalidData {
		t.Errorf("Incorrect error returned. Reported %s, expected %s",
			string(state.stf.Reason), string(payloads.ImageFailure))
	}

	select {
	case acCmd := <-state.ac.cmdCh:
		state.errorCh = make(chan struct{})
		select {
		case cmdCh <- acCmd.cmd:
		case <-time.After(time.Second):
			shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
			t.Fatal("Timed out sending suicide command")
		}
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timedout waiting from suicide command")
	}
	wg.Wait()

	select {
	case <-state.errorCh:
		state.errorCh = nil
		t.Error("Suicide Delete failed unexpectedly")
	default:
	}
}

func sendCommandDuringSuicide(t *testing.T, testCmd interface{}) *instanceTestState {
	var wg sync.WaitGroup
	cfg := standardCfg
	cfg.Image = ""

	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, true)
	if state.stf.Reason != payloads.InvalidData {
		t.Errorf("Incorrect error returned. Reported %s, expected %s",
			string(state.stf.Reason), string(payloads.ImageFailure))
	}

	var acCmd *cmdWrapper
	select {
	case acCmd = <-state.ac.cmdCh:
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timedout waiting from suicide command")
	}

	state.errorCh = make(chan struct{})
	select {
	case cmdCh <- testCmd:
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timed out sending command during suicide")
	}

	select {
	case <-state.errorCh:
		state.errorCh = nil
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timed out waiting on error channel")
	}

	select {
	case cmdCh <- acCmd.cmd:
	case <-time.After(time.Second):
		shutdownInstanceLoop(doneCh, ovsCh, &wg, t)
		t.Fatal("Timed out sending suicide command")
	}

	wg.Wait()

	select {
	case <-state.errorCh:
		state.errorCh = nil
		t.Fatal("Suicide Delete failed unexpectedly")
	default:
	}

	return state
}

// Test deleting an instance that failed to start and is suiciding.
//
// We start the instance loop and then try to start an instance. This should cause
// a suicide command to get sent to the acCmd channel.  We then send a delete
// command to the instance (without the suicide flag set).  This command should
// fail.  We then send the real suicide command received from the acCmd channel,
// which should succeed.
//
// The instanceLoop should start, the start command and the first delete command
// should fail.  The second delete (suicide) should succeed and the loop should
// exit.
func TestDeleteNoInstance(t *testing.T) {
	state := sendCommandDuringSuicide(t, &insDeleteCmd{})
	if state.df.Reason != payloads.DeleteNoInstance {
		t.Errorf("Incorrect error returned. Reported %s, expected %s",
			string(state.df.Reason), string(payloads.DeleteNoInstance))
	}
}

// Test restarting an instance that failed to start and is suiciding.
//
// We start the instance loop and then try to start an instance. This should cause
// a suicide command to get sent to the acCmd channel.  We then send a restart
// command to the instance.  This command should fail.  We then send the suicide
// command received from the acCmd channel, which should succeed.
//
// The instanceLoop should start, the start command and the restart command
// should fail.  The delete (suicide) should succeed and the loop should
// exit.
func TestRestartNoInstance(t *testing.T) {
	state := sendCommandDuringSuicide(t, &insRestartCmd{})
	if state.rf.Reason != payloads.RestartNoInstance {
		t.Errorf("Incorrect error returned. Reported %s, expected %s",
			string(state.rf.Reason), string(payloads.RestartNoInstance))
	}
}

// Test stopping an instance that failed to start and is suiciding.
//
// We start the instance loop and then try to start an instance. This should cause
// a suicide command to get sent to the acCmd channel.  We then send a stop
// command to the instance.  This command should fail.  We then send the suicide
// command received from the acCmd channel, which should succeed.
//
// The instanceLoop should start, the start command and the stop command
// should fail.  The delete (suicide) should succeed and the loop should
// exit.
func TestStopNoInstance(t *testing.T) {
	state := sendCommandDuringSuicide(t, &insStopCmd{})
	if state.sf.Reason != payloads.StopNoInstance {
		t.Errorf("Incorrect error returned. Reported %s, expected %s",
			string(state.sf.Reason), string(payloads.StopNoInstance))
	}
}

// Check the instanceLoop copes when an instance is dropped.
//
// We start the instance loop and then try to start an instance.  Our test virtualizer
// closes the connected channel to indicate that the instance is running.  We then close
// the monitorCloseCh channel informing the instanceLoop that the instance has dropped.
// We then delete the instance.
//
// The instanceLoop and then instance should start correctly.  We should receive
// a state change notification when we simulate the instances untimely demise.
// The instance should then be deleted correctly and the instanceLoop should exit
// cleanly.
func TestLostInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	close(state.monitorClosedCh)

	if !waitForStateChange(t, ovsStopped, ovsCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	// This gets closed by the instanceLoop and so will become available
	// in the deleteInstance select loop if we don't set it to nil.
	state.monitorCh = nil

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

// Check we get an error when starting a running instance.
//
// We start the instance loop and then try to start an instance.  Our test virtualizer
// closes the connected channel to indicate that the instance is running.  We then
// send another start command and delete the instance.
//
// The instanceLoop and then instance should start correctly.  The second start
// command should fail.  The instance should then be deleted correctly and
// the instanceLoop should exit cleanly.
func TestStartRunningInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	if !state.startInstance(t, ovsCh, cmdCh, &cfg, true) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	if state.stf.Reason != payloads.AlreadyRunning {
		t.Errorf("Invalid Error received.  Expected %s found %s",
			string(state.stf.Reason), string(payloads.AlreadyRunning))
	}

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

// Check we can add a volume to an instance
//
// We start the instance loop, add a volume, wait for the instance statistics
// and then delete the instance.
//
// The instanceLoop and then instance should start correctly.  The volume should
// be correctly attached and the stats command should verify this.  The instance
// should be correctly deleted.
func TestAttachVolumeToInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	select {
	case cmdCh <- &insAttachVolumeCmd{testutil.VolumeUUID}:
	case <-time.After(time.Second):
		t.Error("Timed out sending attach volume command")
	}

	select {
	case monCmd := <-state.monitorCh:
		monCmd.(virtualizerAttachCmd).responseCh <- nil
	case <-time.After(time.Second):
		t.Error("Timed out waiting for attach volume command result")
	}

	_ = state.expectStatsUpdateWithVolumes(t, ovsCh, []string{testutil.VolumeUUID})

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

// Check that adding an existing volume fails
//
// We start the instance loop, add a volume, add the volume a second time
// and then delete the instance.
//
// The instanceLoop and then instance should start correctly.  The volume should
// be correctly attached the first time.  The second attempt should fail. The
// instance should be correctly deleted.
func TestAttachExistingVolumeToInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	select {
	case cmdCh <- &insAttachVolumeCmd{testutil.VolumeUUID}:
	case <-time.After(time.Second):
		t.Error("Timed out sending attach volume command")
	}

	select {
	case monCmd := <-state.monitorCh:
		monCmd.(virtualizerAttachCmd).responseCh <- nil
	case <-time.After(time.Second):
		t.Error("Timed out waiting for attach volume command result")
	}

	_ = state.expectStatsUpdateWithVolumes(t, ovsCh, []string{testutil.VolumeUUID})

	select {
	case <-state.errorCh:
		t.Error("Initial Volume attach failed")
	case cmdCh <- &insAttachVolumeCmd{testutil.VolumeUUID}:
	case <-time.After(time.Second):
		t.Error("Timed out sending attach volume command")
	}

	select {
	case <-state.errorCh:
		if state.avf.Reason != payloads.AttachVolumeAlreadyAttached {
			t.Errorf("Unexpected error.  Expected %s got %s",
				payloads.AttachVolumeAlreadyAttached, state.avf.Reason)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for attach to fail")
	}

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

// Check we can detach a volume from an instance
//
// We start the instance loop, add a volume, wait for the instance statistics,
// detach the volume, wait for more statistics and then delete the instance.
//
// The instanceLoop and then instance should start correctly.  The volume should
// be correctly attached and the stats command should verify this.  The volume
// should be successfully detached, verified again by stats, and the instance
// should be correctly deleted.
func TestDetachVolumeFromInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	select {
	case cmdCh <- &insAttachVolumeCmd{testutil.VolumeUUID}:
	case <-time.After(time.Second):
		t.Error("Timed out sending attach volume command")
	}

	select {
	case monCmd := <-state.monitorCh:
		monCmd.(virtualizerAttachCmd).responseCh <- nil
	case <-time.After(time.Second):
		t.Error("Timed out waiting for attach volume command result")
	}

	_ = state.expectStatsUpdateWithVolumes(t, ovsCh, []string{testutil.VolumeUUID})

	select {
	case cmdCh <- &insDetachVolumeCmd{testutil.VolumeUUID}:
	case <-time.After(time.Second):
		t.Error("Timed out sending attach volume command")
	}

	select {
	case monCmd := <-state.monitorCh:
		monCmd.(virtualizerDetachCmd).responseCh <- nil
	case <-time.After(time.Second):
		t.Error("Timed out waiting for attach volume command result")
	}

	_ = state.expectStatsUpdateWithVolumes(t, ovsCh, []string{})

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

// Check that detaching a nonexistent volume fails
//
// We start the instance loop, detach a volume delete the instance.
//
// The instanceLoop and then instance should start correctly.  The volume should
// be fail to be detached as it doesn't exist. The instance should be correctly
// deleted.
func TestDetachNonexistingVolumeFromInstance(t *testing.T) {
	var wg sync.WaitGroup
	cfg := standardCfg
	state, ovsCh, cmdCh, doneCh := startVMWithCFG(t, &wg, &cfg, true, false)

	select {
	case cmdCh <- &insDetachVolumeCmd{testutil.VolumeUUID}:
	case <-time.After(time.Second):
		t.Error("Timed out sending attach volume command")
	}

	select {
	case <-state.errorCh:
		if state.dvf.Reason != payloads.DetachVolumeNotAttached {
			t.Errorf("Unexpected error.  Expected %s got %s",
				payloads.DetachVolumeNotAttached, state.dvf.Reason)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for attach to fail")
	}

	if !state.deleteInstance(t, ovsCh, cmdCh) {
		cleanupShutdownFail(t, cfg.Instance, doneCh, ovsCh, &wg)
	}

	wg.Wait()
}

func TestMain(m *testing.M) {
	flag.Parse()
	var err error
	testInstancesDir, err = ioutil.TempDir("", "instances")
	if err != nil {
		panic(fmt.Sprintf("Unable to create instances dir: %v", err))
	}
	exit := m.Run()
	_ = os.RemoveAll(testInstancesDir)
	os.Exit(exit)
}
