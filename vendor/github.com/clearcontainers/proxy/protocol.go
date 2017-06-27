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
	"errors"
	"fmt"
	"net"

	"github.com/clearcontainers/proxy/api"
)

type commandHandler func([]byte, interface{}, *handlerResponse)

// Encapsulates the different parts of what a handler can return.
type handlerResponse struct {
	err     error
	results map[string]interface{}
	data    []byte
}

func (r *handlerResponse) SetError(err error) {
	r.err = err
}

func (r *handlerResponse) SetErrorMsg(msg string) {
	r.err = errors.New(msg)
}

func (r *handlerResponse) SetErrorf(format string, a ...interface{}) {
	r.SetError(fmt.Errorf(format, a...))
}

func (r *handlerResponse) AddResult(key string, value interface{}) {
	if r.results == nil {
		r.results = make(map[string]interface{})
	}
	r.results[key] = value
}

// SetData sets the data to be sent as the response payload. If both AddResult
// and SetData are called on the same handlerResponse, SetData takes precedence
// and defines what we be returned to the caller.
func (r *handlerResponse) SetData(data []byte) {
	r.data = data
}

// streamHandler is the prototype of function that can be registered to be
// called when receiving a stream frame
type streamHandler func(frame *api.Frame, userData interface{}) error

type protocol struct {
	cmdHandlers    [api.CmdMax]commandHandler
	streamHandlers [api.StreamMax]streamHandler
}

func newProtocol() *protocol {
	return &protocol{}
}

func (proto *protocol) HandleCommand(cmd api.Command, handler commandHandler) {
	proto.cmdHandlers[cmd] = handler
}

// HandleStream registers a callback to call when the protocol receives a
// stream frame of kind stream. The callback is called from a goroutine
// internal to proto.
func (proto *protocol) HandleStream(stream api.Stream, handler streamHandler) {
	proto.streamHandlers[stream] = handler
}

type clientCtx struct {
	conn net.Conn

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
	}
	if err != nil {
		frame = api.NewFrame(api.TypeResponse, opcode, nil)
	}

	frame.Header.InError = true

	return frame
}

func (proto *protocol) handleCommand(ctx *clientCtx, cmd *api.Frame) *api.Frame {
	hr := handlerResponse{}

	// cmd.Header.Opcode is guaranteed to be within the right bounds by
	// ReadFrame().
	handler := proto.cmdHandlers[cmd.Header.Opcode]
	if handler == nil {
		errMsg := fmt.Sprintf("no handler for command %s",
			api.Command(cmd.Header.Opcode))
		return newErrorResponse(cmd.Header.Opcode, errMsg)
	}

	handler(cmd.Payload, ctx.userData, &hr)
	if hr.err != nil {
		return newErrorResponse(cmd.Header.Opcode, hr.err.Error())
	}

	var frame *api.Frame

	if len(hr.data) > 0 {
		// We have a full payload defined.
		frame = api.NewFrame(api.TypeResponse, cmd.Header.Opcode, hr.data)
	} else {
		// Otherwise, we'll marshal hr.results, if we have any.
		var payload interface{}
		var err error

		if len(hr.results) > 0 {
			payload = hr.results
		}
		frame, err = api.NewFrameJSON(api.TypeResponse, cmd.Header.Opcode, payload)
		if err != nil {
			return newErrorResponse(cmd.Header.Opcode, err.Error())
		}
	}

	return frame
}

func (proto *protocol) handlerStream(ctx *clientCtx, frame *api.Frame) error {
	// cmd.Header.Opcode is guaranteed to be within the right bounds by
	// ReadFrame().
	handler := proto.streamHandlers[frame.Header.Opcode]
	if handler == nil {
		return fmt.Errorf("no handler for stream %s",
			api.Stream(frame.Header.Opcode))
	}

	return handler(frame, ctx.userData)
}

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
			return err
		}

		switch frame.Header.Type {
		case api.TypeCommand:
			// Execute the corresponding handler
			resp := proto.handleCommand(ctx, frame)

			// Send the response back to the client.
			if err = api.WriteFrame(conn, resp); err != nil {
				// Something made us unable to write the response back
				// to the client (could be a disconnection, ...).
				return err
			}
		case api.TypeStream:
			if err = proto.handlerStream(ctx, frame); err != nil {
				return err
			}
		default:
			return fmt.Errorf("protocol: unexpected frame type (%v)", frame.Header.Type)
		}
	}
}
