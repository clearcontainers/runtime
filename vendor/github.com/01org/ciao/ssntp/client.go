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

package ssntp

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ssntp/uuid"
	"gopkg.in/yaml.v2"
)

// ClientNotifier is the SSNTP client notification interface.
// Any SSNTP client must implement this interface.
// IMPORTANT: All ClientNotifier implementations must be thread
// safe, i.e. they will very likely be called by multiple go routines.
type ClientNotifier interface {
	// ConnectNotify notifies of a successful connection to an SSNTP server.
	// This notification is mostly useful for clients to know when they're
	// being re-connected to the SSNTP server.
	ConnectNotify()

	// DisconnectNotify notifies of a SSNTP server disconnection.
	// SSNTP Client implementations are not supposed to explicitly
	// reconnect, the SSNTP protocol will handle the reconnection.
	DisconnectNotify()

	// StatusNotify notifies of a pending status frame from the SSNTP server.
	StatusNotify(status Status, frame *Frame)

	// CommandNotify notifies of a pending command frame from the SSNTP server.
	CommandNotify(command Command, frame *Frame)

	// EventNotify notifies of a pending event frame from the SSNTP server.
	EventNotify(event Event, frame *Frame)

	// ErrorNotify notifies of a pending error frame from the SSNTP server.
	// Error frames are always related to the last sent frame.
	ErrorNotify(error Error, frame *Frame)
}

// Client is the SSNTP client structure.
// This is an SSNTP client handle to connect to and
// disconnect from an SSNTP server, and send SSNTP
// frames to it.
// It is an entirely opaque structure, only accessible through
// its public methods.
type Client struct {
	uuid      uuid.UUID
	lUUID     lockedUUID
	uris      []string
	role      Role
	tls       *tls.Config
	ntf       ClientNotifier
	transport string
	port      uint32
	session   *session
	status    connectionStatus
	closed    chan struct{}

	frameWg              sync.WaitGroup
	frameRoutinesChannel chan struct{}

	log Logger

	trace *TraceConfig

	configuration clusterConfiguration
}

func (client *Client) processSSNTPFrame(frame *Frame) {
	defer client.frameWg.Done()

	switch (Type)(frame.Type) {
	case COMMAND:
		if (Command)(frame.Operand) == CONFIGURE {
			client.configuration.setConfiguration(frame.Payload)
		}
		client.ntf.CommandNotify((Command)(frame.Operand), frame)
	case STATUS:
		client.ntf.StatusNotify((Status)(frame.Operand), frame)
	case EVENT:
		client.ntf.EventNotify((Event)(frame.Operand), frame)
	case ERROR:
		client.ntf.ErrorNotify((Error)(frame.Operand), frame)
	default:
		client.SendError(InvalidFrameType, nil)
	}
}

func (client *Client) handleSSNTPServer() {
	defer client.Close()

	for {
		client.ntf.ConnectNotify()

		for {
			client.log.Infof("Waiting for next frame\n")

			var frame Frame
			err := client.session.Read(&frame)
			if err != nil {
				client.status.Lock()
				if client.status.status == ssntpClosed {
					client.status.Unlock()
					return
				}
				client.status.Unlock()

				client.log.Errorf("Read error: %s\n", err)
				client.ntf.DisconnectNotify()
				break
			}

			client.status.Lock()
			if client.status.status == ssntpClosed {
				client.status.Unlock()
				return
			}
			//insure new frame doesn't race with client.Close()
			client.frameWg.Add(1)
			client.status.Unlock()

			go client.processSSNTPFrame(&frame)
		}

		err := client.attemptDial()
		if err != nil {
			client.log.Errorf("%s", err)
			return
		}
	}
}

func (client *Client) sendConnect() (bool, error) {
	var connected ConnectedFrame
	client.log.Infof("Sending CONNECT\n")

	connect := client.session.connectFrame()
	_, err := client.session.Write(connect)
	if err != nil {
		return true, err
	}

	client.log.Infof("Waiting for CONNECTED\n")
	err = client.session.Read(&connected)
	if err != nil {
		return true, err
	}

	client.log.Infof("Received CONNECTED frame:\n%s\n", connected)

	switch connected.Type {
	case STATUS:
		if connected.Operand != (uint8)(CONNECTED) {
			return true, fmt.Errorf("SSNTP Client: Invalid Connected frame")
		}
	case ERROR:
		if connected.Operand != (uint8)(ConnectionFailure) {
			return false, fmt.Errorf("SSNTP Client: Connection failure")
		}

		return true, fmt.Errorf("SSNTP Client: Connection error %s", (Error)(connected.Operand))

	default:
		return true, fmt.Errorf("SSNTP Client: Unknown frame type %d", connected.Type)
	}

	client.session.setDest(connected.Source[:16])

	oidFound, err := verifyRole(client.session.conn, connected.Role)
	if oidFound == false {
		client.log.Errorf("%s\n", err)
		client.SendError(ConnectionFailure, nil)
		return false, fmt.Errorf("SSNTP Client: Connection failure")
	}

	client.status.Lock()
	client.status.status = ssntpConnected
	client.status.Unlock()

	client.configuration.setConfiguration(connected.Payload)

	client.log.Infof("Done with connection\n")

	return true, nil
}

func (client *Client) attemptDial() error {
	delays := []int64{5, 10, 20, 40}

	if len(client.uris) == 0 {
		return fmt.Errorf("No servers to connect to")
	}

	client.status.Lock()
	client.closed = make(chan struct{})
	client.status.Unlock()

	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	for {
	URILoop:
		for d := 0; ; d++ {
			for _, uri := range client.uris {
				client.log.Infof("%s connecting to %s\n", client.uuid, uri)
				conn, err := tls.Dial(client.transport, uri, client.tls)

				client.status.Lock()
				if client.status.status == ssntpClosed {
					client.status.Unlock()
					return fmt.Errorf("Connection closed")
				}
				client.status.Unlock()

				if err == nil {
					client.log.Infof("Connected\n")
					session := newSession(&client.uuid, client.role, 0, conn)
					client.session = session

					break URILoop
				}

				client.log.Errorf("Could not connect to %s (%s)\n", uri, err)
			}

			delay := r.Int63n(delays[d%len(delays)])
			delay++ // Avoid waiting for 0 seconds
			client.log.Errorf("All server URIs failed - retrying in %d seconds\n", delay)

			// Wait for delay before reconnecting or return if the client is closed
			select {
			case <-client.closed:
				return fmt.Errorf("Connection closed")
			case <-time.After(time.Duration(delay) * time.Second):
				break
			}

			continue
		}

		if client.session == nil {
			continue
		}

		reconnect, err := client.sendConnect()
		if err != nil {
			// Dialed but could not connect, try again
			client.log.Errorf("%s", err)
			client.Close()
			if reconnect == true {
				continue
			} else {
				client.ntf.DisconnectNotify()
				return err
			}
		}

		// Dialed and connected, we can proceed
		break
	}

	return nil
}

// Dial attempts to connect to a SSNTP server, as specified by the config argument.
// Dial will try and retry to connect to this server and will wait for it to show
// up if it's temporarily unavailable. A client can be closed while it's still
// trying to connect to the SSNTP server, so that one can properly kill a client if
// e.g. no server will ever come alive.
// Once connected a separate routine will listen for server commands, statuses or
// errors and report them back through the SSNTP client notifier interface.
func (client *Client) Dial(config *Config, ntf ClientNotifier) error {
	if config == nil {
		return fmt.Errorf("SSNTP config missing")
	}

	client.status.Lock()

	if client.status.status == ssntpConnected || client.status.status == ssntpConnecting {
		client.status.Unlock()
		err := fmt.Errorf("Client already connected")
		config.pushToSyncChannel(err)
		return err
	}

	if client.status.status == ssntpClosed {
		client.status.Unlock()
		err := fmt.Errorf("Client already closed")
		config.pushToSyncChannel(err)
		return err
	}

	client.status.status = ssntpConnecting

	client.status.Unlock()

	client.log = config.log()
	config.setCerts()
	role, err := config.role()
	if err != nil {
		client.log.Errorf("%s", err)
		config.pushToSyncChannel(err)
		return err
	}
	client.role = role
	client.lUUID, client.uuid = config.configUUID(client.role)
	client.port = config.port()
	client.transport = config.transport()
	client.uris = config.ConfigURIs(client.uris, client.port)

	client.trace = config.Trace
	client.ntf = ntf
	client.tls = prepareTLSConfig(config, false)

	err = client.attemptDial()
	if err != nil {
		client.log.Errorf("%s", err)
		config.pushToSyncChannel(err)
		return err
	}

	go client.handleSSNTPServer()
	config.pushToSyncChannel(nil)

	return nil
}

// Close terminates the client connection.
func (client *Client) Close() {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return
	}

	if client.session != nil {
		client.session.conn.Close()
	}
	client.status.status = ssntpClosed
	if client.closed != nil {
		close(client.closed)
	}
	client.status.Unlock()

	client.frameRoutinesChannel = make(chan struct{})
	go func(client *Client) {
		client.frameWg.Wait()
		close(client.frameRoutinesChannel)
	}(client)

	select {
	case <-client.frameRoutinesChannel:
		break
	case <-time.After(2 * time.Second):
		break
	}

	freeUUID(client.lUUID)
}

func (client *Client) sendCommand(cmd Command, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("sendCommand: Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.commandFrame(cmd, payload, trace)

	return session.Write(frame)
}

func (client *Client) sendStatus(status Status, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("sendStatus: Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.statusFrame(status, payload, trace)

	return session.Write(frame)
}

func (client *Client) sendEvent(event Event, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("sendEvent: Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.eventFrame(event, payload, trace)

	return session.Write(frame)
}

func (client *Client) sendError(error Error, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("sendError: Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.errorFrame(error, payload, trace)

	return session.Write(frame)
}

// SendCommand sends a specific command and its payload to the SSNTP server.
func (client *Client) SendCommand(cmd Command, payload []byte) (int, error) {
	return client.sendCommand(cmd, payload, client.trace)
}

// SendStatus sends a specific status and its payload to the SSNTP server.
func (client *Client) SendStatus(status Status, payload []byte) (int, error) {
	return client.sendStatus(status, payload, client.trace)
}

// SendEvent sends a specific status and its payload to the SSNTP server.
func (client *Client) SendEvent(event Event, payload []byte) (int, error) {
	return client.sendEvent(event, payload, client.trace)
}

// SendError sends an error back to the SSNTP server.
// This is just for notification purposes, to let e.g. the server know that
// it sent an unexpected frame.
func (client *Client) SendError(error Error, payload []byte) (int, error) {
	return client.sendError(error, payload, client.trace)
}

// SendTracedCommand sends a specific command and its payload to the SSNTP server.
// The SSNTP command frame will be traced according to the trace argument.
func (client *Client) SendTracedCommand(cmd Command, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendCommand(cmd, payload, trace)
}

// SendTracedStatus sends a specific status and its payload to the SSNTP server.
// The SSNTP status frame will be traced according to the trace argument.
func (client *Client) SendTracedStatus(status Status, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendStatus(status, payload, trace)
}

// SendTracedEvent sends a specific status and its payload to the SSNTP server.
// The SSNTP event frame will be traced according to the trace argument.
func (client *Client) SendTracedEvent(event Event, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendEvent(event, payload, trace)
}

// SendTracedError sends an error back to the SSNTP server.
// This is just for notification purposes, to let e.g. the server know that
// it sent an unexpected frame.
// The SSNTP error frame will be traced according to the trace argument.
func (client *Client) SendTracedError(error Error, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendError(error, payload, trace)
}

// Role exports the SSNTP client role.
func (client *Client) Role() Role {
	return client.role
}

// UUID exports the SSNTP client Universally Unique ID.
func (client *Client) UUID() string {
	return client.uuid.String()
}

// ClusterConfiguration returns the latest cluster configuration
// payload a client received. Clients should use that payload to
// configure themselves based on the information provided to them
// by the Scheduler or the Controller.
// Cluster configuration payloads can come from either a CONNECTED
// status frame or a CONFIGURE command one.
func (client *Client) ClusterConfiguration() (payloads.Configure, error) {
	var conf payloads.Configure

	client.configuration.RLock()
	defer client.configuration.RUnlock()

	if client.configuration.configuration == nil {
		return conf, fmt.Errorf("No client configuration available")
	}

	err := yaml.Unmarshal(client.configuration.configuration, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}
