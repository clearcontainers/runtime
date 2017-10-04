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

package ssntp_test

import (
	"fmt"
	. "github.com/ciao-project/ciao/ssntp"
)

type logger struct{}

func (l logger) Infof(format string, args ...interface{}) {
	fmt.Printf("INFO: SSNTP Server: "+format, args...)
}

func (l logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: SSNTP Server: "+format, args...)
}

func (l logger) Warningf(format string, args ...interface{}) {
	fmt.Printf("WARNING: SSNTP Server: "+format, args...)
}

type ssntpDumpServer struct {
	ssntp Server
	name  string
}

func (server *ssntpDumpServer) ConnectNotify(uuid string, role Role) {
	fmt.Printf("%s: %s connected (role 0x%x)\n", server.name, uuid, role)
}

func (server *ssntpDumpServer) DisconnectNotify(uuid string, role Role) {
	fmt.Printf("%s: %s disconnected (role 0x%x)\n", server.name, uuid, role)
}

func (server *ssntpDumpServer) StatusNotify(uuid string, status Status, frame *Frame) {
	fmt.Printf("%s: STATUS %s from %s\n", server.name, status, uuid)
}

func (server *ssntpDumpServer) CommandNotify(uuid string, command Command, frame *Frame) {
	fmt.Printf("%s: COMMAND %s from %s\n", server.name, command, uuid)
}

func (server *ssntpDumpServer) EventNotify(uuid string, event Event, frame *Frame) {
	fmt.Printf("%s: EVENT %s from %s\n", server.name, event, uuid)
}

func (server *ssntpDumpServer) ErrorNotify(uuid string, error Error, frame *Frame) {
	fmt.Printf("%s: ERROR (%s) from %s\n", server.name, error, uuid)
}

func (server *ssntpDumpServer) CommandForward(uuid string, command Command, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(agentUUID)

	return
}

func ExampleServer_Serve() {
	var config Config

	server := &ssntpDumpServer{
		name: "CIAO Echo Server",
	}

	config.Log = logger{}
	config.CAcert = "MyServer.crt"
	config.ForwardRules = []FrameForwardRule{

		/* All STATS commands forwarded to Controllers. */
		{
			Operand: STATS,
			Dest:    Controller,
		},

		/* For START commands, server.CommandForward will decide where to forward them. */
		{
			Operand:        START,
			CommandForward: server,
		},
	}

	server.ssntp.Serve(&config, server)
}
