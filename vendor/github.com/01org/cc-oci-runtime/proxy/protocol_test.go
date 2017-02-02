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

package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A simple way to mock a net.Conn around syscall.socketpair()
type mockServer struct {
	t                      *testing.T
	proto                  *protocol
	serverConn, clientConn net.Conn
}

func newMockServer(t *testing.T, proto *protocol) *mockServer {
	var err error

	server := &mockServer{
		t:     t,
		proto: proto,
	}

	server.serverConn, server.clientConn, err = Socketpair()
	assert.Nil(t, err)

	return server
}

func (server *mockServer) GetClientConn() net.Conn {
	return server.clientConn
}

func (server *mockServer) Serve() {
	server.ServeWithUserData(nil)
}

func (server *mockServer) ServeWithUserData(userData interface{}) {
	if err := server.proto.Serve(server.serverConn, userData); err != nil {
		server.serverConn.Close()
	}

}

func setupMockServer(t *testing.T, proto *protocol) (client net.Conn, server *mockServer) {
	server = newMockServer(t, proto)
	client = server.GetClientConn()
	go server.Serve()

	return client, server
}

// Low level read/write (header + data)
const headerLength = 8 // bytes

func writeMessage(writer io.Writer, data []byte) error {
	buf := make([]byte, headerLength)
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(data)))
	n, err := writer.Write(buf)
	if err != nil {
		return err
	}
	if n != headerLength {
		return errors.New("couldn't write the full header")
	}

	n, err = writer.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.New("couldn't write the full data")
	}

	return nil
}

func readMessage(reader io.Reader) ([]byte, error) {
	buf := make([]byte, headerLength)
	n, err := reader.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != headerLength {
		return nil, errors.New("couldn't read the full header")
	}

	received := 0
	need := int(binary.BigEndian.Uint32(buf[0:4]))
	data := make([]byte, need)
	for received < need {
		n, err := reader.Read(data[received:need])
		if err != nil {
			return nil, err
		}

		received += n
	}

	return data, nil
}

// Test that we correctly give back the user data to handlers
type myUserData struct {
	t  *testing.T
	wg sync.WaitGroup
}

var testUserData myUserData

func userDataHandler(data []byte, userData interface{}, response *handlerResponse) {
	p := userData.(*myUserData)
	assert.Equal(p.t, p, &testUserData)

	p.wg.Done()
}

func TestUserData(t *testing.T) {
	proto := newProtocol()
	proto.Handle("foo", userDataHandler)

	server := newMockServer(t, proto)
	client := server.GetClientConn()
	testUserData.t = t
	go server.ServeWithUserData(&testUserData)

	testUserData.wg.Add(1)
	err := writeMessage(client, []byte(`{ "id": "foo" }`))
	assert.Nil(t, err)

	// make sure the handler runs by waiting for it
	testUserData.wg.Wait()
}

// Tests various behaviours of the protocol main loop and handler dispatching
func simpleHandler(data []byte, userData interface{}, response *handlerResponse) {
}

type Echo struct {
	Arg string
}

func echoHandler(data []byte, userData interface{}, response *handlerResponse) {
	echo := Echo{}
	json.Unmarshal(data, &echo)

	response.AddResult("result", echo.Arg)
}

func returnDataHandler(data []byte, userData interface{}, response *handlerResponse) {
	response.AddResult("foo", "bar")
}

func returnErrorHandler(data []byte, userData interface{}, response *handlerResponse) {
	response.SetErrorMsg("This is an error")
}

func returnDataErrorHandler(data []byte, userData interface{}, response *handlerResponse) {
	response.AddResult("foo", "bar")
	response.SetErrorMsg("This is an error")
}

func TestProtocol(t *testing.T) {
	tests := []struct {
		input, output string
	}{
		{`{"id": "simple"}`, `{"success":true}`},
		{`{"id": "notfound"}`,
			`{"success":false,"error":"no payload named 'notfound'"}`},
		{`{"foo": "bar"}`,
			`{"success":false,"error":"no 'id' field in request"}`},
		// Tests return values from handlers
		{`{"id":"returnData", "data": {"arg": "bar"}}`,
			`{"success":true,"data":{"foo":"bar"}}`},
		{`{"id":"returnError" }`,
			`{"success":false,"error":"This is an error"}`},
		{`{"id":"returnDataError", "data": {"arg": "bar"}}`,
			`{"success":false,"error":"This is an error","data":{"foo":"bar"}}`},
		// Tests we can unmarshal payload data
		{`{"id":"echo", "data": {"arg": "ping"}}`,
			`{"success":true,"data":{"result":"ping"}}`},
	}

	proto := newProtocol()
	proto.Handle("simple", simpleHandler)
	proto.Handle("returnData", returnDataHandler)
	proto.Handle("returnError", returnErrorHandler)
	proto.Handle("returnDataError", returnDataErrorHandler)
	proto.Handle("echo", echoHandler)

	client, _ := setupMockServer(t, proto)

	for _, test := range tests {
		// request
		err := writeMessage(client, []byte(test.input))
		assert.Nil(t, err)

		// response
		buf, err := readMessage(client)
		assert.Nil(t, err)
		assert.Equal(t, test.output, string(buf))
	}
}

// Make sure the server closes the connection when encountering an error
func TestCloseOnError(t *testing.T) {
	proto := newProtocol()
	proto.Handle("simple", simpleHandler)

	client, _ := setupMockServer(t, proto)

	// request
	const garbage string = "sekjewr"
	err := writeMessage(client, []byte(garbage))
	assert.Nil(t, err)

	// response
	buf := make([]byte, 512)
	_, err = client.Read(buf)
	assert.Equal(t, err, io.EOF)
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
