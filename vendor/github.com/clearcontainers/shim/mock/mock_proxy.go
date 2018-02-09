// Copyright (c) 2017 Intel Corporation
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

package mock

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/clearcontainers/proxy/api"
	"github.com/stretchr/testify/assert"
)

const testContainerid = "123456789"
const testToken = "pF56IaDpuax6hihJ5PneB8JypqmOvjkqY-wKGVYqgIM="

// Proxy is an object mocking clearcontainers Proxy
type Proxy struct {
	t              *testing.T
	wg             sync.WaitGroup
	connectionPath string

	// proxy socket
	listener net.Listener

	// single client to serve
	cl net.Conn

	//token to be used for the connection
	token string

	lastStdinStream []byte

	ShimConnected    chan bool
	Signal           chan ShimSignal
	ShimDisconnected chan bool
	StdinReceived    chan bool
}

// NewProxy creates a hyperstart instance
func NewProxy(t *testing.T, path string) *Proxy {
	return &Proxy{
		t:                t,
		connectionPath:   path,
		lastStdinStream:  nil,
		Signal:           make(chan ShimSignal, 5),
		ShimConnected:    make(chan bool),
		ShimDisconnected: make(chan bool),
		StdinReceived:    make(chan bool),
		token:            testToken,
	}
}

// GetProxyToken returns the token that mock proxy uses
// to verify its client connection
func (proxy *Proxy) GetProxyToken() string {
	return proxy.token
}

func newSignalList() []ShimSignal {
	return make([]ShimSignal, 0, 5)
}

// GetLastStdinStream returns the last received stdin stream
func (proxy *Proxy) GetLastStdinStream() []byte {
	return proxy.lastStdinStream
}

func (proxy *Proxy) log(s string) {
	proxy.logF("%s\n", s)
}

func (proxy *Proxy) logF(format string, args ...interface{}) {
	proxy.t.Logf("[Proxy] "+format, args...)
}

type client struct {
	proxy *Proxy
	conn  net.Conn
}

// ConnectShim payload defined here, as it has not been defined
// in proxy api package yet
type ConnectShim struct {
	Token string `json:"token"`
}

// ShimSignal is the struct used to represent the signal received from the shim
type ShimSignal struct {
	Signal int `json:"signalNumber"`
	Row    int `json:"rows,omitempty"`
	Column int `json:"columns,omitempty"`
}

func connectShimHandler(data []byte, userData interface{}, response *handlerResponse) {
	client := userData.(*client)
	proxy := client.proxy

	payload := ConnectShim{}
	err := json.Unmarshal(data, &payload)
	assert.Nil(proxy.t, err)

	if payload.Token != proxy.token {
		response.SetErrorMsg("Invalid Token")
	}

	proxy.logF("ConnectShim(token=%s)", payload.Token)

	response.AddResult("version", api.Version)
	proxy.ShimConnected <- true
}

func signalShimHandler(data []byte, userData interface{}, response *handlerResponse) {
	client := userData.(*client)
	proxy := client.proxy

	signalPayload := ShimSignal{}
	err := json.Unmarshal(data, &signalPayload)
	assert.Nil(proxy.t, err)

	proxy.logF("Proxy received signal: %v", signalPayload)

	proxy.Signal <- signalPayload
}

func disconnectShimHandler(data []byte, userData interface{}, response *handlerResponse) {
	client := userData.(*client)
	proxy := client.proxy

	proxy.log("Client sent DisconnectShim Command")
	proxy.ShimDisconnected <- true
}

func stdinShimHandler(data []byte, userData interface{}, response *handlerResponse) {
	client := userData.(*client)
	proxy := client.proxy

	proxy.lastStdinStream = data
	proxy.StdinReceived <- true
}

// SendStdoutStream sends a Stdout Stream Frame to connected client
func (proxy *Proxy) SendStdoutStream(payload []byte) {
	err := api.WriteStream(proxy.cl, api.StreamStdout, payload)
	assert.Nil(proxy.t, err)
}

// SendStderrStream sends a Stderr Stream Frame to connected client
func (proxy *Proxy) SendStderrStream(payload []byte) {
	err := api.WriteStream(proxy.cl, api.StreamStderr, payload)
	assert.Nil(proxy.t, err)
}

// SendExitNotification sends an Exit Notification Frame to connected client
func (proxy *Proxy) SendExitNotification(payload []byte) {
	err := api.WriteNotification(proxy.cl, api.NotificationProcessExited, payload)
	assert.Nil(proxy.t, err)
}

func (proxy *Proxy) startListening() {

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: proxy.connectionPath, Net: "unix"})
	assert.Nil(proxy.t, err)

	proxy.logF("listening on %s", proxy.connectionPath)

	proxy.listener = l
}

func (proxy *Proxy) serveClient(proto *protocol, newConn net.Conn) {
	newClient := &client{
		proxy: proxy,
		conn:  newConn,
	}
	err := proto.Serve(newConn, newClient)
	proxy.logF("Error serving client : %v\n", err)

	newConn.Close()
	proxy.log("Client closed connection")

	proxy.wg.Done()
}

func (proxy *Proxy) serve() {
	proto := newProtocol()

	proto.Handle(FrameKey{api.TypeCommand, int(api.CmdConnectShim)}, connectShimHandler)
	proto.Handle(FrameKey{api.TypeCommand, int(api.CmdDisconnectShim)}, disconnectShimHandler)
	proto.Handle(FrameKey{api.TypeCommand, int(api.CmdSignal)}, signalShimHandler)
	proto.Handle(FrameKey{api.TypeCommand, int(api.CmdConnectShim)}, connectShimHandler)

	proto.Handle(FrameKey{api.TypeStream, int(api.StreamStdin)}, stdinShimHandler)

	//Wait for a single client connection
	conn, err := proxy.listener.Accept()
	assert.Nil(proxy.t, err)
	assert.NotNil(proxy.t, conn)
	proxy.log("Client connected")

	proxy.cl = conn

	proxy.serveClient(proto, conn)
}

// Start invokes mock proxy instance to start listening.
func (proxy *Proxy) Start() {
	proxy.startListening()
	proxy.wg.Add(1)
	go proxy.serve()
}

// Stop causes  mock proxy instance to stop listening,
// close connection to client and close all channels
func (proxy *Proxy) Stop() {
	proxy.listener.Close()

	if proxy.cl != nil {
		proxy.log("Closing client connection")
		proxy.cl.Close()
		proxy.cl = nil
	} else {
		proxy.log("Client connection already closed")
	}

	proxy.wg.Wait()
	close(proxy.ShimConnected)
	close(proxy.Signal)
	close(proxy.ShimDisconnected)
	close(proxy.StdinReceived)
	os.Remove(proxy.connectionPath)
	proxy.log("Stopped")
}

const (
	flagInError = 1 << (4 + iota)
)

const minHeaderLength = 12

// WriteFrame reimplemented here till the one in api package
// is implemented as goroutine-safe
func WriteFrame(w io.Writer, frame *api.Frame) error {
	header := &frame.Header

	if len(frame.Payload) < header.PayloadLength {
		return fmt.Errorf("frame: bad payload length %d",
			header.PayloadLength)
	}

	// Prepare the header.
	len := minHeaderLength + header.PayloadLength
	buf := make([]byte, len)
	binary.BigEndian.PutUint16(buf[0:2], uint16(header.Version))
	buf[2] = byte(header.HeaderLength / 4)
	flags := byte(0)
	if frame.Header.InError {
		flags |= flagInError
	}
	buf[6] = flags | byte(header.Type)&0xf
	buf[7] = byte(header.Opcode)
	binary.BigEndian.PutUint32(buf[8:8+4], uint32(header.PayloadLength))

	// Write payload if needed
	if header.PayloadLength > 0 {
		copy(buf[minHeaderLength:], frame.Payload[0:header.PayloadLength])
	}

	n, err := w.Write(buf)
	if err != nil {
		return err
	}

	if n != len {
		return errors.New("frame: couldn't write frame")
	}

	return nil
}
