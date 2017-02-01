// // Copyright (c) 2016 Intel Corporation
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

// SsntpTestController is global state for the testutil SSNTP controller
type SsntpTestController struct {
	Ssntp          ssntp.Client
	Name           string
	UUID           string
	CmdChans       map[ssntp.Command]chan Result
	CmdChansLock   *sync.Mutex
	EventChans     map[ssntp.Event]chan Result
	EventChansLock *sync.Mutex
	ErrorChans     map[ssntp.Error]chan Result
	ErrorChansLock *sync.Mutex
}

// Shutdown shuts down the testutil.SsntpTestClient and cleans up state
func (ctl *SsntpTestController) Shutdown() {
	closeControllerChans(ctl)
	ctl.Ssntp.Close()
}

// NewSsntpTestControllerConnection creates an SsntpTestController and dials the server.
// Calling with a unique name parameter string for inclusion in the
// SsntpTestClient.Name field aides in debugging.  The uuid string
// parameter allows tests to specify a known uuid for simpler tests.
func NewSsntpTestControllerConnection(name string, uuid string) (*SsntpTestController, error) {
	if uuid == "" {
		return nil, errors.New("no uuid specified")
	}

	var role ssntp.Role = ssntp.Controller
	ctl := &SsntpTestController{
		Name: "Test " + role.String() + " " + name,
		UUID: uuid,
	}

	ctl.CmdChansLock = &sync.Mutex{}
	ctl.EventChansLock = &sync.Mutex{}
	ctl.ErrorChansLock = &sync.Mutex{}
	openControllerChans(ctl)

	config := &ssntp.Config{
		URI:    "",
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(ssntp.Controller),
		Log:    ssntp.Log,
		UUID:   ctl.UUID,
	}

	if err := ctl.Ssntp.Dial(config, ctl); err != nil {
		return nil, err
	}
	return ctl, nil
}

// AddCmdChan adds an ssntp.Command to the SsntpTestController command channel
func (ctl *SsntpTestController) AddCmdChan(cmd ssntp.Command) chan Result {
	c := make(chan Result)

	ctl.CmdChansLock.Lock()
	ctl.CmdChans[cmd] = c
	ctl.CmdChansLock.Unlock()

	return c
}

// GetCmdChanResult gets a Result from the SsntpTestController command channel
func (ctl *SsntpTestController) GetCmdChanResult(c chan Result, cmd ssntp.Command) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Controller error sending %s command: %s", cmd, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for controller %s command result", cmd)
	}

	return result, err
}

// SendResultAndDelCmdChan deletes an ssntp.Command from the SsntpTestController command channel
func (ctl *SsntpTestController) SendResultAndDelCmdChan(cmd ssntp.Command, result Result) {
	ctl.CmdChansLock.Lock()
	c, ok := ctl.CmdChans[cmd]
	if ok {
		delete(ctl.CmdChans, cmd)
		ctl.CmdChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	ctl.CmdChansLock.Unlock()
}

// AddEventChan adds an ssntp.Event to the SsntpTestController event channel
func (ctl *SsntpTestController) AddEventChan(evt ssntp.Event) chan Result {
	c := make(chan Result)

	ctl.EventChansLock.Lock()
	ctl.EventChans[evt] = c
	ctl.EventChansLock.Unlock()

	return c
}

// GetEventChanResult gets a Result from the SsntpTestController event channel
func (ctl *SsntpTestController) GetEventChanResult(c chan Result, evt ssntp.Event) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Controller error sending %s event: %s", evt, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for controller %s event result", evt)
	}

	return result, err
}

// SendResultAndDelEventChan deletes an ssntpEvent from the SsntpTestController event channel
func (ctl *SsntpTestController) SendResultAndDelEventChan(evt ssntp.Event, result Result) {
	ctl.EventChansLock.Lock()
	c, ok := ctl.EventChans[evt]
	if ok {
		delete(ctl.EventChans, evt)
		ctl.EventChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	ctl.EventChansLock.Unlock()
}

// AddErrorChan adds an ssntp.Error to the SsntpTestController error channel
func (ctl *SsntpTestController) AddErrorChan(error ssntp.Error) chan Result {
	c := make(chan Result)

	ctl.ErrorChansLock.Lock()
	ctl.ErrorChans[error] = c
	ctl.ErrorChansLock.Unlock()

	return c
}

// GetErrorChanResult gets a Result from the SsntpTestController error channel
func (ctl *SsntpTestController) GetErrorChanResult(c chan Result, error ssntp.Error) (result Result, err error) {
	select {
	case result = <-c:
		if result.Err != nil {
			err = fmt.Errorf("Controller error sending %s error: %s", error, result.Err)
		}
	case <-time.After(25 * time.Second):
		err = fmt.Errorf("Timeout waiting for controller %s error result", error)
	}

	return result, err
}

// SendResultAndDelErrorChan deletes an ssntp.Error from the SsntpTestController error channel
func (ctl *SsntpTestController) SendResultAndDelErrorChan(error ssntp.Error, result Result) {
	ctl.ErrorChansLock.Lock()
	c, ok := ctl.ErrorChans[error]
	if ok {
		delete(ctl.ErrorChans, error)
		ctl.ErrorChansLock.Unlock()
		c <- result
		close(c)
		return
	}
	ctl.ErrorChansLock.Unlock()
}

func openControllerChans(ctl *SsntpTestController) {
	ctl.CmdChansLock.Lock()
	ctl.CmdChans = make(map[ssntp.Command]chan Result)
	ctl.CmdChansLock.Unlock()

	ctl.EventChansLock.Lock()
	ctl.EventChans = make(map[ssntp.Event]chan Result)
	ctl.EventChansLock.Unlock()

	ctl.ErrorChansLock.Lock()
	ctl.ErrorChans = make(map[ssntp.Error]chan Result)
	ctl.ErrorChansLock.Unlock()
}

func closeControllerChans(ctl *SsntpTestController) {
	ctl.CmdChansLock.Lock()
	for k := range ctl.CmdChans {
		close(ctl.CmdChans[k])
		delete(ctl.CmdChans, k)
	}
	ctl.CmdChansLock.Unlock()

	ctl.EventChansLock.Lock()
	for k := range ctl.EventChans {
		close(ctl.EventChans[k])
		delete(ctl.EventChans, k)
	}
	ctl.EventChansLock.Unlock()

	ctl.ErrorChansLock.Lock()
	for k := range ctl.ErrorChans {
		close(ctl.ErrorChans[k])
		delete(ctl.ErrorChans, k)
	}
	ctl.ErrorChansLock.Unlock()
}

// ConnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestController
func (ctl *SsntpTestController) ConnectNotify() {
	var result Result

	go ctl.SendResultAndDelEventChan(ssntp.NodeConnected, result)
}

// DisconnectNotify implements the SSNTP client DisconnectNotify callback for SsntpTestController
func (ctl *SsntpTestController) DisconnectNotify() {
	var result Result

	go ctl.SendResultAndDelEventChan(ssntp.NodeDisconnected, result)
}

// StatusNotify implements the SSNTP client StatusNotify callback for SsntpTestController
func (ctl *SsntpTestController) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

// CommandNotify implements the SSNTP client CommandNotify callback for SsntpTestController
func (ctl *SsntpTestController) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	var result Result

	//payload := frame.Payload

	switch command {
	/* FIXME: implement
	case ssntp.START:
	case ssntp.STOP:
	case ssntp.RESTART:
	case ssntp.DELETE:
	*/
	case ssntp.STATS:
		var stats payloads.Stat

		stats.Init()

		err := yaml.Unmarshal(frame.Payload, &stats)
		if err != nil {
			result.Err = err
		}

	default:
		fmt.Fprintf(os.Stderr, "controller unhandled command: %s\n", command.String())
	}

	go ctl.SendResultAndDelCmdChan(command, result)
}

// EventNotify implements the SSNTP client EventNotify callback for SsntpTestController
func (ctl *SsntpTestController) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	var result Result

	switch event {
	// case ssntp.TenantAdded: does not reach controller
	// case ssntp.TenantRemoved: does not reach controller
	case ssntp.NodeConnected:
		//handled by ConnectNotify()
		return
	case ssntp.NodeDisconnected:
		//handled by DisconnectNotify()
		return
	case ssntp.PublicIPAssigned:
		var publicIPAssignedEvent payloads.EventPublicIPAssigned

		err := yaml.Unmarshal(frame.Payload, &publicIPAssignedEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.InstanceDeleted:
		var deleteEvent payloads.EventInstanceDeleted

		err := yaml.Unmarshal(frame.Payload, &deleteEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.TraceReport:
		var traceEvent payloads.Trace

		err := yaml.Unmarshal(frame.Payload, &traceEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.ConcentratorInstanceAdded:
		var concentratorAddedEvent payloads.EventConcentratorInstanceAdded

		err := yaml.Unmarshal(frame.Payload, &concentratorAddedEvent)
		if err != nil {
			result.Err = err
		}
	default:
		fmt.Fprintf(os.Stderr, "controller unhandled event: %s\n", event.String())
	}

	go ctl.SendResultAndDelEventChan(event, result)
}

// ErrorNotify implements the SSNTP client ErrorNotify callback for SsntpTestController
func (ctl *SsntpTestController) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
	var result Result

	//payload := frame.Payload

	switch error {
	/* FIXME: implement
	case ssntp.InvalidFrameType:
	case ssntp.StartFailure:
	case ssntp.StopFailure:
	case ssntp.ConnectionFailure:
	case ssntp.RestartFailure:
	case ssntp.DeleteFailure:
	case ssntp.ConnectionAborted:
	case ssntp.InvalidConfiguration:
	*/
	default:
		fmt.Fprintf(os.Stderr, "controller unhandled error %s\n", error.String())
	}

	go ctl.SendResultAndDelErrorChan(error, result)
}
