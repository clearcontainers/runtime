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

package testutil_test

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/uuid"
	. "github.com/01org/ciao/testutil"
)

var server *SsntpTestServer
var controller *SsntpTestController
var agent *SsntpTestClient
var netAgent *SsntpTestClient
var cnciAgent *SsntpTestClient

func TestSendAgentStatus(t *testing.T) {
	serverCh := server.AddStatusChan(ssntp.READY)

	go agent.SendStatus(16384, 16384, PartialComputeNetworks)

	_, err := server.GetStatusChanResult(serverCh, ssntp.READY)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendNetAgentStatus(t *testing.T) {
	serverCh := server.AddStatusChan(ssntp.READY)

	go netAgent.SendStatus(16384, 16384, MultipleComputeNetworks)

	_, err := server.GetStatusChanResult(serverCh, ssntp.READY)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCNCIStart(t *testing.T) {
	serverCh := server.AddCmdChan(ssntp.START)
	netAgentCh := netAgent.AddCmdChan(ssntp.START)

	go controller.Ssntp.SendCommand(ssntp.START, []byte(CNCIStartYaml))

	_, err := server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = netAgent.GetCmdChanResult(netAgentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}

	serverCh = server.AddEventChan(ssntp.ConcentratorInstanceAdded)
	controllerCh := controller.AddEventChan(ssntp.ConcentratorInstanceAdded)

	// start CNCI agent
	cnciAgent, err = NewSsntpTestClientConnection("CNCI Client", ssntp.CNCIAGENT, CNCIUUID)
	if err != nil {
		t.Fatal(err)
	}

	cnciAgent.SendConcentratorAddedEvent(CNCIInstanceUUID, TenantUUID, CNCIIP, CNCIMAC)

	_, err = server.GetEventChanResult(serverCh, ssntp.ConcentratorInstanceAdded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetEventChanResult(controllerCh, ssntp.ConcentratorInstanceAdded)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStart(t *testing.T) {
	serverCh := server.AddCmdChan(ssntp.START)
	agentCh := agent.AddCmdChan(ssntp.START)

	go controller.Ssntp.SendCommand(ssntp.START, []byte(StartYaml))

	_, err := server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartFailure(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)
	serverCh := server.AddCmdChan(ssntp.START)

	serverErrorCh := server.AddErrorChan(ssntp.StartFailure)
	controllerErrorCh := controller.AddErrorChan(ssntp.StartFailure)
	fmt.Fprintf(os.Stderr, "Expecting server and controller to note: \"%s\"\n", ssntp.StartFailure)

	agent.StartFail = true
	agent.StartFailReason = payloads.FullCloud
	defer func() {
		agent.StartFail = false
		agent.StartFailReason = ""
	}()

	go controller.Ssntp.SendCommand(ssntp.START, []byte(StartYaml))

	_, err := server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil { // server sees the START on its way down to agent
		t.Fatal(err)
	}
	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err == nil { // agent will process the START and does error
		t.Fatal(err)
	}

	_, err = server.GetErrorChanResult(serverErrorCh, ssntp.StartFailure)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.StartFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendStats(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STATS)
	serverCh := server.AddCmdChan(ssntp.STATS)
	controllerCh := controller.AddCmdChan(ssntp.STATS)

	go agent.SendStatsCmd()

	_, err := agent.GetCmdChanResult(agentCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.STATS)
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
	serverCh := server.AddCmdChan(ssntp.START)

	traceConfig := &ssntp.TraceConfig{
		PathTrace: true,
		Start:     time.Now(),
		Label:     []byte("testutilTracedSTART"),
	}

	_, err := controller.Ssntp.SendTracedCommand(ssntp.START, []byte(StartYaml), traceConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendTrace(t *testing.T) {
	agentCh := agent.AddEventChan(ssntp.TraceReport)
	serverCh := server.AddEventChan(ssntp.TraceReport)

	go agent.SendTrace()

	_, err := agent.GetEventChanResult(agentCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetEventChanResult(serverCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartCNCI(t *testing.T) {
	netAgentCh := netAgent.AddCmdChan(ssntp.START)
	serverCh := server.AddCmdChan(ssntp.START)

	_, err := controller.Ssntp.SendCommand(ssntp.START, []byte(CNCIStartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = netAgent.GetCmdChanResult(netAgentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStop(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STOP)
	serverCh := server.AddCmdChan(ssntp.STOP)

	_, err := controller.Ssntp.SendCommand(ssntp.STOP, []byte(StopYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopFailure(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.STOP)
	serverCh := server.AddCmdChan(ssntp.STOP)

	serverErrorCh := server.AddErrorChan(ssntp.StopFailure)
	controllerErrorCh := controller.AddErrorChan(ssntp.StopFailure)
	fmt.Fprintf(os.Stderr, "Expecting server and controller to note: \"%s\"\n", ssntp.StopFailure)

	agent.StopFail = true
	agent.StopFailReason = payloads.StopNoInstance
	defer func() {
		agent.StopFail = false
		agent.StopFailReason = ""
	}()

	go controller.Ssntp.SendCommand(ssntp.STOP, []byte(StopYaml))

	_, err := server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil { // server sees the STOP on its way down to agent
		t.Fatal(err)
	}
	_, err = agent.GetCmdChanResult(agentCh, ssntp.STOP)
	if err == nil { // agent will process the STOP and does error
		t.Fatal(err)
	}

	_, err = server.GetErrorChanResult(serverErrorCh, ssntp.StopFailure)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.StopFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRestart(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.RESTART)
	serverCh := server.AddCmdChan(ssntp.RESTART)

	_, err := controller.Ssntp.SendCommand(ssntp.RESTART, []byte(RestartYaml))
	if err != nil {
		t.Fatal(err)
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRestartFailure(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.RESTART)
	serverCh := server.AddCmdChan(ssntp.RESTART)

	serverErrorCh := server.AddErrorChan(ssntp.RestartFailure)
	controllerErrorCh := controller.AddErrorChan(ssntp.RestartFailure)
	fmt.Fprintf(os.Stderr, "Expecting server and controller to note: \"%s\"\n", ssntp.RestartFailure)

	agent.RestartFail = true
	agent.RestartFailReason = payloads.RestartNoInstance
	defer func() {
		agent.RestartFail = false
		agent.RestartFailReason = ""
	}()

	go controller.Ssntp.SendCommand(ssntp.RESTART, []byte(RestartYaml))

	_, err := server.GetCmdChanResult(serverCh, ssntp.RESTART)
	if err != nil { // server sees the RESTART on its way down to agent
		t.Fatal(err)
	}
	_, err = agent.GetCmdChanResult(agentCh, ssntp.RESTART)
	if err == nil { // agent will process the RESTART and does error
		t.Fatal(err)
	}

	_, err = server.GetErrorChanResult(serverErrorCh, ssntp.RestartFailure)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.RestartFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func doDelete(fail bool) error {
	agentCh := agent.AddCmdChan(ssntp.DELETE)
	serverCh := server.AddCmdChan(ssntp.DELETE)

	var serverErrorCh chan Result
	var controllerErrorCh chan Result

	if fail == true {
		serverErrorCh = server.AddErrorChan(ssntp.DeleteFailure)
		controllerErrorCh = controller.AddErrorChan(ssntp.DeleteFailure)
		fmt.Fprintf(os.Stderr, "Expecting server and controller to note: \"%s\"\n", ssntp.DeleteFailure)

		agent.DeleteFail = true
		agent.DeleteFailReason = payloads.DeleteNoInstance

		defer func() {
			agent.DeleteFail = false
			agent.DeleteFailReason = ""
		}()
	}

	go controller.Ssntp.SendCommand(ssntp.DELETE, []byte(DeleteYaml))

	_, err := server.GetCmdChanResult(serverCh, ssntp.DELETE)
	if err != nil { // server sees the DELETE on its way down to agent
		return err
	}
	_, err = agent.GetCmdChanResult(agentCh, ssntp.DELETE)
	if fail == false && err != nil { // agent unexpected fail
		return err
	}

	if fail == true {
		if err == nil { // agent unexpected success
			return err
		}
		_, err = server.GetErrorChanResult(serverErrorCh, ssntp.DeleteFailure)
		if err != nil {
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
	serverCh := server.AddEventChan(ssntp.InstanceDeleted)
	controllerCh := controller.AddEventChan(ssntp.InstanceDeleted)

	go agent.SendDeleteEvent(InstanceUUID)

	_, err := agent.GetEventChanResult(agentCh, ssntp.InstanceDeleted)
	if err != nil {
		return err
	}
	_, err = server.GetEventChanResult(serverCh, ssntp.InstanceDeleted)
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

func doAttachVolume(fail bool) error {
	agentCh := agent.AddCmdChan(ssntp.AttachVolume)
	serverCh := server.AddCmdChan(ssntp.AttachVolume)

	var serverErrorCh chan Result
	var controllerErrorCh chan Result

	if fail == true {
		serverErrorCh = server.AddErrorChan(ssntp.AttachVolumeFailure)
		controllerErrorCh = controller.AddErrorChan(ssntp.AttachVolumeFailure)
		fmt.Fprintf(os.Stderr, "Expecting server and controller to note: \"%s\"\n", ssntp.AttachVolumeFailure)

		agent.AttachFail = true
		agent.AttachVolumeFailReason = payloads.AttachVolumeAlreadyAttached

		defer func() {
			agent.AttachFail = false
			agent.AttachVolumeFailReason = ""
		}()
	}

	go controller.Ssntp.SendCommand(ssntp.AttachVolume, []byte(AttachVolumeYaml))
	_, err := server.GetCmdChanResult(serverCh, ssntp.AttachVolume)
	if err != nil { // server sees the AttachVolume on its way down to agent
		return err
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.AttachVolume)
	if fail == false && err != nil { // agent unexpected fail
		return err
	}

	if fail == true {
		if err == nil { // agent unexpected success
			return errors.New("Success when Failure expected")
		}
		_, err = server.GetErrorChanResult(serverErrorCh, ssntp.AttachVolumeFailure)
		if err != nil {
			return err
		}
		_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.AttachVolumeFailure)
		if err != nil {
			return err
		}
	}

	return err
}

func TestAttachVolume(t *testing.T) {
	fail := false

	err := doAttachVolume(fail)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAttachVolumeFailure(t *testing.T) {
	fail := true

	err := doAttachVolume(fail)
	if err != nil {
		t.Fatal(err)
	}
}

func doDetachVolume(fail bool) error {
	agentCh := agent.AddCmdChan(ssntp.DetachVolume)
	serverCh := server.AddCmdChan(ssntp.DetachVolume)

	var serverErrorCh chan Result
	var controllerErrorCh chan Result

	if fail == true {
		serverErrorCh = server.AddErrorChan(ssntp.DetachVolumeFailure)
		controllerErrorCh = controller.AddErrorChan(ssntp.DetachVolumeFailure)
		fmt.Fprintf(os.Stderr, "Expecting server and controller to note: \"%s\"\n", ssntp.DetachVolumeFailure)

		agent.DetachFail = true
		agent.DetachVolumeFailReason = payloads.DetachVolumeNotAttached

		defer func() {
			agent.DetachFail = false
			agent.DetachVolumeFailReason = ""
		}()
	}

	go controller.Ssntp.SendCommand(ssntp.DetachVolume, []byte(DetachVolumeYaml))
	_, err := server.GetCmdChanResult(serverCh, ssntp.DetachVolume)
	if err != nil { // server sees the DetachVolume on its way down to agent
		return err
	}

	_, err = agent.GetCmdChanResult(agentCh, ssntp.DetachVolume)
	if fail == false && err != nil { // agent unexpected fail
		return err
	}

	if fail == true {
		if err == nil { // agent unexpected success
			return errors.New("Success when Failure expected")
		}
		_, err = server.GetErrorChanResult(serverErrorCh, ssntp.DetachVolumeFailure)
		if err != nil {
			return err
		}
		_, err = controller.GetErrorChanResult(controllerErrorCh, ssntp.DetachVolumeFailure)
		if err != nil {
			return err
		}
	}

	return err
}

func TestDetachVolume(t *testing.T) {
	fail := false

	err := doDetachVolume(fail)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDetachVolumeFailure(t *testing.T) {
	fail := true

	err := doDetachVolume(fail)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTenantAdded(t *testing.T) {
	serverCh := server.AddEventChan(ssntp.TenantAdded)
	cnciAgentCh := cnciAgent.AddEventChan(ssntp.TenantAdded)

	go agent.SendTenantAddedEvent()

	_, err := server.GetEventChanResult(serverCh, ssntp.TenantAdded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cnciAgent.GetEventChanResult(cnciAgentCh, ssntp.TenantAdded)
	if err != nil {
		t.Fatal(err)
	}
}

func stopServer() error {
	controllerCh := controller.AddEventChan(ssntp.NodeDisconnected)
	netAgentCh := netAgent.AddEventChan(ssntp.NodeDisconnected)
	agentCh := agent.AddEventChan(ssntp.NodeDisconnected)

	go server.Shutdown()

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

	server = StartTestServer()

	//MUST be after StartTestServer becase the channels are initialized on start
	serverCh := server.AddEventChan(ssntp.NodeConnected)

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
	_, err := server.GetEventChanResult(serverCh, ssntp.NodeConnected)
	if err != nil {
		return err
	}
	return nil
}

func TestReconnects(t *testing.T) {
	fmt.Fprintln(os.Stderr, "stopping server")
	err := stopServer()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	fmt.Fprintln(os.Stderr, "restarting server")
	err = restartServer()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTenantRemoved(t *testing.T) {
	serverCh := server.AddEventChan(ssntp.TenantRemoved)
	cnciAgentCh := cnciAgent.AddEventChan(ssntp.TenantRemoved)

	go agent.SendTenantRemovedEvent()

	_, err := server.GetEventChanResult(serverCh, ssntp.TenantRemoved)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cnciAgent.GetEventChanResult(cnciAgentCh, ssntp.TenantRemoved)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublicIPAssigned(t *testing.T) {
	serverCh := server.AddEventChan(ssntp.PublicIPAssigned)
	controllerCh := controller.AddEventChan(ssntp.PublicIPAssigned)

	go cnciAgent.SendPublicIPAssignedEvent()

	_, err := server.GetEventChanResult(serverCh, ssntp.PublicIPAssigned)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.GetEventChanResult(controllerCh, ssntp.PublicIPAssigned)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	var err error

	// anything that uses ssntp must call flag.Parse first
	// due to glog.
	flag.Parse()

	// start server
	server = StartTestServer()

	// start controller
	controllerUUID := uuid.Generate().String()
	controller, err = NewSsntpTestControllerConnection("Controller Client", controllerUUID)
	if err != nil {
		os.Exit(1)
	}

	// start agent
	agent, err = NewSsntpTestClientConnection("AGENT Client", ssntp.AGENT, AgentUUID)
	if err != nil {
		os.Exit(1)
	}

	// start netagent
	netAgent, err = NewSsntpTestClientConnection("NETAGENT Client", ssntp.NETAGENT, NetAgentUUID)
	if err != nil {
		os.Exit(1)
	}

	status := m.Run()

	// stop everybody
	time.Sleep(1 * time.Second)
	controller.Shutdown()

	time.Sleep(1 * time.Second)
	netAgent.Shutdown()

	time.Sleep(1 * time.Second)
	agent.Shutdown()

	time.Sleep(1 * time.Second)
	server.Shutdown()

	os.Exit(status)
}
