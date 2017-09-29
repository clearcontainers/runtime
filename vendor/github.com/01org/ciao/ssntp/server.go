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
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ciao-project/ciao/configuration"
	"github.com/ciao-project/ciao/ssntp/uuid"
)

// ServerNotifier is the SSNTP server notification interface.
// Any SSNTP server must implement this interface.
type ServerNotifier interface {
	// ConnectNotify notifies of a new SSNTP client connection.
	ConnectNotify(uuid string, role Role)

	// DisconnectNotify notifies of a SSNTP client having
	// disconnected from us.
	DisconnectNotify(uuid string, role Role)

	// StatusNotify notifies of a pending status frame.
	// The frame comes from a SSNTP client identified by uuid.
	StatusNotify(uuid string, status Status, frame *Frame)

	// CommandNotify notifies of a pending command frame.
	// The frame comes from a SSNTP client identified by uuid.
	CommandNotify(uuid string, command Command, frame *Frame)

	// EventNotify notifies of a pending event frame.
	// The frame comes from a SSNTP client identified by uuid.
	EventNotify(uuid string, event Event, frame *Frame)

	// ErrorNotify notifies of a pending error frame.
	// The frame comes from a SSNTP client identified by uuid.
	ErrorNotify(uuid string, error Error, frame *Frame)
}

// Server is the SSNTP server structure.
// This is an SSNTP server handle to start listening and handling
// SSNTP client connections, and send SSNTP frames to them.
// It is an entirely opaque structure, only accessible through
// its public methods.
type Server struct {
	uuid          uuid.UUID
	lUUID         lockedUUID
	tls           *tls.Config
	ntf           ServerNotifier
	sessionMutex  sync.RWMutex
	sessions      map[string]*session
	listenerMutex sync.Mutex
	listener      net.Listener
	stopped       boolFlag
	stoppedChan   chan struct{}
	role          Role
	clientWg      sync.WaitGroup

	forwardRules frameForward

	log Logger

	trace *TraceConfig

	configuration clusterConfiguration
}

func sendConnectionFailure(conn net.Conn) *session {
	var session session
	encoder := gob.NewEncoder(conn)

	frame := session.errorFrame(ConnectionFailure, nil, nil)
	encoder.Encode(frame)

	return nil
}

func sendConnectionAborted(conn net.Conn) *session {
	var session session
	encoder := gob.NewEncoder(conn)

	frame := session.errorFrame(ConnectionAborted, nil, nil)
	encoder.Encode(frame)

	return nil
}

func handleClientConnect(server *Server, conn net.Conn) *session {
	var connect ConnectFrame

	decoder := gob.NewDecoder(conn)

	server.log.Infof("Waiting for CONNECT\n")
	setReadTimeout(conn)
	readErr := decoder.Decode(&connect)
	clearReadTimeout(conn)
	if readErr != nil {
		server.log.Errorf("Connect error: %s\n", readErr)
		return sendConnectionFailure(conn)
	}

	server.log.Infof("Received CONNECT frame:\n%s\n", connect)

	tlscon, ok := conn.(*tls.Conn)
	if ok {
		oidFound, err := verifyRole(tlscon, connect.Role)
		if oidFound == false {
			server.log.Errorf("%s\n", err)
			return sendConnectionAborted(conn)
		}
	}

	if connect.Type != COMMAND || connect.Operand != (uint8)(CONNECT) {
		server.log.Errorf("Invalid Connect frame")
		return sendConnectionFailure(conn)
	}

	session := newSession(&server.uuid, server.role, connect.Role, conn)
	session.setDest(connect.Source[:16])

	/* TODO Get the CONFIGURE payload from the config package */
	server.configuration.RLock()
	connected := session.connectedFrame(server.role, server.configuration.configuration)
	server.configuration.RUnlock()

	server.log.Infof("Sending CONNECTED\n")
	_, writeErr := session.Write(connected)
	if writeErr != nil {
		server.log.Errorf("Connected error: %s\n", writeErr)
		return sendConnectionFailure(conn)
	}

	return session
}

func handleSSNTPClient(server *Server, conn net.Conn) {
	defer server.clientWg.Done()
	defer conn.Close()

	server.log.Infof("New client connection\n")
	session := handleClientConnect(server, conn)
	if session == nil {
		return
	}

	uuidString := session.dest.String()
	server.addSession(session, uuidString)
	server.forwardRules.addForwardDestination(session)
	server.ntf.ConnectNotify(uuidString, session.destRole)

	for {
		var frame Frame
		err := session.Read(&frame)
		if err != nil {
			server.log.Infof("Client disconnection: %s %d\n", err)
			server.ntf.DisconnectNotify(uuidString, session.destRole)
			server.forwardRules.deleteForwardDestination(session)
			server.removeSession(uuidString)
			break
		}

		switch frame.Type {
		case COMMAND:
			if (Command)(frame.Operand) == CONFIGURE && session.destRole.IsController() {
				/* TODO Send the CONFIGURE payload to the config package */
				server.configuration.setConfiguration(frame.Payload)
			}
			server.forwardRules.forwardFrame(server, session, (Command)(frame.Operand), &frame)
			server.ntf.CommandNotify(uuidString, (Command)(frame.Operand), &frame)
		case STATUS:
			server.forwardRules.forwardFrame(server, session, (Status)(frame.Operand), &frame)
			server.ntf.StatusNotify(uuidString, (Status)(frame.Operand), &frame)
		case EVENT:
			server.forwardRules.forwardFrame(server, session, (Event)(frame.Operand), &frame)
			server.ntf.EventNotify(uuidString, (Event)(frame.Operand), &frame)
		case ERROR:
			server.forwardRules.forwardFrame(server, session, (Error)(frame.Operand), &frame)
			server.ntf.ErrorNotify(uuidString, (Error)(frame.Operand), &frame)
		default:
			server.SendError(uuidString, InvalidFrameType, nil)
		}
	}
}

/*
 * SSNTP Server methods
 */
func (server *Server) addSession(session *session, uuid string) {
	server.sessionMutex.Lock()
	server.sessions[uuid] = session
	server.sessionMutex.Unlock()
}

func (server *Server) removeSession(uuid string) {
	server.sessionMutex.Lock()
	delete(server.sessions, uuid)
	server.sessionMutex.Unlock()
}

func (server *Server) getSession(uuid string) *session {
	server.sessionMutex.RLock()
	session := server.sessions[uuid]
	server.sessionMutex.RUnlock()

	return session
}

// Serve starts an SSNTP server that will listen and serve SSNTP client
// connections. Notifiers will be called when new clients connect and
// disconnect. And also when statuses, payloads and errors are received.
func (server *Server) Serve(config *Config, ntf ServerNotifier) error {
	var uri string
	var serverPort uint32

	if config == nil {
		return fmt.Errorf("SSNTP config missing")
	}

	server.log = config.log()
	config.setCerts()
	role, err := config.role()
	if err != nil {
		server.log.Errorf("%s", err)
		config.pushToSyncChannel(err)
		return err
	}
	server.role = role

	server.lUUID, server.uuid = config.configUUID(server.role)
	serverPort = config.port()
	transport := config.transport()
	uri = config.URI

	payload, err := configuration.ExtractBlob(config.ConfigURI)

	if err != nil {
		server.log.Errorf("Error loading configuration data from %s: %s - You may have not installed your configuration file yet", config.ConfigURI, err)
	} else {
		server.configuration.setConfiguration(payload)
	}

	server.ntf = ntf
	server.sessions = make(map[string]*session)
	server.forwardRules.init(config.ForwardRules)
	server.tls = prepareTLSConfig(config, true)
	server.forwardRules.forwardRules = config.ForwardRules
	server.trace = config.Trace
	server.stoppedChan = make(chan struct{})

	service := fmt.Sprintf("%s:%d", uri, serverPort)
	listener, err := tls.Listen(transport, service, server.tls)
	if err != nil {
		server.log.Errorf("Failed to start listener (err=%s) on %s\n", err, service)
		config.pushToSyncChannel(err)
		return err
	}
	server.log.Infof("Listening on %s\n", service)

	server.listenerMutex.Lock()
	server.listener = listener
	server.listenerMutex.Unlock()
	defer listener.Close()

	config.pushToSyncChannel(nil)

	for {
		conn, err := listener.Accept()
		if err != nil {
			server.stopped.Lock()
			if server.stopped.flag == true {
				server.stopped.Unlock()
				break
			}
			server.stopped.Unlock()
			continue
		}

		server.clientWg.Add(1)
		go handleSSNTPClient(server, conn)
	}

	if server.stoppedChan != nil {
		close(server.stoppedChan)
	}

	return nil
}

// ServeThreadSync is a helper that start Serve() in a
// dedicated go routine and returns synchronously, i.e.
// when Serve() is ready to accept SSNTP clients or failed.
func (server *Server) ServeThreadSync(config *Config, ntf ServerNotifier) error {
	if config.SyncChannel == nil {
		config.SyncChannel = make(chan error)
	}

	go func() {
		server.Serve(config, ntf)
	}()

	select {
	case err := <-config.SyncChannel:
		return err
	case <-time.After(time.Second):
		return fmt.Errorf("Timeout receiving server notification")
	}
}

// Stop terminates the server listening operation
// and closes all client connections.
func (server *Server) Stop() {
	server.stopped.Lock()
	server.stopped.flag = true
	server.stopped.Unlock()

	server.listenerMutex.Lock()
	if server.listener != nil {
		server.listener.Close()
	}
	server.listenerMutex.Unlock()

	server.sessionMutex.RLock()
	for uuid, session := range server.sessions {
		server.log.Infof("Closing connection for %s\n", uuid)
		session.conn.Close()
	}
	server.sessionMutex.RUnlock()

	server.clientWg.Wait()

	select {
	case <-server.stoppedChan:
		break
	case <-time.After(2 * time.Second):
		server.log.Errorf("Timeout waiting for main server thread\n")
	}

	freeUUID(server.lUUID)
}

func (server *Server) sendCommand(uuid string, cmd Command, payload []byte, trace *TraceConfig) (int, error) {
	session := server.getSession(uuid)
	if session == nil {
		return -1, fmt.Errorf("Unknown UUID %s", uuid)
	}

	frame := session.commandFrame(cmd, payload, trace)
	return session.Write(frame)
}

func (server *Server) sendStatus(uuid string, status Status, payload []byte, trace *TraceConfig) (int, error) {
	session := server.getSession(uuid)
	if session == nil {
		return -1, fmt.Errorf("Unknown UUID %s", uuid)
	}

	frame := session.statusFrame(status, payload, trace)
	return session.Write(frame)
}

func (server *Server) sendEvent(uuid string, event Event, payload []byte, trace *TraceConfig) (int, error) {
	session := server.getSession(uuid)
	if session == nil {
		return -1, fmt.Errorf("Unknown UUID %s", uuid)
	}

	frame := session.eventFrame(event, payload, trace)
	return session.Write(frame)
}

func (server *Server) sendError(uuid string, error Error, payload []byte, trace *TraceConfig) (int, error) {
	session := server.getSession(uuid)
	if session == nil {
		return -1, fmt.Errorf("Unknown UUID %s", uuid)
	}

	frame := session.errorFrame(error, payload, trace)
	return session.Write(frame)
}

// SendCommand sends a specific command and its payload to a client.
// The client is specified by its uuid
func (server *Server) SendCommand(uuid string, cmd Command, payload []byte) (int, error) {
	return server.sendCommand(uuid, cmd, payload, server.trace)
}

// SendStatus sends a specific status and its payload to a client.
// The client is specified by its uuid
func (server *Server) SendStatus(uuid string, status Status, payload []byte) (int, error) {
	return server.sendStatus(uuid, status, payload, server.trace)
}

// SendEvent sends a specific status and its payload to a client.
// The client is specified by its uuid
func (server *Server) SendEvent(uuid string, event Event, payload []byte) (int, error) {
	return server.sendEvent(uuid, event, payload, server.trace)
}

// SendError sends an error back to a client.
// The client is specified by its uuid
func (server *Server) SendError(uuid string, error Error, payload []byte) (int, error) {
	return server.sendError(uuid, error, payload, server.trace)
}

// SendTracedCommand sends a specific command and its payload to a client.
// The SSNTP command frame will be traced according to the trace argument.
// The client is specified by its uuid
func (server *Server) SendTracedCommand(uuid string, cmd Command, payload []byte, trace *TraceConfig) (int, error) {
	return server.sendCommand(uuid, cmd, payload, trace)
}

// SendTracedStatus sends a specific status and its payload to a client.
// The SSNTP status frame will be traced according to the trace argument.
// The client is specified by its uuid
func (server *Server) SendTracedStatus(uuid string, status Status, payload []byte, trace *TraceConfig) (int, error) {
	return server.sendStatus(uuid, status, payload, trace)
}

// SendTracedEvent sends a specific event and its payload to a client.
// The SSNTP event frame will be traced according to the trace argument.
// The client is specified by its uuid
func (server *Server) SendTracedEvent(uuid string, event Event, payload []byte, trace *TraceConfig) (int, error) {
	return server.sendEvent(uuid, event, payload, trace)
}

// SendTracedError sends an error back to a client.
// The SSNTP error frame will be traced according to the trace argument.
// The client is specified by its uuid
func (server *Server) SendTracedError(uuid string, error Error, payload []byte, trace *TraceConfig) (int, error) {
	return server.sendError(uuid, error, payload, trace)
}

// UUID exports the SSNTP server Universally Unique ID.
func (server *Server) UUID() string {
	return server.uuid.String()
}

// ClientRole returns the role of the ssntp session peer with the specified uuid.
func (server *Server) ClientRole(uuid string) (Role, error) {
	server.sessionMutex.RLock()
	session := server.sessions[uuid]
	defer server.sessionMutex.RUnlock()
	if session == nil {
		return UNKNOWN, fmt.Errorf("SSNTP session missing for uuid %s", uuid)
	}
	return session.destRole, nil
}
