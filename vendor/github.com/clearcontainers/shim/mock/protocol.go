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
	"errors"
	"fmt"
	"net"

	"github.com/clearcontainers/proxy/api"
)

// XXX: could do with its own package to remove that ugly namespacing
type protocolHandler func([]byte, interface{}, *handlerResponse)

// Encapsulates the different parts of what a handler can return.
type handlerResponse struct {
	err     error
	results map[string]interface{}
}

// SetError indicates sets error for the response.
func (r *handlerResponse) SetError(err error) {
	r.err = err
}

// SetErrorMsg sets an error with the passed string for the response.
func (r *handlerResponse) SetErrorMsg(msg string) {
	r.err = errors.New(msg)
}

// SetErrorf sets an error with the formatted string for the response.
func (r *handlerResponse) SetErrorf(format string, a ...interface{}) {
	r.SetError(fmt.Errorf(format, a...))
}

// AddResult adds the given key/val to the response.
func (r *handlerResponse) AddResult(key string, value interface{}) {
	if r.results == nil {
		r.results = make(map[string]interface{})
	}
	r.results[key] = value
}

// FrameKey is a struct composed of the the frame type and opcode,
// used as a key for retrieving the handler for handling the frame.
type FrameKey struct {
	ftype  api.FrameType
	opcode int
}

func newFrameKey(frameType api.FrameType, opcode int) FrameKey {
	return FrameKey{
		ftype:  frameType,
		opcode: opcode,
	}
}

type protocol struct {
	cmdHandlers map[FrameKey]protocolHandler
}

func newProtocol() *protocol {
	return &protocol{
		cmdHandlers: make(map[FrameKey]protocolHandler),
	}
}

// Handle retreives the handler for handling the frame
func (proto *protocol) Handle(key FrameKey, handler protocolHandler) bool {
	if _, ok := proto.cmdHandlers[key]; ok {
		return false
	}
	proto.cmdHandlers[key] = handler
	return true
}

type clientCtx struct {
	conn     net.Conn
	userData interface{}
}

func newErrorResponse(opcode int, errMsg string) *api.Frame {
	frame, err := api.NewFrameJSON(api.TypeResponse, opcode, &api.ErrorResponse{
		Message: errMsg,
	})

	if err != nil {
		frame, err = api.NewFrameJSON(api.TypeResponse, opcode, &api.ErrorResponse{
			Message: fmt.Sprintf("couldn't marshal response: %v", err),
		})
		if err != nil {
			frame = api.NewFrame(api.TypeResponse, opcode, nil)
		}
	}

	frame.Header.InError = true
	return frame
}

func (proto *protocol) handleCommand(ctx *clientCtx, cmd *api.Frame) *api.Frame {
	hr := handlerResponse{}

	// cmd.Header.Opcode is guaranteed to be within the right bounds by
	// ReadFrame().
	handler := proto.cmdHandlers[FrameKey{cmd.Header.Type, int(cmd.Header.Opcode)}]

	handler(cmd.Payload, ctx.userData, &hr)
	if hr.err != nil {
		return newErrorResponse(cmd.Header.Opcode, hr.err.Error())
	}

	var payload interface{}
	if len(hr.results) > 0 {
		payload = hr.results
	}

	frame, err := api.NewFrameJSON(api.TypeResponse, cmd.Header.Opcode, payload)
	if err != nil {
		return newErrorResponse(cmd.Header.Opcode, err.Error())
	}
	return frame
}

// Serve serves the client connection in a continuous loop.
func (proto *protocol) Serve(conn net.Conn, userData interface{}) error {
	ctx := &clientCtx{
		conn:     conn,
		userData: userData,
	}

	for {
		frame, err := api.ReadFrame(conn)
		if err != nil {
			// EOF or the client isn't even sending proper JSON,
			// just kill the connection
			fmt.Printf("Finished serving client: %v", err)
			return err
		}

		if frame.Header.Type != api.TypeCommand && frame.Header.Type != api.TypeStream {
			// EOF or the client isn't even sending proper JSON,
			// just kill the connection
			return fmt.Errorf("serve: expected a command got a %v", frame.Header.Type)
		}

		// Execute the corresponding handler
		resp := proto.handleCommand(ctx, frame)

		// Send the response back to the client.
		if err = WriteFrame(conn, resp); err != nil {
			// Something made us unable to write the response back
			// to the client (could be a disconnection, ...).
			return err
		}
	}
}
