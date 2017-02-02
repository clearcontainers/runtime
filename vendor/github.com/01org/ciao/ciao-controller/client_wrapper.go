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
	"fmt"
	"sync"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/ssntp"
)

type ssntpClientWrapper struct {
	ctl        *controller
	name       string
	realClient controllerClient

	CmdChans       map[ssntp.Command]chan struct{}
	CmdChansLock   sync.Mutex
	EventChans     map[ssntp.Event]chan struct{}
	EventChansLock sync.Mutex
	ErrorChans     map[ssntp.Error]chan struct{}
	ErrorChansLock sync.Mutex
}

func (client *ssntpClientWrapper) ConnectNotify() {
	client.realClient.ConnectNotify()
}

func (client *ssntpClientWrapper) DisconnectNotify() {
	client.realClient.DisconnectNotify()
}

func (client *ssntpClientWrapper) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
	client.realClient.StatusNotify(status, frame)
}

func (client *ssntpClientWrapper) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	client.realClient.CommandNotify(command, frame)
	client.sendAndDelCmdChan(command)
}

func (client *ssntpClientWrapper) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	client.realClient.EventNotify(event, frame)
	client.sendAndDelEventChan(event)
}

func (client *ssntpClientWrapper) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	client.realClient.ErrorNotify(err, frame)
	client.sendAndDelErrorChan(err)
}

func newWrappedSSNTPClient(ctl *controller, config *ssntp.Config) (*ssntpClientWrapper, error) {
	realClient := &ssntpClient{name: "ciao Controller", ctl: ctl}
	client := &ssntpClientWrapper{name: "ciao Controller", realClient: realClient}
	client.openClientChans()

	ssntp := client.realClient.ssntpClient()
	err := ssntp.Dial(config, client)
	return client, err
}

func (client *ssntpClientWrapper) StartTracedWorkload(config string, startTime time.Time, label string) error {
	return client.realClient.StartTracedWorkload(config, startTime, label)
}

func (client *ssntpClientWrapper) StartWorkload(config string) error {
	return client.realClient.StartWorkload(config)
}

func (client *ssntpClientWrapper) DeleteInstance(instanceID string, nodeID string) error {
	return client.realClient.DeleteInstance(instanceID, nodeID)
}

func (client *ssntpClientWrapper) StopInstance(instanceID string, nodeID string) error {
	return client.realClient.StopInstance(instanceID, nodeID)
}

func (client *ssntpClientWrapper) RestartInstance(instanceID string, nodeID string) error {
	return client.realClient.RestartInstance(instanceID, nodeID)
}

func (client *ssntpClientWrapper) EvacuateNode(nodeID string) error {
	return client.realClient.EvacuateNode(nodeID)
}

func (client *ssntpClientWrapper) mapExternalIP(t types.Tenant, m types.MappedIP) error {
	return client.realClient.mapExternalIP(t, m)
}

func (client *ssntpClientWrapper) unMapExternalIP(t types.Tenant, m types.MappedIP) error {
	return client.realClient.unMapExternalIP(t, m)
}

func (client *ssntpClientWrapper) attachVolume(volID string, instanceID string, nodeID string) error {
	return client.realClient.attachVolume(volID, instanceID, nodeID)
}

func (client *ssntpClientWrapper) detachVolume(volID string, instanceID string, nodeID string) error {
	return client.realClient.detachVolume(volID, instanceID, nodeID)
}

func (client *ssntpClientWrapper) ssntpClient() *ssntp.Client {
	return client.realClient.ssntpClient()
}

func (client *ssntpClientWrapper) Disconnect() {
	client.realClient.Disconnect()
	client.closeClientChans()
}

func (client *ssntpClientWrapper) openClientChans() {
	client.CmdChansLock.Lock()
	client.CmdChans = make(map[ssntp.Command]chan struct{})
	client.CmdChansLock.Unlock()

	client.EventChansLock.Lock()
	client.EventChans = make(map[ssntp.Event]chan struct{})
	client.EventChansLock.Unlock()

	client.ErrorChansLock.Lock()
	client.ErrorChans = make(map[ssntp.Error]chan struct{})
	client.ErrorChansLock.Unlock()
}

func (client *ssntpClientWrapper) closeClientChans() {
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
}

// addCmdChan monitors for a ssntp.Command to be received.
func (client *ssntpClientWrapper) addCmdChan(cmd ssntp.Command) chan struct{} {
	c := make(chan struct{})

	client.CmdChansLock.Lock()
	client.CmdChans[cmd] = c
	client.CmdChansLock.Unlock()

	return c
}

// getCmdChan waits for a response on a supplied channel for the desired
// ssntp.Command.
func (client *ssntpClientWrapper) getCmdChan(c chan struct{}, cmd ssntp.Command) error {
	select {
	case <-c:
		return nil
	case <-time.After(25 * time.Second):
		err := fmt.Errorf("Timeout waiting for client %s command", cmd)
		return err
	}
}

func (client *ssntpClientWrapper) sendAndDelCmdChan(cmd ssntp.Command) {
	client.CmdChansLock.Lock()
	c, ok := client.CmdChans[cmd]
	if ok {
		delete(client.CmdChans, cmd)
		client.CmdChansLock.Unlock()
		c <- struct{}{}
		close(c)
		return
	}
	client.CmdChansLock.Unlock()
}

// addErrorChan monitors for a ssntp.Error to be received.
func (client *ssntpClientWrapper) addErrorChan(cmd ssntp.Error) chan struct{} {
	c := make(chan struct{})

	client.ErrorChansLock.Lock()
	client.ErrorChans[cmd] = c
	client.ErrorChansLock.Unlock()

	return c
}

// getErrorChan waits for a response on a supplied channel for the desired
// ssntp.Error.
func (client *ssntpClientWrapper) getErrorChan(c chan struct{}, cmd ssntp.Error) error {
	select {
	case <-c:
		return nil
	case <-time.After(25 * time.Second):
		err := fmt.Errorf("Timeout waiting for client %s error", cmd)
		return err
	}
}

func (client *ssntpClientWrapper) sendAndDelErrorChan(cmd ssntp.Error) {
	client.ErrorChansLock.Lock()
	c, ok := client.ErrorChans[cmd]
	if ok {
		delete(client.ErrorChans, cmd)
		client.ErrorChansLock.Unlock()
		c <- struct{}{}
		close(c)
		return
	}
	client.ErrorChansLock.Unlock()
}

// addEventChan monitors for a ssntp.Event to be received.
func (client *ssntpClientWrapper) addEventChan(cmd ssntp.Event) chan struct{} {
	c := make(chan struct{})

	client.EventChansLock.Lock()
	client.EventChans[cmd] = c
	client.EventChansLock.Unlock()

	return c
}

// getEventChan waits for a response on a supplied channel for the desired
// ssntp.Event.
func (client *ssntpClientWrapper) getEventChan(c chan struct{}, cmd ssntp.Event) error {
	select {
	case <-c:
		return nil
	case <-time.After(25 * time.Second):
		err := fmt.Errorf("Timeout waiting for client %s event", cmd)
		return err
	}
}

func (client *ssntpClientWrapper) sendAndDelEventChan(cmd ssntp.Event) {
	client.EventChansLock.Lock()
	c, ok := client.EventChans[cmd]
	if ok {
		delete(client.EventChans, cmd)
		client.EventChansLock.Unlock()
		c <- struct{}{}
		close(c)
		return
	}
	client.EventChansLock.Unlock()
}
