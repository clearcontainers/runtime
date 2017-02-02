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
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
)

type cmdWrapper struct {
	instance string
	cmd      interface{}
}
type statusCmd struct{}

// serverConn is an abstract interface representing a connection to
// a server.  It contains methods to connect to the server and to
// send information to the server, such as commands or events.
type serverConn interface {
	SendError(error ssntp.Error, payload []byte) (int, error)
	SendEvent(event ssntp.Event, payload []byte) (int, error)
	Dial(config *ssntp.Config, ntf ssntp.ClientNotifier) error
	SendStatus(status ssntp.Status, payload []byte) (int, error)
	SendCommand(cmd ssntp.Command, payload []byte) (int, error)
	Role() ssntp.Role
	UUID() string
	Close()
	isConnected() bool
	setStatus(status bool)
	ClusterConfiguration() (payloads.Configure, error)
}

// ssntpConn is a concrete implementation of serverConn.  It represents
// a connection to an SSNTP server, i.e., the scheduler.
type ssntpConn struct {
	sync.RWMutex
	ssntp.Client
	connected bool
}

func (s *ssntpConn) isConnected() bool {
	s.RLock()
	defer s.RUnlock()
	return s.connected
}

func (s *ssntpConn) setStatus(status bool) {
	s.Lock()
	s.connected = status
	s.Unlock()
}

// agentClient is a structure that serves two purposes.  It holds contains
// a serverConn object and so can be used to send commands to an SSNTP
// server.  It also implements the ssntp.ClientNotifier interface and so
// can be passed to serverConn.Dial.
type agentClient struct {
	conn  serverConn
	cmdCh chan *cmdWrapper
}

func (client *agentClient) DisconnectNotify() {
	client.conn.setStatus(false)
	glog.Warning("disconnected")
}

func (client *agentClient) ConnectNotify() {
	client.conn.setStatus(true)
	client.cmdCh <- &cmdWrapper{"", &statusCmd{}}
	glog.Info("connected")
}

func (client *agentClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
	glog.Infof("STATUS %s", status)
}

func (client *agentClient) CommandNotify(cmd ssntp.Command, frame *ssntp.Frame) {
	payload := frame.Payload

	switch cmd {
	case ssntp.START:
		start, cn, md := splitYaml(payload)
		cfg, payloadErr := parseStartPayload(start)
		if payloadErr != nil {
			startError := &startError{
				payloadErr.err,
				payloads.StartFailureReason(payloadErr.code),
			}
			startError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %v", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{cfg.Instance, &insStartCmd{cn, md, frame, cfg, time.Now()}}
	case ssntp.RESTART:
		instance, payloadErr := parseRestartPayload(payload)
		if payloadErr != nil {
			restartError := &restartError{
				payloadErr.err,
				payloads.RestartFailureReason(payloadErr.code),
			}
			restartError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %v", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insRestartCmd{}}
	case ssntp.STOP:
		instance, payloadErr := parseStopPayload(payload)
		if payloadErr != nil {
			stopError := &stopError{
				payloadErr.err,
				payloads.StopFailureReason(payloadErr.code),
			}
			stopError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insStopCmd{}}
	case ssntp.DELETE:
		instance, payloadErr := parseDeletePayload(payload)
		if payloadErr != nil {
			deleteError := &deleteError{
				payloadErr.err,
				payloads.DeleteFailureReason(payloadErr.code),
			}
			deleteError.send(client.conn, "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insDeleteCmd{}}
	case ssntp.AttachVolume:
		instance, volume, payloadErr := parseAttachVolumePayload(payload)
		if payloadErr != nil {
			attachVolumeError := &attachVolumeError{
				payloadErr.err,
				payloads.AttachVolumeFailureReason(payloadErr.code),
			}
			attachVolumeError.send(client.conn, "", "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insAttachVolumeCmd{volume}}
	case ssntp.DetachVolume:
		instance, volume, payloadErr := parseDetachVolumePayload(payload)
		if payloadErr != nil {
			detachVolumeError := &detachVolumeError{
				payloadErr.err,
				payloads.DetachVolumeFailureReason(payloadErr.code),
			}
			detachVolumeError.send(client.conn, "", "")
			glog.Errorf("Unable to parse YAML: %s", payloadErr.err)
			return
		}
		client.cmdCh <- &cmdWrapper{instance, &insDetachVolumeCmd{volume}}
	}
}

func (client *agentClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	glog.Infof("EVENT %s", event)
}

func (client *agentClient) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	glog.Infof("ERROR %d", err)
}
