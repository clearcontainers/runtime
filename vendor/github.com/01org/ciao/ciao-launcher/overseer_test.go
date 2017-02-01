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
	"encoding/gob"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
)

const memInfoContents = `
MemTotal:        1999368 kB
MemFree:         1289644 kB
MemAvailable:    1885704 kB
Buffers:           38796 kB
Cached:           543892 kB
SwapCached:            0 kB
Active:           456232 kB
Inactive:         175996 kB
Active(anon):      50128 kB
Inactive(anon):     5396 kB
Active(file):     406104 kB
Inactive(file):   170600 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:       2045948 kB
SwapFree:        2045948 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         49580 kB
Mapped:            62960 kB
Shmem:              5988 kB
Slab:              55396 kB
SReclaimable:      40152 kB
SUnreclaim:        15244 kB
KernelStack:        2176 kB
PageTables:         4196 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:     3045632 kB
Committed_AS:     380776 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
HardwareCorrupted:     0 kB
AnonHugePages:     16384 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       57280 kB
DirectMap2M:     1990656 kB
`

const loadAvgContents = `
0.00 0.01 0.05 1/134 23379
`

const statContents = `
cpu  29164 292 87649 17177990 544 0 580 0 0 0
cpu0 29164 292 87649 17177990 544 0 580 0 0 0
intr 28478654 38 10 0 0 0 0 0 0 0 0 0 0 156 0 0 169437 0 0 0 163737 19303499 21210 29 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
ctxt 54009655
btime 1465121906
processes 55793
procs_running 1
procs_blocked 0
softirq 2742553 2 1348123 34687 170653 103600 0 45 0 0 1085443
`

type overseerTestState struct {
	t        *testing.T
	ac       *agentClient
	statusCh chan *payloads.Ready
	statsCh  chan *payloads.Stat
}

func (v *overseerTestState) SendError(error ssntp.Error, payload []byte) (int, error) {
	return 0, nil
}

func (v *overseerTestState) SendEvent(event ssntp.Event, payload []byte) (int, error) {
	return 0, nil
}

func (v *overseerTestState) Dial(config *ssntp.Config, ntf ssntp.ClientNotifier) error {
	return nil
}

func (v *overseerTestState) SendStatus(status ssntp.Status, payload []byte) (int, error) {
	if v.statusCh == nil {
		return 0, nil
	}
	switch status {
	case ssntp.READY:
		ready := &payloads.Ready{}
		err := yaml.Unmarshal(payload, ready)
		if err != nil {
			v.t.Errorf("Failed to unmarshall READY status %v", err)
		}
		v.statusCh <- ready
	case ssntp.FULL:
		v.statusCh <- nil
	}

	return 0, nil
}

func (v *overseerTestState) SendCommand(cmd ssntp.Command, payload []byte) (int, error) {

	switch cmd {
	case ssntp.STATS:
		if v.statsCh == nil {
			return 0, nil
		}
		stats := &payloads.Stat{}
		err := yaml.Unmarshal(payload, stats)
		if err != nil {
			v.t.Errorf("Failed to unmarshall Stats %v", err)
		}
		v.statsCh <- stats
	}

	return 0, nil
}

func (v *overseerTestState) Role() ssntp.Role {
	return ssntp.AGENT | ssntp.NETAGENT
}

func (v *overseerTestState) UUID() string {
	return "test-uuid"
}

func (v *overseerTestState) Close() {

}

func (v *overseerTestState) isConnected() bool {
	return true
}

func (v *overseerTestState) setStatus(status bool) {

}

func (v *overseerTestState) ClusterConfiguration() (payloads.Configure, error) {
	return payloads.Configure{}, nil
}

type procPaths struct {
	procDir string
	memInfo string
	stat    string
	loadavg string
}

func createGoodProcFiles() (*procPaths, error) {
	procDir, err := ioutil.TempDir("", "overseer-proc-files")
	if err != nil {
		return nil, err
	}
	pp := &procPaths{
		procDir: procDir,
		memInfo: path.Join(procDir, "memInfo"),
		stat:    path.Join(procDir, "stat"),
		loadavg: path.Join(procDir, "loadavg"),
	}

	err = ioutil.WriteFile(pp.memInfo, []byte(memInfoContents), 0755)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(pp.stat, []byte(statContents), 0755)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(pp.loadavg, []byte(loadAvgContents), 0755)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

func shutdownOverseer(ovsCh chan<- interface{}, state *overseerTestState) {
	close(ovsCh)

DONE:
	for {
		select {
		case <-state.statusCh:
		case <-state.statsCh:
		default:
			break DONE
		}
	}
}

func addInstance(t *testing.T, ovsCh chan<- interface{}, state *overseerTestState, needStats bool) *payloads.Stat {
	addCh := make(chan ovsAddResult)

	select {
	case <-state.statusCh:
	case <-state.statsCh:
	case ovsCh <- &ovsAddCmd{
		instance: "test-instance",
		cfg: &vmConfig{
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
		},
		targetCh: addCh,
	}:
	case <-time.After(time.Second):
		t.Fatal("Unable to add instance")
	}

	var stats *payloads.Stat
	var addResult *ovsAddResult
	timer := time.After(time.Second)
DONE:
	for {
		select {
		case <-state.statusCh:
		case stats = <-state.statsCh:
			if addResult != nil {
				break DONE
			}
		case ar := <-addCh:
			if addResult == nil {
				addResult = &ar
			}
			if !needStats || stats != nil {
				break DONE
			}
		case <-timer:
			t.Fatal("Timed out waiting for Stats and AddResult")
			break DONE
		}
	}

	if !addResult.canAdd {
		t.Error("Unable to add instance")
	}

	return stats
}

func removeInstance(t *testing.T, ovsCh chan<- interface{}, state *overseerTestState, needStats bool) *payloads.Stat {
	removeCh := make(chan error)

	select {
	case ovsCh <- &ovsRemoveCmd{
		instance: "test-instance",
		errCh:    removeCh,
	}:
	case <-state.statusCh:
	case <-state.statsCh:
	case <-time.After(time.Second):
		t.Fatal("Unable to remove instance")
	}

	var stats *payloads.Stat
	var err error
	gotErr := false
	timer := time.After(time.Second)
DONE:
	for {
		select {
		case <-state.statusCh:
		case stats = <-state.statsCh:
			if gotErr {
				break DONE
			}
		case err = <-removeCh:
			gotErr = true
			if !needStats || stats != nil {
				break DONE
			}
		case <-timer:
			t.Fatal("Timed out waiting for Stats and RemoveResult")
			break DONE
		}
	}

	if err != nil {
		t.Errorf("Unable to delete instance: %v", err)
	}

	return stats
}

func getStatusStats(t *testing.T, ovsCh chan<- interface{},
	state *overseerTestState) (*payloads.Ready, *payloads.Stat) {
	select {
	case ovsCh <- &ovsStatsStatusCmd{}:
	case <-time.After(time.Second):
		t.Fatal("Unable to send ovsStatsStatusCmd")
	}

	var ready *payloads.Ready
	var stats *payloads.Stat
	timer := time.After(time.Second)
DONE:
	for {
		select {
		case ready = <-state.statusCh:
			if state.statsCh == nil || stats != nil {
				break DONE
			}
		case stats = <-state.statsCh:
			if state.statusCh == nil || ready != nil {
				break DONE
			}
		case <-timer:
			t.Fatal("Timed out waiting for Status or Stats")
			break DONE
		}
	}

	return ready, stats
}

func createTestInstance(t *testing.T, instancesDir string) {

	cfg := &vmConfig{
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
	instanceDir := path.Join(instancesDir, "test-instance")
	err := os.Mkdir(instanceDir, 0755)
	if err != nil {
		t.Fatalf("Unable to create instance directory")
	}

	cfgFilePath := path.Join(instanceDir, instanceState)
	cfgFile, err := os.OpenFile(cfgFilePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Unable to create state file %v", err)
	}
	defer func() { _ = cfgFile.Close() }()

	enc := gob.NewEncoder(cfgFile)
	err = enc.Encode(cfg)
	if err != nil {
		t.Fatalf("Failed to store state information %v", err)
	}
}

// Checks that the overseer go routine can be started and stopped.
//
// We start the overseer and then close the overseer channel to
// shut it down.
//
// Overseer should start and stop cleanly
func TestStartStopOverseer(t *testing.T) {
	diskLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{t: t}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*900,
		pp.memInfo, pp.stat, pp.loadavg)
	close(ovsCh)
	wg.Wait()
}

// Check the overseer sends stats when there are no instances.
//
// Start the overseer with a stats interval of 300ms.  Wait
// for a stats command.
//
// A stats command should be received.  Its instance array should
// be empty
func TestEmptyStats(t *testing.T) {
	diskLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:       t,
		statsCh: make(chan *payloads.Stat),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Millisecond*300,
		pp.memInfo, pp.stat, pp.loadavg)

	var stats *payloads.Stat
	timer := time.After(time.Second)
DONE:
	for {
		select {
		case stats = <-state.statsCh:
			break DONE
		case <-timer:
			t.Fatal("Timed out waiting for Status or Stats")
			break DONE
		}
	}

	if len(stats.Instances) != 0 {
		t.Errorf("Zero instances expected.  Found %d", len(stats.Instances))
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}

// Check the overseer sends a status command
//
// Start the overseer with a high stats interval and send an ovsStatusCmd.
// Shutdown the overseer.
//
// A status command should be received.  The overseer should shut down cleanly.
func TestEmptyStatus(t *testing.T) {
	diskLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:        t,
		statusCh: make(chan *payloads.Ready),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*1000,
		pp.memInfo, pp.stat, pp.loadavg)
	select {
	case ovsCh <- &ovsStatusCmd{}:
	case <-time.After(time.Second):
		t.Fatal("Unable to send ovsStatusCmd")
	}

	var ready *payloads.Ready
	timer := time.After(time.Second)
DONE:
	for {
		select {
		case ready = <-state.statusCh:
			break DONE
		case <-timer:
			t.Fatal("Timed out waiting for Status or Stats")
			break DONE
		}
	}

	if ready.NodeUUID != state.UUID() {
		t.Errorf("Unexpected UUID received for READY event, expected %s got %s",
			state.UUID(), ready.NodeUUID)
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}

// Check the overseer sends a FULL status command when no instances are
// available
//
// Start the overseer with a high stats interval and maxInstances set to zero
// and send an ovsStatusCmd.  Shutdown the overseer.
//
// A ssntp.FULL status command should be received.  The overseer should shut
// down cleanly.
func TestFullStatus(t *testing.T) {
	defer func(instances int) { maxInstances = instances }(maxInstances)
	maxInstances = 0

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:        t,
		statusCh: make(chan *payloads.Ready),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*1000,
		pp.memInfo, pp.stat, pp.loadavg)
	select {
	case ovsCh <- &ovsStatusCmd{}:
	case <-time.After(time.Second):
		t.Fatal("Unable to send ovsStatusCmd")
	}

	var ready *payloads.Ready
	timer := time.After(time.Second)
DONE:
	for {
		select {
		case ready = <-state.statusCh:
			break DONE
		case <-timer:
			t.Fatal("Timed out waiting for Status or Stats")
			break DONE
		}
	}

	if ready != nil {
		t.Errorf("Expected a FULL status message")
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}

// Check we can add and delete an instance
//
// Start the overseer, send and ovsAddCmd, check the instance is reflected
// in the next stats command.  Send an ovsDeleteCmd, check the instance is
// no longer present in the next stats command.  Shutdown overseer.
//
// It should be possible to add and delete an instance and statistics sent
// by the overseer should be updated accordingly.
func TestAddDelete(t *testing.T) {
	diskLimit = false
	memLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:       t,
		statsCh: make(chan *payloads.Stat),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*1000,
		pp.memInfo, pp.stat, pp.loadavg)

	_ = addInstance(t, ovsCh, state, false)
	_, stats := getStatusStats(t, ovsCh, state)
	if len(stats.Instances) != 1 {
		t.Errorf("1 instance expected.  Found: %d", len(stats.Instances))
	}

	_ = removeInstance(t, ovsCh, state, false)
	_, stats = getStatusStats(t, ovsCh, state)
	if len(stats.Instances) != 0 {
		t.Errorf("0 instances expected.  Found: %d", len(stats.Instances))
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}

// Checks overseer detects initial instances
//
// Prepopulate the temporary instance directory with an instance and
// start the overseer.  Then wait for a stats command and shut down
// overseer.
//
// The overseer should start correctly and the stats command should
// indicate that there is one instance pending.  The overseer should
// shutdown correctly.
func TestInitialInstance(t *testing.T) {
	diskLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	createTestInstance(t, instancesDir)

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:       t,
		statsCh: make(chan *payloads.Stat),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Millisecond*300,
		pp.memInfo, pp.stat, pp.loadavg)

	timer := time.After(time.Second)
	var stats *payloads.Stat
DONE:
	for {
		select {
		case stats = <-state.statsCh:
			break DONE
		case <-timer:
			t.Fatal("Timed out waiting for Stats")
			break DONE
		}
	}

	if len(stats.Instances) != 1 && stats.Instances[0].InstanceUUID != "test-instance" {
		t.Error("Expected one running instance called test-instance")
	}

	close(ovsCh)
	wg.Wait()
}

// Check that the ovsGetCmd works correctly.
//
// Start the overseer and add an instance.  Then try to get the
// newly added instance.  Shut down the overseer.
//
// The newly added instance should be retrieved correctly.  It's state
// should be set to pending.
func TestGet(t *testing.T) {
	diskLimit = false
	memLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t: t,
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*1000,
		pp.memInfo, pp.stat, pp.loadavg)

	_ = addInstance(t, ovsCh, state, false)

	getCh := make(chan ovsGetResult)
	select {
	case ovsCh <- &ovsGetCmd{
		instance: "test-instance",
		targetCh: getCh,
	}:
	case <-time.After(time.Second):
		t.Fatal("Unable to send ovsGetCmd")
	}

	timer := time.After(time.Second)

DONE:
	for {
		select {
		case getRes := <-getCh:
			if getRes.running != ovsPending {
				t.Error("Expected pending running state")
			}
			break DONE
		case <-timer:
			t.Fatal("Timed out waiting for get result")
			break DONE
		}
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}

// Checks the ovsStatsStatus command works
//
// Start up the overseer, send an an ovsStatsStatusCmd and then wait for the
// events from the overseer.  Close down the overseer.
//
// A stats command and a status event should be received.  The overseer should
// shut down correctly.
func TestStatsStatus(t *testing.T) {
	diskLimit = false
	memLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:        t,
		statusCh: make(chan *payloads.Ready),
		statsCh:  make(chan *payloads.Stat),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*1000,
		pp.memInfo, pp.stat, pp.loadavg)

	ready, stats := getStatusStats(t, ovsCh, state)
	if ready.NodeUUID != state.UUID() {
		t.Errorf("Unexpected UUID received for READY event, expected %s got %s",
			state.UUID(), ready.NodeUUID)
	}

	if len(stats.Instances) != 0 {
		t.Errorf("Zero instances expected.  Found %d", len(stats.Instances))
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}

// Check that the ovsStateChange command works correctly.
//
// Start the overseer, add an instance, set the instances state to
// running and then issue a statsStatusCommand.
//
// A stats command should be received for the instance and the state
// should be running.
func TestStateChange(t *testing.T) {
	diskLimit = false
	memLimit = false

	instancesDir, err := ioutil.TempDir("", "overseer-tests")
	if err != nil {
		t.Fatalf("Unable to create temporary directory")
	}
	defer func() { _ = os.RemoveAll(instancesDir) }()

	pp, err := createGoodProcFiles()
	if err != nil {
		t.Fatalf("Unable to create proc files")
	}
	defer func() { _ = os.RemoveAll(pp.procDir) }()

	var wg sync.WaitGroup
	state := &overseerTestState{
		t:       t,
		statsCh: make(chan *payloads.Stat),
	}
	state.ac = &agentClient{conn: state, cmdCh: make(chan *cmdWrapper)}

	ovsCh := startOverseerFull(instancesDir, &wg, state.ac, time.Second*1000,
		pp.memInfo, pp.stat, pp.loadavg)

	_ = addInstance(t, ovsCh, state, false)

	select {
	case ovsCh <- &ovsStateChange{
		instance: "test-instance",
		state:    ovsRunning,
	}:
	case <-time.After(time.Second):
		t.Fatal("Unable to send ovsGetCmd")
	}

	_, stats := getStatusStats(t, ovsCh, state)
	if len(stats.Instances) != 1 && stats.Instances[0].State != payloads.Running {
		t.Error("Expected one running instance")
	}

	shutdownOverseer(ovsCh, state)
	wg.Wait()
}
