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

package api

import (
	"encoding/json"
	"errors"
	"net"
	"os"
)

// The Client struct can be used to issue proxy API calls with a convenient
// high level API.
type Client struct {
	conn *net.UnixConn
}

// NewClient creates a new client object to communicate with the proxy using
// the connection conn. The user should call Close() once finished with the
// client object to close conn.
func NewClient(conn *net.UnixConn) *Client {
	return &Client{
		conn: conn,
	}
}

// Close a client, closing the underlying AF_UNIX socket.
func (client *Client) Close() {
	client.conn.Close()
}

func (client *Client) sendPayload(id string, payload interface{}) (*Response, error) {
	var err error

	req := Request{}
	req.ID = id
	if payload != nil {
		if req.Data, err = json.Marshal(payload); err != nil {
			return nil, err
		}
	}

	if err := WriteMessage(client.conn, &req); err != nil {
		return nil, err
	}

	resp := Response{}
	if err := ReadMessage(client.conn, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func errorFromResponse(resp *Response) error {
	// We should always have an error with the response, but better safe
	// than sorry.
	if resp.Success == false {
		if resp.Error != "" {
			return errors.New(resp.Error)
		}

		return errors.New("unknown error")
	}

	return nil
}

// HelloOptions holds extra arguments one can pass to the Hello function. See
// the Hello payload for more details.
type HelloOptions struct {
	Console string
}

// HelloReturn contains the return values from Hello. See the Hello and
// HelloResult payloads.
type HelloReturn struct {
	Version int
}

// Hello wraps the Hello payload (see payload description for more details)
func (client *Client) Hello(containerID, ctlSerial, ioSerial string,
	options *HelloOptions) (*HelloReturn, error) {
	hello := Hello{
		ContainerID: containerID,
		CtlSerial:   ctlSerial,
		IoSerial:    ioSerial,
	}

	if options != nil {
		hello.Console = options.Console
	}

	resp, err := client.sendPayload("hello", &hello)
	if err != nil {
		return nil, err
	}

	ret := &HelloReturn{}

	val, ok := resp.Data["version"]
	if !ok {
		return nil, errors.New("hello: no version in response")
	}
	ret.Version = int(val.(float64))

	return ret, errorFromResponse(resp)
}

// AttachOptions holds extra arguments one can pass to the Attach function. See
// the Attach payload for more details.
type AttachOptions struct {
}

// AttachReturn contains the return values from Hello. See the Hello and
// AttachResult payloads.
type AttachReturn struct {
	Version int
}

// Attach wraps the Attach payload (see payload description for more details)
func (client *Client) Attach(containerID string, options *AttachOptions) (*AttachReturn, error) {
	hello := Attach{
		ContainerID: containerID,
	}

	resp, err := client.sendPayload("attach", &hello)
	if err != nil {
		return nil, err
	}

	ret := &AttachReturn{}

	val, ok := resp.Data["version"]
	if !ok {
		return nil, errors.New("attach: no version in response")
	}
	ret.Version = int(val.(float64))

	return ret, errorFromResponse(resp)
}

// AllocateIo wraps the AllocateIo payload (see payload description for more details)
func (client *Client) AllocateIo(nStreams int) (ioBase uint64, ioFile *os.File, err error) {
	allocate := AllocateIo{
		NStreams: nStreams,
	}

	resp, err := client.sendPayload("allocateIO", &allocate)
	if err != nil {
		return
	}

	err = errorFromResponse(resp)
	if err != nil {
		return
	}

	val, ok := resp.Data["ioBase"]
	if !ok {
		return 0, nil, errors.New("allocateio: no ioBase in response")
	}

	ioBase = (uint64)(val.(float64))

	// I/O fd
	newFd, err := ReadFd(client.conn)
	if err != nil {
		return 0, nil, errors.New("allocateio: couldn't read fd")
	}

	ioFile = os.NewFile(uintptr(newFd), "")

	return
}

// Hyper wraps the Hyper payload (see payload description for more details)
func (client *Client) Hyper(hyperName string, hyperMessage interface{}) error {
	var data []byte

	if hyperMessage != nil {
		var err error

		data, err = json.Marshal(hyperMessage)
		if err != nil {
			return err
		}
	}

	hyper := Hyper{
		HyperName: hyperName,
		Data:      data,
	}

	resp, err := client.sendPayload("hyper", &hyper)
	if err != nil {
		return err
	}

	return errorFromResponse(resp)
}

// Bye wraps the Bye payload (see payload description for more details)
func (client *Client) Bye(containerID string) error {
	bye := Bye{
		ContainerID: containerID,
	}

	resp, err := client.sendPayload("bye", &bye)
	if err != nil {
		return err
	}

	return errorFromResponse(resp)
}
