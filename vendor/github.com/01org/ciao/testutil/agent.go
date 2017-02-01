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
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
)

// SsntpTestClient is global state for the testutil SSNTP client worker
type SsntpTestClient struct {
	Ssntp                  ssntp.Client
	Name                   string
	instances              []payloads.InstanceStat
	instancesLock          *sync.Mutex
	ticker                 *time.Ticker
	UUID                   string
	Role                   ssntp.Role
	StartFail              bool
	StartFailReason        payloads.StartFailureReason
	StopFail               bool
	StopFailReason         payloads.StopFailureReason
	RestartFail            bool
	RestartFailReason      payloads.RestartFailureReason
	DeleteFail             bool
	DeleteFailReason       payloads.DeleteFailureReason
	AttachFail             bool
	AttachVolumeFailReason payloads.AttachVolumeFailureReason
	DetachFail             bool
	DetachVolumeFailReason payloads.DetachVolumeFailureReason
	traces                 []*ssntp.Frame
	tracesLock             *sync.Mutex

	CmdChans        map[ssntp.Command]chan Result
	CmdChansLock    *sync.Mutex
	EventChans      map[ssntp.Event]chan Result
	EventChansLock  *sync.Mutex
	ErrorChans      map[ssntp.Error]chan Result
	ErrorChansLock  *sync.Mutex
	StatusChans     map[ssntp.Status]chan Result
	StatusChansLock *sync.Mutex
}

// Shutdown shuts down the testutil.SsntpTestClient and cleans up state
func (client *SsntpTestClient) Shutdown() {
	closeClientChans(client)
	client.Ssntp.Close()
}

// NewSsntpTestClientConnection creates an SsntpTestClient and dials the server.
// Calling with a unique name parameter string for inclusion in the SsntpTestClient.Name
// field aides in debugging.  The role parameter is mandatory.  The uuid string
// parameter allows tests to specify a known uuid for simpler tests.
func NewSsntpTestClientConnection(name string, role ssntp.Role, uuid string) (*SsntpTestClient, error) {
	if role == ssntp.UNKNOWN {
		return nil, errors.New("no role specified")
	}
	if uuid == "" {
		return nil, errors.New("no uuid specified")
	}

	client := new(SsntpTestClient)
	client.Name = "Test " + role.String() + " " + name
	client.UUID = uuid
	client.Role = role
	client.StartFail = false
	client.CmdChansLock = &sync.Mutex{}
	client.EventChansLock = &sync.Mutex{}
	client.ErrorChansLock = &sync.Mutex{}
	client.StatusChansLock = &sync.Mutex{}
	openClientChans(client)
	client.instancesLock = &sync.Mutex{}
	client.tracesLock = &sync.Mutex{}

	config := &ssntp.Config{
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(role),
		Log:    ssntp.Log,
		UUID:   client.UUID,
	}

	if err := client.Ssntp.Dial(config, client); err != nil {
		return nil, err
	}
	return client, nil
}

// AddCmdChan adds an ssntp.Command to the SsntpTestClient command channel
func (client *SsntpTestClient) AddCmdChan(cmd ssntp.Command) chan Result {
	c := make(chan Result)

	client.CmdChansLock.Lock()
	client.CmdChans[cmd] = c
	client.CmdChansLock.Unlock()

	return c
}

// GetCmdChanResult gets a Result from the SsntpTestClient command channel
func (client *SsntpTestClient) GetCmdChanResult(c chan Result, cmd ssntp.Command) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Client error sending %s command: %s", cmd, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for client %s command result", cmd)
	}

	return result, err
}

// SendResultAndDelCmdChan deletes an ssntp.Command from the SsntpTestClient command channel
func (client *SsntpTestClient) SendResultAndDelCmdChan(cmd ssntp.Command, result Result) {
	client.CmdChansLock.Lock()
	c, ok := client.CmdChans[cmd]
	if ok {
		delete(client.CmdChans, cmd)
		client.CmdChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	client.CmdChansLock.Unlock()
}

// AddEventChan adds a ssntp.Event to the SsntpTestClient event channel
func (client *SsntpTestClient) AddEventChan(evt ssntp.Event) chan Result {
	c := make(chan Result)

	client.EventChansLock.Lock()
	client.EventChans[evt] = c
	client.EventChansLock.Unlock()

	return c
}

// GetEventChanResult gets a Result from the SsntpTestClient event channel
func (client *SsntpTestClient) GetEventChanResult(c chan Result, evt ssntp.Event) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Client error sending %s event: %s", evt, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for client %s event result", evt)
	}

	return result, err
}

// SendResultAndDelEventChan deletes an ssntp.Event from the SsntpTestClient event channel
func (client *SsntpTestClient) SendResultAndDelEventChan(evt ssntp.Event, result Result) {
	client.EventChansLock.Lock()
	c, ok := client.EventChans[evt]
	if ok {
		delete(client.EventChans, evt)
		client.EventChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	client.EventChansLock.Unlock()
}

// AddErrorChan adds a ssntp.Error to the SsntpTestClient error channel
func (client *SsntpTestClient) AddErrorChan(error ssntp.Error) chan Result {
	c := make(chan Result)

	client.ErrorChansLock.Lock()
	client.ErrorChans[error] = c
	client.ErrorChansLock.Unlock()

	return c
}

// GetErrorChanResult gets a Result from the SsntpTestClient error channel
func (client *SsntpTestClient) GetErrorChanResult(c chan Result, error ssntp.Error) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Client error sending %s error: %s", error, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for client %s error result", error)
	}

	return result, err
}

// SendResultAndDelErrorChan deletes an ssntp.Error from the SsntpTestClient error channel
func (client *SsntpTestClient) SendResultAndDelErrorChan(error ssntp.Error, result Result) {
	client.ErrorChansLock.Lock()
	c, ok := client.ErrorChans[error]
	if ok {
		delete(client.ErrorChans, error)
		client.ErrorChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	client.ErrorChansLock.Unlock()
}

// AddStatusChan adds an ssntp.Status to the SsntpTestClient status channel
func (client *SsntpTestClient) AddStatusChan(status ssntp.Status) chan Result {
	c := make(chan Result)

	client.StatusChansLock.Lock()
	client.StatusChans[status] = c
	client.StatusChansLock.Unlock()

	return c
}

// GetStatusChanResult gets a Result from the SsntpTestClient status channel
func (client *SsntpTestClient) GetStatusChanResult(c chan Result, status ssntp.Status) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Client error sending %s status: %s", status, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for client %s status result", status)
	}

	return result, err
}

// SendResultAndDelStatusChan deletes an ssntp.Status from the SsntpTestClient status channel
func (client *SsntpTestClient) SendResultAndDelStatusChan(status ssntp.Status, result Result) {
	client.StatusChansLock.Lock()
	c, ok := client.StatusChans[status]
	if ok {
		delete(client.StatusChans, status)
		client.StatusChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	client.StatusChansLock.Unlock()
}

func openClientChans(client *SsntpTestClient) {
	client.CmdChansLock.Lock()
	client.CmdChans = make(map[ssntp.Command]chan Result)
	client.CmdChansLock.Unlock()

	client.EventChansLock.Lock()
	client.EventChans = make(map[ssntp.Event]chan Result)
	client.EventChansLock.Unlock()

	client.ErrorChansLock.Lock()
	client.ErrorChans = make(map[ssntp.Error]chan Result)
	client.ErrorChansLock.Unlock()

	client.StatusChansLock.Lock()
	client.StatusChans = make(map[ssntp.Status]chan Result)
	client.StatusChansLock.Unlock()
}

func closeClientChans(client *SsntpTestClient) {
	client.CmdChansLock.Lock()
	for k := range client.CmdChans {
		close(client.CmdChans[k])
		delete(client.CmdChans, k)
	}
	client.CmdChansLock.Unlock()

	client.EventChansLock.Lock()
	for k := range client.EventChans {
		close(client.EventChans[k])
		delete(client.EventChans, k)
	}
	client.EventChansLock.Unlock()

	client.ErrorChansLock.Lock()
	for k := range client.ErrorChans {
		close(client.ErrorChans[k])
		delete(client.ErrorChans, k)
	}
	client.ErrorChansLock.Unlock()

	client.StatusChansLock.Lock()
	for k := range client.StatusChans {
		close(client.StatusChans[k])
		delete(client.StatusChans, k)
	}
	client.StatusChansLock.Unlock()
}

// ConnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestClient
func (client *SsntpTestClient) ConnectNotify() {
	var result Result

	go client.SendResultAndDelEventChan(ssntp.NodeConnected, result)
}

// DisconnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestClient
func (client *SsntpTestClient) DisconnectNotify() {
	var result Result

	go client.SendResultAndDelEventChan(ssntp.NodeDisconnected, result)
}

// StatusNotify implements the SSNTP client StatusNotify callback for SsntpTestClient
func (client *SsntpTestClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

func (client *SsntpTestClient) handleStart(payload []byte) Result {
	var result Result
	var cmd payloads.Start

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		result.Err = err
		return result
	}

	result.InstanceUUID = cmd.Start.InstanceUUID
	result.TenantUUID = cmd.Start.TenantUUID
	result.NodeUUID = client.UUID

	if client.Role.IsNetAgent() {
		result.CNCI = true
	}

	if client.StartFail == true {
		result.Err = errors.New(client.StartFailReason.String())
		client.sendStartFailure(cmd.Start.InstanceUUID, client.StartFailReason)
		go client.SendResultAndDelErrorChan(ssntp.StartFailure, result)
		return result
	}

	istat := payloads.InstanceStat{
		InstanceUUID:  cmd.Start.InstanceUUID,
		State:         payloads.Running,
		MemoryUsageMB: 0,
		DiskUsageMB:   0,
		CPUUsage:      0,
	}

	client.instancesLock.Lock()
	client.instances = append(client.instances, istat)
	client.instancesLock.Unlock()
	return result
}

func (client *SsntpTestClient) handleStop(payload []byte) Result {
	var result Result
	var cmd payloads.Stop

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		result.Err = err
		return result
	}

	if client.StopFail == true {
		result.Err = errors.New(client.StopFailReason.String())
		client.sendStopFailure(cmd.Stop.InstanceUUID, client.StopFailReason)
		go client.SendResultAndDelErrorChan(ssntp.StopFailure, result)
		return result
	}

	client.instancesLock.Lock()
	defer client.instancesLock.Unlock()
	for i := range client.instances {
		istat := client.instances[i]
		if istat.InstanceUUID == cmd.Stop.InstanceUUID {
			client.instances[i].State = payloads.Exited
		}
	}

	return result
}

func (client *SsntpTestClient) handleRestart(payload []byte) Result {
	var result Result
	var cmd payloads.Restart

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		result.Err = err
		return result
	}

	if client.RestartFail == true {
		result.Err = errors.New(client.RestartFailReason.String())
		client.sendRestartFailure(cmd.Restart.InstanceUUID, client.RestartFailReason)
		go client.SendResultAndDelErrorChan(ssntp.RestartFailure, result)
		return result
	}

	client.instancesLock.Lock()
	defer client.instancesLock.Unlock()
	for i := range client.instances {
		istat := client.instances[i]
		if istat.InstanceUUID == cmd.Restart.InstanceUUID {
			client.instances[i].State = payloads.Running
		}
	}

	return result
}

func (client *SsntpTestClient) handleDelete(payload []byte) Result {
	var result Result
	var cmd payloads.Delete

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		result.Err = err
		return result
	}

	if client.DeleteFail == true {
		result.Err = errors.New(client.DeleteFailReason.String())
		client.sendDeleteFailure(cmd.Delete.InstanceUUID, client.DeleteFailReason)
		go client.SendResultAndDelErrorChan(ssntp.DeleteFailure, result)
		return result
	}

	client.instancesLock.Lock()
	defer client.instancesLock.Unlock()
	for i := range client.instances {
		istat := client.instances[i]
		if istat.InstanceUUID == cmd.Delete.InstanceUUID {
			client.instances = append(client.instances[:i], client.instances[i+1:]...)
			break
		}
	}

	return result
}

func (client *SsntpTestClient) handleAttachVolume(payload []byte) Result {
	var result Result
	var cmd payloads.AttachVolume

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		result.Err = err
		return result
	}

	if client.AttachFail == true {
		result.Err = errors.New(client.AttachVolumeFailReason.String())
		client.sendAttachVolumeFailure(cmd.Attach.InstanceUUID, cmd.Attach.VolumeUUID, client.AttachVolumeFailReason)
		client.SendResultAndDelErrorChan(ssntp.AttachVolumeFailure, result)
		return result
	}

	// update statistics for volume
	client.instancesLock.Lock()
	for i, istat := range client.instances {
		if istat.InstanceUUID == cmd.Attach.InstanceUUID {
			client.instances[i].Volumes = append(istat.Volumes, cmd.Attach.VolumeUUID)
		}
	}
	client.instancesLock.Unlock()

	return result
}

func (client *SsntpTestClient) handleDetachVolume(payload []byte) Result {
	var result Result
	var cmd payloads.DetachVolume

	err := yaml.Unmarshal(payload, &cmd)
	if err != nil {
		result.Err = err
		return result
	}

	if client.DetachFail == true {
		result.Err = errors.New(client.DetachVolumeFailReason.String())
		client.sendDetachVolumeFailure(cmd.Detach.InstanceUUID, cmd.Detach.VolumeUUID, client.DetachVolumeFailReason)
		client.SendResultAndDelErrorChan(ssntp.DetachVolumeFailure, result)
		return result
	}

	return result
}

// CommandNotify implements the SSNTP client CommandNotify callback for SsntpTestClient
func (client *SsntpTestClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	payload := frame.Payload

	var result Result

	if frame.Trace != nil {
		frame.SetEndStamp()
		client.tracesLock.Lock()
		client.traces = append(client.traces, frame)
		client.tracesLock.Unlock()
	}

	switch command {
	/* FIXME: implement
	case ssntp.CONNECT:
	case ssntp.STATS:
	case ssntp.EVACUATE:
	case ssntp.AssignPublicIP:
	case ssntp.ReleasePublicIP:
	case ssntp.CONFIGURE:
	*/
	case ssntp.START:
		result = client.handleStart(payload)

	case ssntp.STOP:
		result = client.handleStop(payload)

	case ssntp.RESTART:
		result = client.handleRestart(payload)

	case ssntp.DELETE:
		result = client.handleDelete(payload)

	case ssntp.AttachVolume:
		result = client.handleAttachVolume(payload)

	case ssntp.DetachVolume:
		result = client.handleDetachVolume(payload)

	default:
		fmt.Fprintf(os.Stderr, "client %s unhandled command %s\n", client.Role.String(), command.String())
	}

	go client.SendResultAndDelCmdChan(command, result)
}

// EventNotify is an SSNTP callback stub for SsntpTestClient
func (client *SsntpTestClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	var result Result

	switch event {
	case ssntp.NodeConnected:
		//handled by ConnectNotify()
		return
	case ssntp.NodeDisconnected:
		//handled by DisconnectNotify()
		return
	case ssntp.TenantAdded:
		var tenantAddedEvent payloads.EventTenantAdded

		err := yaml.Unmarshal(frame.Payload, &tenantAddedEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.TenantRemoved:
		var tenantRemovedEvent payloads.EventTenantRemoved

		err := yaml.Unmarshal(frame.Payload, &tenantRemovedEvent)
		if err != nil {
			result.Err = err
		}
	default:
		fmt.Fprintf(os.Stderr, "client %s unhandled event: %s\n", client.Role.String(), event.String())
	}

	go client.SendResultAndDelEventChan(event, result)
}

// ErrorNotify is an SSNTP callback stub for SsntpTestClient
func (client *SsntpTestClient) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
}

// SendStatsCmd pushes an ssntp.STATS command frame from the SsntpTestClient
func (client *SsntpTestClient) SendStatsCmd() {
	var result Result

	client.instancesLock.Lock()
	payload := StatsPayload(client.UUID, client.Name, client.instances, nil)
	client.instancesLock.Unlock()

	y, err := yaml.Marshal(payload)
	if err != nil {
		result.Err = err
	} else {
		_, err = client.Ssntp.SendCommand(ssntp.STATS, y)
		if err != nil {
			result.Err = err
		}
	}

	go client.SendResultAndDelCmdChan(ssntp.STATS, result)
}

// SendStatus pushes an ssntp status frame from the SsntpTestClient with
// the indicated total and available memory statistics
func (client *SsntpTestClient) SendStatus(memTotal int, memAvail int) {
	var result Result

	payload := ReadyPayload(client.UUID, memTotal, memAvail)

	y, err := yaml.Marshal(payload)
	if err != nil {
		result.Err = err
	} else {
		_, err = client.Ssntp.SendStatus(ssntp.READY, y)
		if err != nil {
			result.Err = err
		}
	}

	go client.SendResultAndDelCmdChan(ssntp.STATS, result)
}

// SendTrace allows an SsntpTestClient to push an ssntp.TraceReport event frame
func (client *SsntpTestClient) SendTrace() {
	var result Result
	var s payloads.Trace

	client.tracesLock.Lock()
	defer client.tracesLock.Unlock()

	for _, f := range client.traces {
		t, err := f.DumpTrace()
		if err != nil {
			continue
		}

		s.Frames = append(s.Frames, *t)
	}

	y, err := yaml.Marshal(&s)
	if err != nil {
		result.Err = err
	} else {
		client.traces = nil

		_, err = client.Ssntp.SendEvent(ssntp.TraceReport, y)
		if err != nil {
			result.Err = err
		}
	}

	go client.SendResultAndDelEventChan(ssntp.TraceReport, result)
}

// SendDeleteEvent allows an SsntpTestClient to push an ssntp.InstanceDeleted event frame
func (client *SsntpTestClient) SendDeleteEvent(uuid string) {
	var result Result

	evt := payloads.InstanceDeletedEvent{
		InstanceUUID: uuid,
	}

	event := payloads.EventInstanceDeleted{
		InstanceDeleted: evt,
	}

	y, err := yaml.Marshal(event)
	if err != nil {
		result.Err = err
	} else {
		_, err = client.Ssntp.SendEvent(ssntp.InstanceDeleted, y)
		if err != nil {
			result.Err = err
		}
	}

	go client.SendResultAndDelEventChan(ssntp.InstanceDeleted, result)
}

// SendTenantAddedEvent allows an SsntpTestClient to push an ssntp.TenantAdded event frame
func (client *SsntpTestClient) SendTenantAddedEvent() {
	var result Result

	_, err := client.Ssntp.SendEvent(ssntp.TenantAdded, []byte(TenantAddedYaml))
	if err != nil {
		result.Err = err
	}

	go client.SendResultAndDelEventChan(ssntp.TenantAdded, result)
}

// SendTenantRemovedEvent allows an SsntpTestClient to push an ssntp.TenantRemoved event frame
func (client *SsntpTestClient) SendTenantRemovedEvent() {
	var result Result

	_, err := client.Ssntp.SendEvent(ssntp.TenantRemoved, []byte(TenantRemovedYaml))
	if err != nil {
		result.Err = err
	}

	go client.SendResultAndDelEventChan(ssntp.TenantRemoved, result)
}

// SendPublicIPAssignedEvent allows an SsntpTestClient to push an ssntp.PublicIPAssigned event frame
func (client *SsntpTestClient) SendPublicIPAssignedEvent() {
	var result Result

	_, err := client.Ssntp.SendEvent(ssntp.PublicIPAssigned, []byte(AssignedIPYaml))
	if err != nil {
		result.Err = err
	}

	go client.SendResultAndDelEventChan(ssntp.PublicIPAssigned, result)
}

// SendPublicIPUnassignedEvent allows an SsntpTestClient to push an ssntp.PublicIPUnassigned event frame
func (client *SsntpTestClient) SendPublicIPUnassignedEvent() {
	var result Result

	_, err := client.Ssntp.SendEvent(ssntp.PublicIPUnassigned, []byte(UnassignedIPYaml))
	if err != nil {
		result.Err = err
	}

	go client.SendResultAndDelEventChan(ssntp.PublicIPUnassigned, result)
}

// SendConcentratorAddedEvent allows an SsntpTestClient to push an ssntp.ConcentratorInstanceAdded event frame
func (client *SsntpTestClient) SendConcentratorAddedEvent(instanceUUID string, tenantUUID string, ip string, vnicMAC string) {
	var result Result

	evt := payloads.ConcentratorInstanceAddedEvent{
		InstanceUUID:    instanceUUID,
		TenantUUID:      tenantUUID,
		ConcentratorIP:  ip,
		ConcentratorMAC: vnicMAC,
	}
	result.InstanceUUID = instanceUUID

	event := payloads.EventConcentratorInstanceAdded{
		CNCIAdded: evt,
	}

	y, err := yaml.Marshal(event)
	if err != nil {
		result.Err = err
	} else {
		_, err = client.Ssntp.SendEvent(ssntp.ConcentratorInstanceAdded, y)
		if err != nil {
			result.Err = err
		}
	}

	go client.SendResultAndDelEventChan(ssntp.ConcentratorInstanceAdded, result)
}

func (client *SsntpTestClient) sendStartFailure(instanceUUID string, reason payloads.StartFailureReason) {
	e := payloads.ErrorStartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.StartFailure, y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (client *SsntpTestClient) sendStopFailure(instanceUUID string, reason payloads.StopFailureReason) {
	e := payloads.ErrorStopFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.StopFailure, y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (client *SsntpTestClient) sendRestartFailure(instanceUUID string, reason payloads.RestartFailureReason) {
	e := payloads.ErrorRestartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.RestartFailure, y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (client *SsntpTestClient) sendDeleteFailure(instanceUUID string, reason payloads.DeleteFailureReason) {
	e := payloads.ErrorDeleteFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.DeleteFailure, y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (client *SsntpTestClient) sendAttachVolumeFailure(instanceUUID string, volumeUUID string, reason payloads.AttachVolumeFailureReason) {
	e := payloads.ErrorAttachVolumeFailure{
		InstanceUUID: instanceUUID,
		VolumeUUID:   volumeUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.AttachVolumeFailure, y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (client *SsntpTestClient) sendDetachVolumeFailure(instanceUUID string, volumeUUID string, reason payloads.DetachVolumeFailureReason) {
	e := payloads.ErrorDetachVolumeFailure{
		InstanceUUID: instanceUUID,
		VolumeUUID:   volumeUUID,
		Reason:       reason,
	}

	y, err := yaml.Marshal(e)
	if err != nil {
		return
	}

	_, err = client.Ssntp.SendError(ssntp.DetachVolumeFailure, y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
