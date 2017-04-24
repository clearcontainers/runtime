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

package testutil

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
)

// SsntpTestServer is global state for the testutil SSNTP server
type SsntpTestServer struct {
	Ssntp ssntp.Server

	clients        []string
	clientsLock    *sync.Mutex
	netClients     []string
	netClientsLock *sync.Mutex

	CmdChans        map[ssntp.Command]chan Result
	CmdChansLock    *sync.Mutex
	EventChans      map[ssntp.Event]chan Result
	EventChansLock  *sync.Mutex
	ErrorChans      map[ssntp.Error]chan Result
	ErrorChansLock  *sync.Mutex
	StatusChans     map[ssntp.Status]chan Result
	StatusChansLock *sync.Mutex
}

// AddCmdChan adds an ssntp.Command to the SsntpTestServer command channel
func (server *SsntpTestServer) AddCmdChan(cmd ssntp.Command) chan Result {
	c := make(chan Result)

	server.CmdChansLock.Lock()
	server.CmdChans[cmd] = c
	server.CmdChansLock.Unlock()

	return c
}

// GetCmdChanResult gets a Result from the SsntpTestServer command channel
func (server *SsntpTestServer) GetCmdChanResult(c chan Result, cmd ssntp.Command) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Server error on %s command: %s", cmd, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for server %s command result", cmd)
	}

	return result, err
}

// SendResultAndDelCmdChan deletes an ssntp.Command from the SsntpTestServer command channel
func (server *SsntpTestServer) SendResultAndDelCmdChan(cmd ssntp.Command, result Result) {
	server.CmdChansLock.Lock()
	c, ok := server.CmdChans[cmd]
	if ok {
		delete(server.CmdChans, cmd)
		server.CmdChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	server.CmdChansLock.Unlock()
}

// AddEventChan adds an ssntp.Event to the SsntpTestServer event channel
func (server *SsntpTestServer) AddEventChan(evt ssntp.Event) chan Result {
	c := make(chan Result)

	server.EventChansLock.Lock()
	server.EventChans[evt] = c
	server.EventChansLock.Unlock()

	return c
}

// GetEventChanResult gets a Result from the SsntpTestServer event channel
func (server *SsntpTestServer) GetEventChanResult(c chan Result, evt ssntp.Event) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Server error handling %s event: %s", evt, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for server %s event result", evt)
	}

	return result, err
}

// SendResultAndDelEventChan deletes an ssntp.Event from the SsntpTestServer event channel
func (server *SsntpTestServer) SendResultAndDelEventChan(evt ssntp.Event, result Result) {
	server.EventChansLock.Lock()
	c, ok := server.EventChans[evt]
	if ok {
		delete(server.EventChans, evt)
		server.EventChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	server.EventChansLock.Unlock()
}

// AddErrorChan adds an ssntp.Error to the SsntpTestServer error channel
func (server *SsntpTestServer) AddErrorChan(error ssntp.Error) chan Result {
	c := make(chan Result)

	server.ErrorChansLock.Lock()
	server.ErrorChans[error] = c
	server.ErrorChansLock.Unlock()

	return c
}

// GetErrorChanResult gets a CmdResult from the SsntpTestServer error channel
func (server *SsntpTestServer) GetErrorChanResult(c chan Result, error ssntp.Error) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Server error handling %s error: %s", error, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for server %s error result", error)
	}

	return result, err
}

// SendResultAndDelErrorChan deletes an ssntp.Error from the SsntpTestServer error channel
func (server *SsntpTestServer) SendResultAndDelErrorChan(error ssntp.Error, result Result) {
	server.ErrorChansLock.Lock()
	c, ok := server.ErrorChans[error]
	if ok {
		delete(server.ErrorChans, error)
		server.ErrorChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	server.ErrorChansLock.Unlock()
}

// AddStatusChan adds an ssntp.Status to the SsntpTestServer status channel
func (server *SsntpTestServer) AddStatusChan(status ssntp.Status) chan Result {
	c := make(chan Result)

	server.StatusChansLock.Lock()
	server.StatusChans[status] = c
	server.StatusChansLock.Unlock()

	return c
}

// GetStatusChanResult gets a Result from the SsntpTestServer status channel
func (server *SsntpTestServer) GetStatusChanResult(c chan Result, status ssntp.Status) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Server error handling %s status: %s", status, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for server %s status result", status)
	}

	return result, err
}

// SendResultAndDelStatusChan deletes an ssntp.Status from the SsntpTestServer status channel
func (server *SsntpTestServer) SendResultAndDelStatusChan(status ssntp.Status, result Result) {
	server.StatusChansLock.Lock()
	c, ok := server.StatusChans[status]
	if ok {
		delete(server.StatusChans, status)
		server.StatusChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	server.StatusChansLock.Unlock()
}

func openServerChans(server *SsntpTestServer) {
	server.CmdChansLock.Lock()
	server.CmdChans = make(map[ssntp.Command]chan Result)
	server.CmdChansLock.Unlock()

	server.EventChansLock.Lock()
	server.EventChans = make(map[ssntp.Event]chan Result)
	server.EventChansLock.Unlock()

	server.ErrorChansLock.Lock()
	server.ErrorChans = make(map[ssntp.Error]chan Result)
	server.ErrorChansLock.Unlock()

	server.StatusChansLock.Lock()
	server.StatusChans = make(map[ssntp.Status]chan Result)
	server.StatusChansLock.Unlock()
}

func closeServerChans(server *SsntpTestServer) {
	server.CmdChansLock.Lock()
	for k := range server.CmdChans {
		close(server.CmdChans[k])
		delete(server.CmdChans, k)
	}
	server.CmdChansLock.Unlock()

	server.EventChansLock.Lock()
	for k := range server.EventChans {
		close(server.EventChans[k])
		delete(server.EventChans, k)
	}
	server.EventChansLock.Unlock()

	server.ErrorChansLock.Lock()
	for k := range server.ErrorChans {
		close(server.ErrorChans[k])
		delete(server.ErrorChans, k)
	}
	server.ErrorChansLock.Unlock()

	server.StatusChansLock.Lock()
	for k := range server.StatusChans {
		close(server.StatusChans[k])
		delete(server.StatusChans, k)
	}
	server.StatusChansLock.Unlock()
}

// ConnectNotify implements an SSNTP ConnectNotify callback for SsntpTestServer
func (server *SsntpTestServer) ConnectNotify(uuid string, role ssntp.Role) {
	var result Result

	switch role {
	case ssntp.AGENT:
		server.clientsLock.Lock()
		defer server.clientsLock.Unlock()
		server.clients = append(server.clients, uuid)

	case ssntp.NETAGENT:
		server.netClientsLock.Lock()
		defer server.netClientsLock.Unlock()
		server.netClients = append(server.netClients, uuid)
	}

	go server.SendResultAndDelEventChan(ssntp.NodeConnected, result)
}

// DisconnectNotify implements an SSNTP DisconnectNotify callback for SsntpTestServer
func (server *SsntpTestServer) DisconnectNotify(uuid string, role ssntp.Role) {
	var result Result

	switch role {
	case ssntp.AGENT:
		server.clientsLock.Lock()
		for index := range server.clients {
			if server.clients[index] == uuid {
				server.clients = append(server.clients[:index], server.clients[index+1:]...)
				break
			}
		}
		server.clientsLock.Unlock()

	case ssntp.NETAGENT:
		server.netClientsLock.Lock()
		for index := range server.netClients {
			if server.netClients[index] == uuid {
				server.netClients = append(server.netClients[:index], server.netClients[index+1:]...)
				break
			}
		}
		server.netClientsLock.Unlock()
	}

	go server.SendResultAndDelEventChan(ssntp.NodeDisconnected, result)
}

// StatusNotify is an SSNTP callback stub for SsntpTestServer
func (server *SsntpTestServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
	var result Result

	switch status {
	case ssntp.READY:
		fmt.Fprintf(os.Stderr, "server received READY from node %s\n", uuid)
	default:
		fmt.Fprintf(os.Stderr, "server unhandled status frame from node %s\n", uuid)
	}

	go server.SendResultAndDelStatusChan(status, result)
}

func getAttachVolumeResult(payload []byte, result *Result) {
	var volCmd payloads.AttachVolume

	err := yaml.Unmarshal(payload, &volCmd)
	result.Err = err
	if err == nil {
		result.NodeUUID = volCmd.Attach.WorkloadAgentUUID
		result.InstanceUUID = volCmd.Attach.InstanceUUID
		result.VolumeUUID = volCmd.Attach.VolumeUUID
	}
}

func getDetachVolumeResult(payload []byte, result *Result) {
	var volCmd payloads.DetachVolume

	err := yaml.Unmarshal(payload, &volCmd)
	result.Err = err
	if err == nil {
		result.NodeUUID = volCmd.Detach.WorkloadAgentUUID
		result.InstanceUUID = volCmd.Detach.InstanceUUID
		result.VolumeUUID = volCmd.Detach.VolumeUUID
	}
}

func getStartResults(payload []byte, result *Result) {
	var startCmd payloads.Start
	var nn bool

	err := yaml.Unmarshal(payload, &startCmd)
	result.Err = err
	if err == nil {
		resources := startCmd.Start.RequestedResources

		for i := range resources {
			if resources[i].Type == payloads.NetworkNode {
				nn = true
				break
			}
		}
		result.InstanceUUID = startCmd.Start.InstanceUUID
		result.TenantUUID = startCmd.Start.TenantUUID
		result.CNCI = nn
	}
}

// CommandNotify implements an SSNTP CommandNotify callback for SsntpTestServer
func (server *SsntpTestServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	var result Result

	payload := frame.Payload

	switch command {
	/*TODO:
	case CONNECT:
	case AssignPublicIP:
	case ReleasePublicIP:
	case CONFIGURE:
	*/
	case ssntp.START:
		getStartResults(payload, &result)

	case ssntp.DELETE:
		var delCmd payloads.Delete

		err := yaml.Unmarshal(payload, &delCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = delCmd.Delete.InstanceUUID
			server.Ssntp.SendCommand(delCmd.Delete.WorkloadAgentUUID, command, frame.Payload)
		}

	case ssntp.STOP:
		var stopCmd payloads.Stop

		err := yaml.Unmarshal(payload, &stopCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = stopCmd.Stop.InstanceUUID
			server.Ssntp.SendCommand(stopCmd.Stop.WorkloadAgentUUID, command, frame.Payload)
		}

	case ssntp.RESTART:
		var restartCmd payloads.Restart

		err := yaml.Unmarshal(payload, &restartCmd)
		result.Err = err
		if err == nil {
			result.InstanceUUID = restartCmd.Restart.InstanceUUID
			server.Ssntp.SendCommand(restartCmd.Restart.WorkloadAgentUUID, command, frame.Payload)
		}

	case ssntp.EVACUATE:
		var evacCmd payloads.Evacuate

		err := yaml.Unmarshal(payload, &evacCmd)
		result.Err = err
		if err == nil {
			result.NodeUUID = evacCmd.Evacuate.WorkloadAgentUUID
		}

	case ssntp.STATS:
		var statsCmd payloads.Stat

		err := yaml.Unmarshal(payload, &statsCmd)
		result.Err = err

	case ssntp.AttachVolume:
		getAttachVolumeResult(payload, &result)

	case ssntp.DetachVolume:
		getDetachVolumeResult(payload, &result)

	default:
		fmt.Fprintf(os.Stderr, "server unhandled command %s\n", command.String())
	}

	go server.SendResultAndDelCmdChan(command, result)
}

// EventNotify implements an SSNTP EventNotify callback for SsntpTestServer
func (server *SsntpTestServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
	var result Result

	payload := frame.Payload

	switch event {
	case ssntp.NodeConnected:
		//handled by ConnectNotify()
		return
	case ssntp.NodeDisconnected:
		//handled by DisconnectNotify()
		return
	case ssntp.TraceReport:
		var traceEvent payloads.Trace

		result.Err = yaml.Unmarshal(payload, &traceEvent)
	case ssntp.InstanceDeleted:
		var deleteEvent payloads.EventInstanceDeleted

		result.Err = yaml.Unmarshal(payload, &deleteEvent)
	case ssntp.ConcentratorInstanceAdded:
		// forward rule auto-sends to controllers
	case ssntp.TenantAdded:
		// forwards to CNCI via server.EventForward()
	case ssntp.TenantRemoved:
		// forwards to CNCI via server.EventForward()
	case ssntp.PublicIPAssigned:
		// forwards from CNCI Controller(s) via server.EventForward()
	default:
		fmt.Fprintf(os.Stderr, "server unhandled event %s\n", event.String())
	}

	go server.SendResultAndDelEventChan(event, result)
}

func getConcentratorUUID(event ssntp.Event, payload []byte) (string, error) {
	switch event {
	default:
		return "", fmt.Errorf("unsupported ssntp.Event type \"%s\"", event)
	case ssntp.TenantAdded:
		var ev payloads.EventTenantAdded
		err := yaml.Unmarshal(payload, &ev)
		return ev.TenantAdded.ConcentratorUUID, err
	case ssntp.TenantRemoved:
		var ev payloads.EventTenantRemoved
		err := yaml.Unmarshal(payload, &ev)
		return ev.TenantRemoved.ConcentratorUUID, err
	}
}

func fwdEventToCNCI(event ssntp.Event, payload []byte) (ssntp.ForwardDestination, error) {
	var dest ssntp.ForwardDestination

	concentratorUUID, err := getConcentratorUUID(event, payload)
	if err != nil || concentratorUUID == "" {
		dest.SetDecision(ssntp.Discard)
	}

	dest.AddRecipient(concentratorUUID)
	return dest, err
}

// EventForward implements and SSNTP EventForward callback for SsntpTestServer
func (server *SsntpTestServer) EventForward(uuid string, event ssntp.Event, frame *ssntp.Frame) ssntp.ForwardDestination {
	var err error
	var dest ssntp.ForwardDestination
	var result Result

	switch event {
	case ssntp.TenantAdded:
		fallthrough
	case ssntp.TenantRemoved:
		dest, err = fwdEventToCNCI(event, frame.Payload)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "server error parsing event yaml for forwarding")
		result.Err = err
	}

	go server.SendResultAndDelEventChan(event, result)

	return dest
}

// ErrorNotify is an SSNTP callback stub for SsntpTestServer
func (server *SsntpTestServer) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
	var result Result

	//payload := frame.Payload

	switch error {
	case ssntp.InvalidFrameType: //FIXME
		fallthrough

	case ssntp.StartFailure: //FIXME
		fallthrough

	case ssntp.StopFailure: //FIXME
		fallthrough

	case ssntp.ConnectionFailure: //FIXME
		fallthrough

	case ssntp.RestartFailure: //FIXME
		fallthrough

	case ssntp.DeleteFailure: //FIXME
		fallthrough

	case ssntp.ConnectionAborted: //FIXME
		fallthrough

	case ssntp.InvalidConfiguration: //FIXME
		fallthrough

	default:
		fmt.Fprintf(os.Stderr, "server unhandled error %s\n", error.String())
	}

	go server.SendResultAndDelErrorChan(error, result)
}

func (server *SsntpTestServer) handleStart(payload []byte) (dest ssntp.ForwardDestination) {
	var startCmd payloads.Start
	var nn bool

	err := yaml.Unmarshal(payload, &startCmd)

	if err != nil {
		return
	}

	resources := startCmd.Start.RequestedResources

	for i := range resources {
		if resources[i].Type == payloads.NetworkNode {
			nn = true
			break
		}
	}

	if nn {
		server.netClientsLock.Lock()
		defer server.netClientsLock.Unlock()
		if len(server.netClients) > 0 {
			index := rand.Intn(len(server.netClients))
			dest.AddRecipient(server.netClients[index])
		}
	} else {
		server.clientsLock.Lock()
		defer server.clientsLock.Unlock()
		if len(server.clients) > 0 {
			index := rand.Intn(len(server.clients))
			dest.AddRecipient(server.clients[index])
		}
	}

	return dest
}

func (server *SsntpTestServer) handleAttachVolume(payload []byte) ssntp.ForwardDestination {
	var cmd payloads.AttachVolume
	var dest ssntp.ForwardDestination

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		return dest
	}

	server.clientsLock.Lock()
	defer server.clientsLock.Unlock()

	for _, c := range server.clients {
		if c == cmd.Attach.WorkloadAgentUUID {
			dest.AddRecipient(c)
		}
	}

	return dest
}

func (server *SsntpTestServer) handleDetachVolume(payload []byte) ssntp.ForwardDestination {
	var cmd payloads.DetachVolume
	var dest ssntp.ForwardDestination

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		return dest
	}

	server.clientsLock.Lock()
	defer server.clientsLock.Unlock()

	for _, c := range server.clients {
		if c == cmd.Detach.WorkloadAgentUUID {
			dest.AddRecipient(c)
		}
	}

	return dest
}

// CommandForward implements an SSNTP CommandForward callback for SsntpTestServer
func (server *SsntpTestServer) CommandForward(uuid string, command ssntp.Command, frame *ssntp.Frame) (dest ssntp.ForwardDestination) {
	payload := frame.Payload

	switch command {
	case ssntp.START:
		dest = server.handleStart(payload)
	case ssntp.AttachVolume:
		dest = server.handleAttachVolume(payload)
	case ssntp.DetachVolume:
		dest = server.handleDetachVolume(payload)
	case ssntp.EVACUATE:
		fallthrough
	case ssntp.STOP:
		fallthrough
	case ssntp.DELETE:
		fallthrough
	case ssntp.RESTART:
		//TODO: dest, instanceUUID = sched.fwdCmdToComputeNode(command, payload)
	default:
		dest.SetDecision(ssntp.Discard)
	}

	return dest
}

// Shutdown shuts down the testutil.SsntpTestServer and cleans up state
func (server *SsntpTestServer) Shutdown() {
	closeServerChans(server)
	server.Ssntp.Stop()
}

// StartTestServer starts a go routine for based on a
// testutil.SsntpTestServer configuration with standard ssntp.FrameRorwardRules
func StartTestServer() *SsntpTestServer {
	server := new(SsntpTestServer)
	server.clientsLock = &sync.Mutex{}
	server.netClientsLock = &sync.Mutex{}

	server.CmdChansLock = &sync.Mutex{}
	server.EventChansLock = &sync.Mutex{}
	server.ErrorChansLock = &sync.Mutex{}
	server.StatusChansLock = &sync.Mutex{}
	openServerChans(server)

	serverConfig := ssntp.Config{
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(ssntp.SERVER),
		Log:    ssntp.Log,
		ForwardRules: []ssntp.FrameForwardRule{
			{ // all STATS commands go to all Controllers
				Operand: ssntp.STATS,
				Dest:    ssntp.Controller,
			},
			{ // all TraceReport events go to all Controllers
				Operand: ssntp.TraceReport,
				Dest:    ssntp.Controller,
			},
			{ // all InstanceDeleted events go to all Controllers
				Operand: ssntp.InstanceDeleted,
				Dest:    ssntp.Controller,
			},
			{ // all ConcentratorInstanceAdded events go to all Controllers
				Operand: ssntp.ConcentratorInstanceAdded,
				Dest:    ssntp.Controller,
			},
			{ // all StartFailure errors go to all Controllers
				Operand: ssntp.StartFailure,
				Dest:    ssntp.Controller,
			},
			{ // all StopFailure errors go to all Controllers
				Operand: ssntp.StopFailure,
				Dest:    ssntp.Controller,
			},
			{ // all RestartFailure errors go to all Controllers
				Operand: ssntp.RestartFailure,
				Dest:    ssntp.Controller,
			},
			{ // all DeleteFailure errors go to all Controllers
				Operand: ssntp.DeleteFailure,
				Dest:    ssntp.Controller,
			},
			{ // all VolumeAttachFailure errors go to all Controllers
				Operand: ssntp.AttachVolumeFailure,
				Dest:    ssntp.Controller,
			},
			{ // all VolumeDetachFailure errors go to all Controllers
				Operand: ssntp.DetachVolumeFailure,
				Dest:    ssntp.Controller,
			},
			{ // all PublicIPAssigned events go to all Controllers
				Operand: ssntp.PublicIPAssigned,
				Dest:    ssntp.Controller,
			},
			{ // all START command are processed by the Command forwarder
				Operand:        ssntp.START,
				CommandForward: server,
			},
			{ // all RESTART command are processed by the Command forwarder
				Operand:        ssntp.RESTART,
				CommandForward: server,
			},
			{ // all STOP command are processed by the Command forwarder
				Operand:        ssntp.STOP,
				CommandForward: server,
			},
			{ // all DELETE command are processed by the Command forwarder
				Operand:        ssntp.DELETE,
				CommandForward: server,
			},
			{ // all EVACUATE command are processed by the Command forwarder
				Operand:        ssntp.EVACUATE,
				CommandForward: server,
			},
			{ // all TenantAdded events are processed by the Event forwarder
				Operand:      ssntp.TenantAdded,
				EventForward: server,
			},
			{ // all TenantRemoved events are processed by the Event forwarder
				Operand:      ssntp.TenantRemoved,
				EventForward: server,
			},
			{ // all AttachVolume commands are processed by the Command forwarder
				Operand:        ssntp.AttachVolume,
				CommandForward: server,
			},
			{ // all DetachVolume commands are processed by the Command forwarder
				Operand:        ssntp.DetachVolume,
				CommandForward: server,
			},
		},
	}

	go server.Ssntp.Serve(&serverConfig, server)
	return server
}
