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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/01org/ciao/testutil"
)

/****************************************************************************/
// pulled from testutil/client_server_test.go and modified slightly

// SSNTP entities for integrated test cases
var server *ssntpSchedulerServer
var controller *testutil.SsntpTestController
var agent *testutil.SsntpTestClient
var netAgent *testutil.SsntpTestClient
var cnciAgent *testutil.SsntpTestClient

// these status sends need to come early so the agents are marked online
// for later ssntp.START's
func TestSendAgentStatus(t *testing.T) {
	var wg sync.WaitGroup

	server.cnMutex.Lock()
	cn := server.cnMap[testutil.AgentUUID]
	if cn == nil {
		t.Fatalf("agent node not connected (uuid: %s)", testutil.AgentUUID)
	}
	server.cnMutex.Unlock()

	wg.Add(1)
	go func() {
		agent.SendStatus(163840, 163840)
		wg.Done()
	}()

	wg.Wait()
	tgtStatus := ssntp.READY
	waitForAgent(testutil.AgentUUID, &tgtStatus)

	server.cnMutex.Lock()
	cn = server.cnMap[testutil.AgentUUID]
	cn.mutex.Lock()
	defer cn.mutex.Unlock()
	if cn != nil && cn.status != tgtStatus {
		t.Fatalf("agent node incorrect status: expected %s, got %s", tgtStatus.String(), cn.status.String())
	}
	server.cnMutex.Unlock()
}
func TestSendNetAgentStatus(t *testing.T) {
	var wg sync.WaitGroup

	server.nnMutex.Lock()
	nn := server.nnMap[testutil.NetAgentUUID]
	if nn == nil {
		t.Fatalf("netagent node not connected (uuid: %s)", testutil.NetAgentUUID)
	}
	server.nnMutex.Unlock()

	wg.Add(1)
	go func() {
		netAgent.SendStatus(163840, 163840)
		wg.Done()
	}()

	wg.Wait()
	tgtStatus := ssntp.READY
	waitForNetAgent(testutil.NetAgentUUID, &tgtStatus)

	server.nnMutex.Lock()
	nn = server.nnMap[testutil.NetAgentUUID]
	nn.mutex.Lock()
	defer nn.mutex.Unlock()
	if nn != nil && nn.status != tgtStatus {
		t.Fatalf("netagent node incorrect status: expected %s, got %s", tgtStatus.String(), nn.status.String())
	}
	server.nnMutex.Unlock()
}

func TestCNCIStart(t *testing.T) {
	netAgentCh := netAgent.AddCmdChan(ssntp.START)

	go controller.Ssntp.SendCommand(ssntp.START, []byte(testutil.CNCIStartYaml))

	_, err := netAgent.GetCmdChanResult(netAgentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}

	// start CNCI agent
	cnciAgent, err = testutil.NewSsntpTestClientConnection("CNCI Client", ssntp.CNCIAGENT, testutil.CNCIUUID)
	if err != nil {
		t.Fatal(err)
	}

	controllerCh := controller.AddEventChan(ssntp.ConcentratorInstanceAdded)

	cnciAgent.SendConcentratorAddedEvent(testutil.CNCIInstanceUUID, testutil.TenantUUID, testutil.CNCIIP, testutil.CNCIMAC)

	_, err = controller.GetEventChanResult(controllerCh, ssntp.ConcentratorInstanceAdded)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStart(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)

	go controller.Ssntp.SendCommand(ssntp.START, []byte(testutil.StartYaml))

	_, err := agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartFailure(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)

	controllerErrorCh := controller.AddErrorChan(ssntp.StartFailure)
	fmt.Printf("Expecting controller to note: \"%s\"\n", ssntp.StartFailure)

	agent.StartFail = true
	agent.StartFailReason = payloads.FullCloud
	defer func() {
		agent.StartFail = false
		agent.StartFailReason = ""
	}()

	go controller.Ssntp.SendCommand(ssntp.START, []byte(testutil.StartYaml))

	_, err := agent.GetCmdChanResult(agentCh, ssntp.START)
	if err == nil { // agent will process the START and does error
		t.Fatal(err)
	}

	_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.StartFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendStats(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STATS)
	controllerCh := controller.AddCmdChan(ssntp.STATS)

	go agent.SendStatsCmd()

	_, err := agent.GetCmdChanResult(agentCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetCmdChanResult(controllerCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartTraced(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)

	traceConfig := &ssntp.TraceConfig{
		PathTrace: true,
		Start:     time.Now(),
		Label:     []byte("testutilTracedSTART"),
	}

	_, err := controller.Ssntp.SendTracedCommand(ssntp.START, []byte(testutil.StartYaml), traceConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendTrace(t *testing.T) {
	agentCh := agent.AddEventChan(ssntp.TraceReport)

	go agent.SendTrace()

	_, err := agent.GetEventChanResult(agentCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartCNCI(t *testing.T) {
	netAgentCh := netAgent.AddCmdChan(ssntp.START)

	_, err := controller.Ssntp.SendCommand(ssntp.START, []byte(testutil.CNCIStartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = netAgent.GetCmdChanResult(netAgentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStop(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STOP)

	_, err := controller.Ssntp.SendCommand(ssntp.STOP, []byte(testutil.StopYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopFailure(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STOP)

	controllerErrorCh := controller.AddErrorChan(ssntp.StopFailure)
	fmt.Printf("Expecting controller to note: \"%s\"\n", ssntp.StopFailure)

	agent.StopFail = true
	agent.StopFailReason = payloads.StopNoInstance
	defer func() {
		agent.StopFail = false
		agent.StopFailReason = ""
	}()

	go controller.Ssntp.SendCommand(ssntp.STOP, []byte(testutil.StopYaml))

	_, err := agent.GetCmdChanResult(agentCh, ssntp.STOP)
	if err == nil { // agent will process the STOP and does error
		t.Fatal(err)
	}

	_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.StopFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRestart(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.RESTART)

	_, err := controller.Ssntp.SendCommand(ssntp.RESTART, []byte(testutil.RestartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRestartFailure(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.RESTART)

	controllerErrorCh := controller.AddErrorChan(ssntp.RestartFailure)
	fmt.Printf("Expecting controller to note: \"%s\"\n", ssntp.RestartFailure)

	agent.RestartFail = true
	agent.RestartFailReason = payloads.RestartNoInstance
	defer func() {
		agent.RestartFail = false
		agent.RestartFailReason = ""
	}()

	go controller.Ssntp.SendCommand(ssntp.RESTART, []byte(testutil.RestartYaml))

	_, err := agent.GetCmdChanResult(agentCh, ssntp.RESTART)
	if err == nil { // agent will process the RESTART and does error
		t.Fatal(err)
	}

	_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.RestartFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func doDelete(fail bool) error {
	agentCh := agent.AddCmdChan(ssntp.DELETE)

	var controllerErrorCh chan testutil.Result

	if fail == true {
		controllerErrorCh = controller.AddErrorChan(ssntp.DeleteFailure)
		fmt.Printf("Expecting controller to note: \"%s\"\n", ssntp.DeleteFailure)

		agent.DeleteFail = true
		agent.DeleteFailReason = payloads.DeleteNoInstance

		defer func() {
			agent.DeleteFail = false
			agent.DeleteFailReason = ""
		}()
	}

	go controller.Ssntp.SendCommand(ssntp.DELETE, []byte(testutil.DeleteYaml))

	_, err := agent.GetCmdChanResult(agentCh, ssntp.DELETE)
	if fail == false && err != nil { // agent unexpected fail
		return err
	}

	if fail == true {
		if err == nil { // agent unexpected success
			return err
		}
		_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.DeleteFailure)
		if err != nil {
			return err
		}
	}

	return nil
}

func propagateInstanceDeleted() error {
	agentCh := agent.AddEventChan(ssntp.InstanceDeleted)
	controllerCh := controller.AddEventChan(ssntp.InstanceDeleted)

	go agent.SendDeleteEvent(testutil.InstanceUUID)

	_, err := agent.GetEventChanResult(agentCh, ssntp.InstanceDeleted)
	if err != nil {
		return err
	}
	_, err = controller.GetEventChanResult(controllerCh, ssntp.InstanceDeleted)
	if err != nil {
		return err
	}
	return nil
}

func TestDelete(t *testing.T) {
	fail := false

	err := doDelete(fail)
	if err != nil {
		t.Fatal(err)
	}

	err = propagateInstanceDeleted()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFailure(t *testing.T) {
	fail := true

	err := doDelete(fail)
	if err != nil {
		t.Fatal(err)
	}
}

func stopServer() error {
	controllerCh := controller.AddEventChan(ssntp.NodeDisconnected)
	netAgentCh := netAgent.AddEventChan(ssntp.NodeDisconnected)
	agentCh := agent.AddEventChan(ssntp.NodeDisconnected)

	go server.ssntp.Stop()

	_, err := controller.GetEventChanResult(controllerCh, ssntp.NodeDisconnected)
	if err != nil {
		return err
	}
	_, err = netAgent.GetEventChanResult(netAgentCh, ssntp.NodeDisconnected)
	if err != nil {
		return err
	}
	_, err = agent.GetEventChanResult(agentCh, ssntp.NodeDisconnected)
	if err != nil {
		return err
	}
	return nil
}

func restartServer() error {
	controllerCh := controller.AddEventChan(ssntp.NodeConnected)
	netAgentCh := netAgent.AddEventChan(ssntp.NodeConnected)
	agentCh := agent.AddEventChan(ssntp.NodeConnected)
	cnciAgentCh := cnciAgent.AddEventChan(ssntp.NodeConnected)

	server = configSchedulerServer()
	if server == nil {
		return errors.New("unable to configure scheduler")
	}
	go server.ssntp.Serve(server.config, server)
	//go heartBeatLoop(server)  ...handy for debugging

	if controller != nil {
		_, err := controller.GetEventChanResult(controllerCh, ssntp.NodeConnected)
		if err != nil {
			return err
		}
	}
	if netAgent != nil {
		_, err := netAgent.GetEventChanResult(netAgentCh, ssntp.NodeConnected)
		if err != nil {
			return err
		}
	}
	if agent != nil {
		_, err := agent.GetEventChanResult(agentCh, ssntp.NodeConnected)
		if err != nil {
			return err
		}
	}
	if cnciAgent != nil {
		_, err := cnciAgent.GetEventChanResult(cnciAgentCh, ssntp.NodeConnected)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestReconnects(t *testing.T) {
	err := stopServer()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	err = restartServer()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTenantAdded(t *testing.T) {
	cnciAgentCh := cnciAgent.AddEventChan(ssntp.TenantAdded)

	go agent.SendTenantAddedEvent()

	_, err := cnciAgent.GetEventChanResult(cnciAgentCh, ssntp.TenantAdded)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTenantRemoved(t *testing.T) {
	cnciAgentCh := cnciAgent.AddEventChan(ssntp.TenantRemoved)

	go agent.SendTenantRemovedEvent()

	_, err := cnciAgent.GetEventChanResult(cnciAgentCh, ssntp.TenantRemoved)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublicIPAssigned(t *testing.T) {
	controllerCh := controller.AddEventChan(ssntp.PublicIPAssigned)

	go cnciAgent.SendPublicIPAssignedEvent()

	_, err := controller.GetEventChanResult(controllerCh, ssntp.PublicIPAssigned)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublicIPUnassigned(t *testing.T) {
	controllerCh := controller.AddEventChan(ssntp.PublicIPUnassigned)

	go cnciAgent.SendPublicIPUnassignedEvent()

	_, err := controller.GetEventChanResult(controllerCh, ssntp.PublicIPUnassigned)
	if err != nil {
		t.Fatal(err)
	}
}

func waitForController(uuid string) {
	for {
		server.controllerMutex.Lock()
		c := server.controllerMap[uuid]
		server.controllerMutex.Unlock()

		if c == nil {
			fmt.Printf("awaiting controller %s\n", uuid)
			time.Sleep(50 * time.Millisecond)
		} else {
			return
		}
	}
}
func waitForAgent(uuid string, status *ssntp.Status) {
	for {
		server.cnMutex.Lock()
		cn := server.cnMap[uuid]
		cn.mutex.Lock()
		server.cnMutex.Unlock()

		if cn == nil {
			fmt.Printf("awaiting agent %s\n", uuid)
			time.Sleep(50 * time.Millisecond)
		} else if status != nil && *status != cn.status {
			fmt.Printf("awaiting agent %s in state %s\n", uuid, status.String())
			time.Sleep(50 * time.Millisecond)
		} else {
			cn.mutex.Unlock()
			return
		}
		cn.mutex.Unlock()
	}
}
func waitForNetAgent(uuid string, status *ssntp.Status) {
	for {
		server.nnMutex.Lock()
		nn := server.nnMap[uuid]
		nn.mutex.Lock()
		server.nnMutex.Unlock()

		if nn == nil {
			fmt.Printf("awaiting netagent %s\n", uuid)
			time.Sleep(50 * time.Millisecond)
		} else if status != nil && *status != nn.status {
			fmt.Printf("awaiting netagent %s in state %s\n", uuid, status.String())
			time.Sleep(50 * time.Millisecond)
		} else {
			nn.mutex.Unlock()
			return
		}
		nn.mutex.Unlock()
	}
}

func ssntpTestsSetup() error {
	var err error

	// start server
	server = configSchedulerServer()
	if server == nil {
		return errors.New("unable to configure scheduler")
	}
	go server.ssntp.Serve(server.config, server)
	//go heartBeatLoop(server)  ...handy for debugging

	// start controller
	controllerUUID := uuid.Generate().String()
	controller, err = testutil.NewSsntpTestControllerConnection("Controller Client", controllerUUID)
	if err != nil {
		return err
	}

	// start agent
	agent, err = testutil.NewSsntpTestClientConnection("AGENT Client", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		return err
	}

	// start netagent
	netAgent, err = testutil.NewSsntpTestClientConnection("NETAGENT Client", ssntp.NETAGENT, testutil.NetAgentUUID)
	if err != nil {
		return err
	}

	// insure the three clients are connected:
	waitForController(controllerUUID)
	waitForAgent(testutil.AgentUUID, nil)
	waitForNetAgent(testutil.NetAgentUUID, nil)

	return nil
}

func ssntpTestsTeardown() {
	// stop everybody
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		controller.Shutdown()
		wg.Done()
	}()

	go func() {
		netAgent.Shutdown()
		wg.Done()
	}()

	go func() {
		agent.Shutdown()
		wg.Done()
	}()

	fmt.Println("Awaiting clients' shutdown")
	wg.Wait()
	fmt.Println("Got clients' shutdown")
	fmt.Println("Awaiting server shutdown")
	server.ssntp.Stop()
	fmt.Println("Got server shutdown")
}
