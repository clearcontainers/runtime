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
	"os"

	"github.com/01org/cc-oci-runtime/proxy/api"
)

// XXX: could do with its own package to remove that ugly namespacing
type protocolHandler func([]byte, interface{}, *handlerResponse)

// Encapsulates the different parts of what a handler can return.
type handlerResponse struct {
	err     error
	results map[string]interface{}
	file    *os.File
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

func (r *handlerResponse) SetFile(f *os.File) {
	r.file = f
}

type protocol struct {
	handlers map[string]protocolHandler
}

func newProtocol() *protocol {
	return &protocol{
		handlers: make(map[string]protocolHandler),
	}
}

func (proto *protocol) Handle(cmd string, handler protocolHandler) {
	proto.handlers[cmd] = handler
}

type clientCtx struct {
	conn net.Conn

	userData interface{}
}

func (proto *protocol) handleRequest(ctx *clientCtx, req *api.Request, hr *handlerResponse) *api.Response {
	if req.ID == "" {
		return &api.Response{
			Success: false,
			Error:   "no 'id' field in request",
		}
	}

	handler, ok := proto.handlers[req.ID]
	if !ok {
		return &api.Response{
			Success: false,
			Error:   fmt.Sprintf("no payload named '%s'", req.ID),
		}
	}

	handler(req.Data, ctx.userData, hr)
	if hr.err != nil {
		return &api.Response{
			Success: false,
			Error:   hr.err.Error(),
			Data:    hr.results,
		}
	}

	return &api.Response{
		Success: true,
		Data:    hr.results,
	}
}

func (proto *protocol) Serve(conn net.Conn, userData interface{}) error {
	ctx := &clientCtx{
		conn:     conn,
		userData: userData,
	}

	for {
		// Parse a request.
		req := api.Request{}
		hr := handlerResponse{}

		err := api.ReadMessage(conn, &req)
		if err != nil {
			// EOF or the client isn't even sending proper JSON,
			// just kill the connection
			return err
		}

		// Execute the corresponding handler
		resp := proto.handleRequest(ctx, &req, &hr)

		// Send the response back to the client.
		if err = api.WriteMessage(conn, resp); err != nil {
			// Something made us unable to write the response back
			// to the client (could be a disconnection, ...).
			return err
		}

		// And send a fd if the handler associated a file with the response
		if hr.file != nil {
			if err = api.WriteFd(conn.(*net.UnixConn), int(hr.file.Fd())); err != nil {
				hr.file.Close()
				return err
			}
			hr.file.Close()
		}

	}
}
