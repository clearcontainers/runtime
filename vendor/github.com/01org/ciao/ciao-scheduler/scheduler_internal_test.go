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

package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/testutil"
)

// an ssntpSchedulerServer instance for non-SSNTP unit tests
var sched *ssntpSchedulerServer

/****************************************************************************/
// dummy controller creation

func spinUpController(sched *ssntpSchedulerServer, ident int, status controllerStatus) {
	var controller controllerStat
	controller.status = status
	controller.uuid = fmt.Sprintf("%08d", ident)

	sched.controllerMutex.Lock()
	defer sched.controllerMutex.Unlock()

	if controller.status == controllerMaster {
		// master at the front of list
		sched.controllerList = append([]*controllerStat{&controller}, sched.controllerList...)
	} else {
		// backup controllers at the end of the list
		sched.controllerList = append(sched.controllerList, &controller)
	}
	sched.controllerMap[controller.uuid] = &controller
}

/****************************************************************************/
// dummy node creation

func spinUpComputeNode(sched *ssntpSchedulerServer, ident int, RAM int) {
	var node nodeStat
	node.status = ssntp.READY
	node.uuid = fmt.Sprintf("%08d", ident)
	node.memTotalMB = RAM
	node.memAvailMB = RAM
	node.load = 0
	node.cpus = 4

	sched.cnMutex.Lock()
	defer sched.cnMutex.Unlock()

	if sched.cnMap[node.uuid] != nil {
		fmt.Println("poorly written test: ignoring creation of duplicate compute node")
		return
	}

	sched.cnList = append(sched.cnList, &node)
	sched.cnMap[node.uuid] = &node
}

func spinUpComputeNodeLarge(sched *ssntpSchedulerServer, ident int) {
	spinUpComputeNode(sched, ident, 141312)
}
func spinUpComputeNodeSmall(sched *ssntpSchedulerServer, ident int) {
	spinUpComputeNode(sched, ident, 16384)
}
func spinUpComputeNodeVerySmall(sched *ssntpSchedulerServer, ident int) {
	spinUpComputeNode(sched, ident, 200)
}

func spinUpNetworkNode(sched *ssntpSchedulerServer, ident int, RAM int) {
	var node nodeStat
	node.status = ssntp.READY
	node.uuid = fmt.Sprintf("%08d", ident)
	node.memTotalMB = RAM
	node.memAvailMB = RAM
	node.load = 0
	node.cpus = 4

	sched.nnMutex.Lock()
	defer sched.nnMutex.Unlock()

	if sched.nnMap[node.uuid] != nil {
		fmt.Println("poorly written test: ignoring creation of duplicate network node")
		return
	}

	sched.nnMap[node.uuid] = &node
}

func spinUpNetworkNodeLarge(sched *ssntpSchedulerServer, ident int) {
	spinUpNetworkNode(sched, ident, 141312)
}
func spinUpNetworkNodeSmall(sched *ssntpSchedulerServer, ident int) {
	spinUpNetworkNode(sched, ident, 16384)
}
func spinUpNetworkNodeVerySmall(sched *ssntpSchedulerServer, ident int) {
	spinUpNetworkNode(sched, ident, 200)
}

/****************************************************************************/
// dummy workload creation

// set up a dummy START command
func createStartWorkload(vCpus int, memMB int, diskMB int) *payloads.Start {
	var work payloads.Start

	work.Start.InstanceUUID = "c73322e8-d5fe-4d57-874c-dcee4fd368cd"
	work.Start.ImageUUID = "b265f62b-e957-47fd-a0a2-6dc261c7315c"

	reqVcpus := payloads.RequestedResource{
		Type:      "vcpus",
		Value:     vCpus,
		Mandatory: true,
	}
	reqMem := payloads.RequestedResource{
		Type:      "mem_mb",
		Value:     memMB,
		Mandatory: true,
	}
	work.Start.RequestedResources = append(work.Start.RequestedResources, reqVcpus)
	work.Start.RequestedResources = append(work.Start.RequestedResources, reqMem)

	//TODO: add EstimatedResources

	work.Start.FWType = payloads.EFI
	work.Start.InstancePersistence = payloads.Host

	return &work
}

// TODO: create, use other commands

/****************************************************************************/

func TestMain(m *testing.M) {
	flag.Parse()

	err := ssntpTestsSetup()
	if err != nil {
		os.Exit(1)
	}

	ret := m.Run()

	ssntpTestsTeardown()
	os.Exit(ret)
}

func TestPickComputeNode(t *testing.T) {
	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}

	var work = createStartWorkload(2, 256, 10000)
	resources, err := sched.getWorkloadResources(work)
	if err != nil ||
		resources.instanceUUID != "c73322e8-d5fe-4d57-874c-dcee4fd368cd" ||
		resources.memReqMB != 256 {
		t.Fatalf("bad workload resources %s, %d", resources.instanceUUID, resources.memReqMB)
	}

	// no compute nodes
	node := PickComputeNode(sched, "", &resources)
	if node != nil {
		t.Error("fount fit in empty node list")
	}

	// 1st compute node, with little memory
	spinUpComputeNodeVerySmall(sched, 1)
	node = PickComputeNode(sched, "", &resources)
	if node != nil {
		t.Error("found fit when none should exist")
	}

	// 2nd compute node, with little memory
	spinUpComputeNodeVerySmall(sched, 2)
	node = PickComputeNode(sched, "", &resources)
	if node != nil {
		t.Error("found fit when none should exist")
	}

	// 3rd compute node, with a lot of memory
	spinUpComputeNodeLarge(sched, 3)
	node = PickComputeNode(sched, "", &resources)
	if node == nil {
		t.Error("found no fit when one should exist")
	}

	// 100 compute nodes := earlier 1 + 1 + 1 + now 97 more compute nodes
	for i := 4; i < 100; i++ {
		spinUpComputeNode(sched, i, 256*i)
	}
	node = PickComputeNode(sched, "", &resources)
	if node == nil {
		t.Error("failed to fit in hundred node list")
	}

	// MRU set somewhere arbitrary
	sched.cnMRUIndex = 42
	node = PickComputeNode(sched, "", &resources)
	if node == nil {
		t.Error("failed to find fit after MRU")
	}
}

func benchmarkPickComputeNode(b *testing.B, nodecount int) {
	sched = configSchedulerServer()
	if sched == nil {
		b.Fatal("unable to configure test scheduler")
	}

	// eg: idle, small compute nodes
	for i := 0; i < nodecount; i++ {
		spinUpComputeNode(sched, i, 16138)
	}

	var work = createStartWorkload(2, 256, 10000)
	resources, err := sched.getWorkloadResources(work)
	if err != nil {
		b.Fatal("bad workload resources")
	}

	b.ResetTimer()
	// setup complete

	for i := 0; i < b.N; i++ {
		PickComputeNode(sched, "", &resources)
	}
}

func BenchmarkPickComputeNode10(b *testing.B) {
	benchmarkPickComputeNode(b, 10)
}
func BenchmarkPickComputeNode100(b *testing.B) {
	benchmarkPickComputeNode(b, 100)
}
func BenchmarkPickComputeNode1000(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 1k cn picker bench in short mode.")
	}
	benchmarkPickComputeNode(b, 1000)
}
func BenchmarkPickComputeNode10000(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 10k cn picker bench in short mode.")
	}
	benchmarkPickComputeNode(b, 10000)
}
func BenchmarkPickComputeNode100000(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 100k cn picker bench in short mode.")
	}
	benchmarkPickComputeNode(b, 100000)
}
func BenchmarkPickComputeNode1000000(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 1Mc n picker bench in short mode.")
	}
	benchmarkPickComputeNode(b, 1000000)
}

func TestHeartBeatController(t *testing.T) {
	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}

	// zero controllers
	beatTxt := heartBeatControllers(sched)
	if beatTxt != " -no Controller- \t\t\t\t\t" {
		t.Error("missing header for empty controller list")
	}

	// first controller
	spinUpController(sched, 1, controllerMaster)
	beatTxt = heartBeatControllers(sched)
	expected := "controller-00000001:MASTER\t\t\t\t"
	if beatTxt != expected {
		t.Errorf("expected \"%s\", got \"%s\"", expected, beatTxt)
	}

	// third controller (doesn't show)
	spinUpController(sched, 2, controllerBackup)
	spinUpController(sched, 3, controllerBackup)
	beatTxt = heartBeatControllers(sched)
	expected = "controller-00000001:MASTER, controller-00000002:BACKUP\t\t"
	if beatTxt != expected {
		t.Errorf("expected \"%s\", got \"%s\"", expected, beatTxt)
	}

	// multiple masters not allowed
	spinUpController(sched, 4, controllerMaster)
	beatTxt = heartBeatControllers(sched)
	expected = "ERROR multiple controller masters"
	if beatTxt != expected {
		t.Errorf("expected \"%s\", got \"%s\"", expected, beatTxt)
	}
}

func TestHeartBeatComputeNodes(t *testing.T) {
	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}

	// zero compute nodes
	beatTxt := heartBeatComputeNodes(sched)
	if beatTxt != " -no Compute Nodes-" {
		t.Error("missing header for empty node list")
	}

	// first compute node
	spinUpComputeNode(sched, 1, 16138)
	beatTxt = heartBeatComputeNodes(sched)
	expected := "node-00000001:READY:16138/16138,0"
	if beatTxt != expected {
		t.Errorf("expected \"%s\", got \"%s\"", expected, beatTxt)
	}

	// fifth compute node (doesn't show)
	spinUpComputeNode(sched, 2, 256)
	spinUpComputeNode(sched, 3, 10000)
	spinUpComputeNode(sched, 4, 42)
	spinUpComputeNode(sched, 5, 44032)
	beatTxt = heartBeatComputeNodes(sched)
	expected = "node-00000001:READY:16138/16138,0, node-00000002:READY:256/256,0, node-00000003:READY:10000/10000,0, node-00000004:READY:42/42,0"
	if beatTxt != expected {
		t.Errorf("expected \"%s\", got \"%s\"", expected, beatTxt)
	}
}

func TestHeartBeat(t *testing.T) {
	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}

	beatTxt := heartBeat(sched, 0)
	expected := "** idle / disconnected **\n"
	if beatTxt != expected {
		t.Errorf("expected \"%s\", got \"%s\"", expected, beatTxt)
	}

	spinUpController(sched, 1, controllerMaster)
	spinUpController(sched, 2, controllerBackup)
	spinUpController(sched, 3, controllerBackup)
	spinUpComputeNode(sched, 1, 16138)
	spinUpComputeNode(sched, 2, 256)
	spinUpComputeNode(sched, 3, 10000)
	spinUpComputeNode(sched, 4, 42)
	spinUpComputeNode(sched, 5, 44032)
	spinUpNetworkNode(sched, 1001, 16138)
	spinUpNetworkNode(sched, 1002, 256)
	spinUpNetworkNode(sched, 1003, 10000)
	spinUpNetworkNode(sched, 1004, 42)
	spinUpNetworkNode(sched, 1005, 44032)
	beatTxt = heartBeat(sched, 1)
	expected = "controller-00000001:MASTER, controller-00000002:BACKUP\t\tnode-00000001:READY:16138/16138,0, node-00000002:READY:256/256,0, node-00000003:READY:10000/10000,0, node-00000004:READY:42/42,0"
	if beatTxt != expected {
		t.Errorf("expected:\n\"%s\"\ngot:\n\"%s\"", expected, beatTxt)
	}

	beatTxt = heartBeat(sched, heartBeatHeaderFreq-1)
	expectedWithHeader := "Controllers\t\t\t\t\tCompute Nodes\n" + expected
	if beatTxt != expectedWithHeader {
		t.Errorf("expected:\n\"%s\"\ngot:\n\"%s\"", expectedWithHeader, beatTxt)
	}
}

func controllerMods() {
	// controller in and out
	ConnectController(sched, "1")
	DisconnectController(sched, "1")

	// controller master and two backups
	ConnectController(sched, "1")
	ConnectController(sched, "2")
	ConnectController(sched, "3")

	// remove a backup
	DisconnectController(sched, "3")

	// remove master
	DisconnectController(sched, "1")

	// remove last
	DisconnectController(sched, "2")
}
func computeNodeMods() {
	// compute node in and out
	ConnectComputeNode(sched, "1")
	DisconnectComputeNode(sched, "1")

	// multiple compute nodes in various orders
	ConnectComputeNode(sched, "1")
	ConnectComputeNode(sched, "2")
	ConnectComputeNode(sched, "3")
	ConnectComputeNode(sched, "4")
	DisconnectComputeNode(sched, "1")
	DisconnectComputeNode(sched, "2")
	DisconnectComputeNode(sched, "3")
	DisconnectComputeNode(sched, "4")
	ConnectComputeNode(sched, "1")
	ConnectComputeNode(sched, "2")
	ConnectComputeNode(sched, "3")
	ConnectComputeNode(sched, "4")
	DisconnectComputeNode(sched, "4")
	DisconnectComputeNode(sched, "3")
	DisconnectComputeNode(sched, "2")
	DisconnectComputeNode(sched, "1")
	ConnectComputeNode(sched, "1")
	ConnectComputeNode(sched, "2")
	ConnectComputeNode(sched, "3")
	ConnectComputeNode(sched, "4")
	DisconnectComputeNode(sched, "3")
	DisconnectComputeNode(sched, "1")
	DisconnectComputeNode(sched, "4")
	DisconnectComputeNode(sched, "2")
}
func networkNodeMods() {
	// network node in and out
	ConnectNetworkNode(sched, "1")
	DisconnectNetworkNode(sched, "1")

	// multiple network nodes in various orders
	ConnectNetworkNode(sched, "1")
	ConnectNetworkNode(sched, "2")
	ConnectNetworkNode(sched, "3")
	ConnectNetworkNode(sched, "4")
	DisconnectNetworkNode(sched, "1")
	DisconnectNetworkNode(sched, "2")
	DisconnectNetworkNode(sched, "3")
	DisconnectNetworkNode(sched, "4")
	ConnectNetworkNode(sched, "1")
	ConnectNetworkNode(sched, "2")
	ConnectNetworkNode(sched, "3")
	ConnectNetworkNode(sched, "4")
	DisconnectNetworkNode(sched, "4")
	DisconnectNetworkNode(sched, "3")
	DisconnectNetworkNode(sched, "2")
	DisconnectNetworkNode(sched, "1")
	ConnectNetworkNode(sched, "1")
	ConnectNetworkNode(sched, "2")
	ConnectNetworkNode(sched, "3")
	ConnectNetworkNode(sched, "4")
	DisconnectNetworkNode(sched, "3")
	DisconnectNetworkNode(sched, "1")
	DisconnectNetworkNode(sched, "4")
	DisconnectNetworkNode(sched, "2")
}

func clientMiscMods() {
	/* various interleaved ******************************/
	ConnectNetworkNode(sched, "a")
	ConnectComputeNode(sched, "1")
	ConnectController(sched, "1")
	DisconnectController(sched, "1")
	DisconnectComputeNode(sched, "1")
	DisconnectNetworkNode(sched, "a")
	ConnectNetworkNode(sched, "a")
	ConnectComputeNode(sched, "1")
	ConnectNetworkNode(sched, "b")
	ConnectController(sched, "c1")
	DisconnectController(sched, "c1")
	ConnectComputeNode(sched, "2")
	DisconnectComputeNode(sched, "1")
	DisconnectNetworkNode(sched, "a")
	ConnectController(sched, "c1")
	ConnectController(sched, "c2")
	ConnectComputeNode(sched, "3")
	ConnectComputeNode(sched, "4")
	ConnectComputeNode(sched, "5")
	DisconnectComputeNode(sched, "2")
	DisconnectNetworkNode(sched, "b")
	DisconnectController(sched, "c2")
	DisconnectController(sched, "c1")
	DisconnectComputeNode(sched, "3")
	DisconnectComputeNode(sched, "4")
	DisconnectComputeNode(sched, "5")
}

// TestClientMgmtLocking should run to completion without deadlocking or
// panic'ing.  If it does not, "go test -race" should highlight the
// problem.
func TestClientMgmtLocking(t *testing.T) {
	var wg sync.WaitGroup

	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}

	// simple first serial sanity check
	controllerMods()
	computeNodeMods()
	networkNodeMods()
	clientMiscMods()

	// now in parallel
	var iters int
	if testing.Short() {
		iters = 10000 // ~4sec
	} else {
		iters = 100000 // ~40sec
	}
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			controllerMods()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			computeNodeMods()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			networkNodeMods()
		}
	}()
	wg.Wait()
}

func TestStartWorkload(t *testing.T) {
	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}
	spinUpController(sched, 1, controllerMaster)
	var controllerUUID = fmt.Sprintf("%08d", 1)
	spinUpController(sched, 2, controllerBackup)
	spinUpController(sched, 3, controllerBackup)

	spinUpComputeNode(sched, 1, 16138)
	spinUpComputeNode(sched, 2, 256)
	spinUpComputeNode(sched, 3, 10000)
	spinUpComputeNode(sched, 4, 42)
	spinUpComputeNode(sched, 5, 44032)

	spinUpNetworkNode(sched, 1001, 16138)
	spinUpNetworkNode(sched, 1002, 256)
	spinUpNetworkNode(sched, 1003, 10000)
	spinUpNetworkNode(sched, 1004, 42)
	spinUpNetworkNode(sched, 1005, 44032)

	var dest string

	// controller starts with starting a CNCI if none are present for a tenant
	fwd, uuid := startWorkload(sched, controllerUUID, []byte(testutil.CNCIStartYaml))
	decision := fwd.Decision()
	recipients := fwd.Recipients()
	if decision != ssntp.Forward {
		t.Errorf("bad decision, got 0x%x, expected 0x%x", decision, ssntp.Forward)
	}
	if uuid != testutil.CNCIInstanceUUID {
		t.Errorf("bad uuid, got %s, expected %s", uuid, testutil.CNCIInstanceUUID)
	}
	for _, dest = range recipients[:] {
		if sched.nnMap[dest] == nil {
			t.Errorf("CNCI sent to non-network-node %s", dest)
		}
	}

	// then controller starts the tenant workload
	fwd, uuid = startWorkload(sched, controllerUUID, []byte(testutil.StartYaml))
	decision = fwd.Decision()
	recipients = fwd.Recipients()
	if decision != ssntp.Forward ||
		uuid != testutil.InstanceUUID {
		t.Errorf("unable to start workload 1, got decision=0x%x, workload uuid=%s", decision, uuid)
	}
	for _, dest = range recipients[:] {
		if sched.cnMap[dest] == nil {
			t.Errorf("tenant workload sent to non-compute-node %s", dest)
		}
	}

	// remove MRU compute compute node
	DisconnectComputeNode(sched, dest)

	// later starts another tenant workload
	fwd, uuid = startWorkload(sched, controllerUUID, []byte(testutil.StartYaml))
	decision = fwd.Decision()
	recipients = fwd.Recipients()
	if decision != ssntp.Forward ||
		uuid != testutil.InstanceUUID {
		t.Errorf("unable to start workload 2, got decision=0x%x, workload uuid=%s", decision, uuid)
	}
	for _, dest = range recipients[:] {
		if sched.cnMap[dest] == nil {
			t.Errorf("tenant workload sent to non-compute-node %s", dest)
		}
	}
}

func TestGetWorkloadAgentUUID(t *testing.T) {
	sched = configSchedulerServer()
	if sched == nil {
		t.Fatal("unable to configure test scheduler")
	}

	var stringTests = []struct {
		cmd                  ssntp.Command
		yaml                 []byte
		expectedInstanceUUID string
		expectedAgentUUID    string
	}{
		{ssntp.RESTART, []byte(testutil.RestartYaml), testutil.InstanceUUID, testutil.AgentUUID},
		{ssntp.STOP, []byte(testutil.StopYaml), testutil.InstanceUUID, testutil.AgentUUID},
		{ssntp.DELETE, []byte(testutil.DeleteYaml), testutil.InstanceUUID, testutil.AgentUUID},
		{ssntp.EVACUATE, []byte(testutil.EvacuateYaml), "", testutil.AgentUUID},
		{ssntp.AttachVolume, []byte(testutil.AttachVolumeYaml), testutil.InstanceUUID, testutil.AgentUUID},
	}
	for _, test := range stringTests {
		instanceUUID, agentUUID, _ := GetWorkloadAgentUUID(sched, test.cmd, test.yaml)
		if instanceUUID != test.expectedInstanceUUID {
			t.Errorf("failed to get correct instanceUUID, expected %s, got %s", test.expectedInstanceUUID, instanceUUID)
		}
		if agentUUID != test.expectedAgentUUID {
			t.Errorf("failed to get correct workloadAgentUUID, expected %s, got %s", test.expectedAgentUUID, agentUUID)
		}
	}
}
