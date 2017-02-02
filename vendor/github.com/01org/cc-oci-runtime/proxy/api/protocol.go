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
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

const headerLength = 8 // in bytes

type header struct {
	length uint32
	flags  uint32
}

// A Request is a JSON message sent from a client to the proxy. This message
// embed a payload identified by "id". A payload can have data associated with
// it. It's useful to think of Request as an RPC call with "id" as function
// name and "data" as arguments.
//
// The list of possible payloads are documented in this package.
//
// Each Request has a corresponding Response message sent back from the proxy.
type Request struct {
	ID   string          `json:"id"`
	Data json.RawMessage `json:"data,omitempty"`
}

// A Response is a JSON message sent back from the proxy to a client after a
// Request has been issued. The Response holds the result of the Request,
// including its success state and optional data. It's useful to think of
// Response as the result of an RPC call with ("success", "error") describing
// if the call has been successul and "data" holding the optional results.
type Response struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ReadMessage reads a message from reader. A message is either a Request or a
// Response
func ReadMessage(reader io.Reader, msg interface{}) error {
	buf := make([]byte, headerLength)
	n, err := reader.Read(buf)
	if err != nil {
		return err
	}
	if n != headerLength {
		return errors.New("couldn't read the full header")
	}

	header := header{
		length: binary.BigEndian.Uint32(buf[0:4]),
		flags:  binary.BigEndian.Uint32(buf[4:8]),
	}

	received := 0
	need := int(header.length)
	data := make([]byte, need)
	for received < need {
		n, err := reader.Read(data[received:need])
		if err != nil {
			return err
		}

		received += n
	}

	err = json.Unmarshal(data, msg)
	if err != nil {
		return err
	}

	return nil
}

// WriteMessage writes a message into writer. A message is either a Request for
// a Response
func WriteMessage(writer io.Writer, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

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
