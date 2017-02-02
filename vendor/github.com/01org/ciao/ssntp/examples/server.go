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
// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/01org/ciao/ssntp"
)

type ssntpEchoServer struct {
	ssntp        ssntp.Server
	name         string
	nConnections int
	nCommands    int
	nStatuses    int
	nErrors      int
}

type logger struct{}

func (l logger) Infof(format string, args ...interface{}) {
	fmt.Printf("INFO: Server example: "+format, args...)
}

func (l logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: Server example: "+format, args...)
}

func (l logger) Warningf(format string, args ...interface{}) {
	fmt.Printf("WARNING: Server example: "+format, args...)
}

func (server *ssntpEchoServer) ConnectNotify(uuid string, role ssntp.Role) {
	server.nConnections++
	fmt.Printf("%s: %s connected (role 0x%x, current connections %d)\n", server.name, uuid, role, server.nConnections)
}

func (server *ssntpEchoServer) DisconnectNotify(uuid string, role ssntp.Role) {
	server.nConnections--
	fmt.Printf("%s: %s disconnected (role 0x%x, current connections %d)\n", server.name, uuid, role, server.nConnections)
}

func (server *ssntpEchoServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
	server.nStatuses++
	fmt.Printf("%s: STATUS (#%d) from %s\n", server.name, server.nStatuses, uuid)

	server.ssntp.SendStatus(uuid, status, frame.Payload)
}

func (server *ssntpEchoServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	server.nCommands++

	server.ssntp.SendCommand(uuid, command, frame.Payload)
}

func (server *ssntpEchoServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
}

func (server *ssntpEchoServer) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
	server.nErrors++
	fmt.Printf("%s: ERROR (#%d)from %s\n", server.name, server.nErrors, uuid)
}

func main() {
	var cert = flag.String("cert", "/etc/pki/ciao/client.pem", "Client certificate")
	var CAcert = flag.String("cacert", "/etc/pki/ciao/ca_cert.crt", "CA certificate")
	var cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
	var config ssntp.Config

	flag.Parse()
	server := &ssntpEchoServer{
		name:         "CIAO Echo Server",
		nConnections: 0,
		nCommands:    0,
		nStatuses:    0,
		nErrors:      0,
	}

	if len(*cpuprofile) != 0 {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Print(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	config.Log = logger{}
	config.CAcert = *CAcert
	config.Cert = *cert
	// config.DebugInterface = true
	// Forward STATS to all Controllers
	config.ForwardRules = []ssntp.FrameForwardRule{
		{
			Operand: ssntp.STATS,
			Dest:    ssntp.Controller,
		},
	}

	server.ssntp.Serve(&config, server)
}
