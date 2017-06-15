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

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
)

type ssntpTestServer struct {
	ssntp        ssntp.Server
	name         string
	nConnections int
	nCommands    int
	nStatuses    int
	nErrors      int
	nEvents      int
}

const cnciUUID = "3390740c-dce9-48d6-b83a-a717417072ce"
const tenantUUID = "2491851d-dce9-48d6-b83a-a717417072ce"
const instanceUUID = "2478251d-dce9-48d6-b83a-a717417072ce"
const agentUUID = "2478711d-dce9-48d6-b83a-a717417072ce"
const cnciIP = "192.168.0.110"
const agentIP = "192.168.0.101"
const instancePublicIP = "10.1.2.3"
const instancePrivateIP = "192.168.0.2"
const vnicMAC = "aa:bb:cc:01:02:03"
const tenantSubnet = "172.16.1.0/24"
const subnetKey = (172 + 16<<8 + 1<<16)

func publicIPCmd() (cmd payloads.PublicIPCommand) {
	cmd.ConcentratorUUID = cnciUUID
	cmd.TenantUUID = tenantUUID
	cmd.InstanceUUID = instanceUUID
	cmd.PublicIP = instancePublicIP
	cmd.PrivateIP = instancePrivateIP
	cmd.VnicMAC = vnicMAC
	return cmd
}

func tenantEvent() (evt payloads.TenantAddedEvent) {
	evt.AgentUUID = agentUUID
	evt.AgentIP = agentIP
	evt.TenantUUID = tenantUUID
	evt.TenantSubnet = tenantSubnet
	evt.ConcentratorUUID = cnciUUID
	evt.ConcentratorIP = cnciIP
	evt.SubnetKey = subnetKey
	return evt
}

func assignPublicIPMarshal() ([]byte, error) {
	assignIP := payloads.CommandAssignPublicIP{
		AssignIP: publicIPCmd(),
	}
	y, err := yaml.Marshal(&assignIP)
	if err != nil {
		return nil, err
	}
	return y, nil
}

func releasePublicIPMarshal() ([]byte, error) {
	releaseIP := payloads.CommandReleasePublicIP{
		ReleaseIP: publicIPCmd(),
	}
	y, err := yaml.Marshal(&releaseIP)
	if err != nil {
		return nil, err
	}
	return y, nil
}

func tenantAddedMarshal() ([]byte, error) {
	tenantAdded := payloads.EventTenantAdded{
		TenantAdded: tenantEvent(),
	}
	y, err := yaml.Marshal(&tenantAdded)
	if err != nil {
		return nil, err
	}
	return y, nil
}

func tenantRemovedMarshal() ([]byte, error) {
	tenantRemoved := payloads.EventTenantRemoved{
		TenantRemoved: tenantEvent(),
	}
	y, err := yaml.Marshal(&tenantRemoved)
	if err != nil {
		return nil, err
	}
	return y, nil
}

type logger struct{}

func (l logger) Infof(format string, args ...interface{}) {
	fmt.Printf("INFO: Test Server: "+format, args...)
}

func (l logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: Test Server: "+format, args...)
}

func (l logger) Warningf(format string, args ...interface{}) {
	fmt.Printf("WARNING: Test Server: "+format, args...)
}

func (server *ssntpTestServer) ConnectNotify(uuid string, role ssntp.Role) {
	server.nConnections++
	fmt.Printf("%s: %s connected (role 0x%x, current connections %d)\n", server.name, uuid, role, server.nConnections)

	//Send out the command and events right here
	//Also create a table to drive this with type, type, payload
	if role == ssntp.CNCIAGENT {
		payload, _ := tenantAddedMarshal()
		_, _ = server.ssntp.SendEvent(uuid, ssntp.TenantAdded, payload)
		time.Sleep(time.Second)

		payload, _ = assignPublicIPMarshal()
		_, _ = server.ssntp.SendCommand(uuid, ssntp.AssignPublicIP, payload)
		time.Sleep(time.Second)

		payload, _ = releasePublicIPMarshal()
		_, _ = server.ssntp.SendCommand(uuid, ssntp.ReleasePublicIP, payload)
		time.Sleep(time.Second)

		payload, _ = tenantRemovedMarshal()
		_, _ = server.ssntp.SendEvent(uuid, ssntp.TenantRemoved, payload)
		time.Sleep(time.Second)

		payload, _ = tenantAddedMarshal()
		_, _ = server.ssntp.SendEvent(uuid, ssntp.TenantAdded, payload)
		time.Sleep(time.Second)
	}

}

func (server *ssntpTestServer) DisconnectNotify(uuid string, role ssntp.Role) {
	server.nConnections--
	fmt.Printf("%s: %s disconnected (current connections %d)\n", server.name, uuid, server.nConnections)
}

func (server *ssntpTestServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
	server.nStatuses++
	fmt.Printf("%s: STATUS (#%d) from %s\n", server.name, server.nStatuses, uuid)
	//server.ssntp.SendStatus(uuid, status, payload)
}

func (server *ssntpTestServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	server.nCommands++
	//server.ssntp.SendCommand(uuid, command, payload)
	fmt.Printf("%s: CMD (#%d) from %s\n", server.name, server.nCommands, uuid)
}

func (server *ssntpTestServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
	payload := frame.Payload

	server.nEvents++
	fmt.Printf("%s: EVENT (#%d)from %s\n", server.name, server.nEvents, uuid)
	if event == ssntp.ConcentratorInstanceAdded {
		var cnciAdded payloads.EventConcentratorInstanceAdded

		err := yaml.Unmarshal(payload, &cnciAdded)
		if err != nil {
			fmt.Printf("Error unmarshaling cnciAdded [%s]\n", err)
		}
		fmt.Printf("CNCI Added Event [%d]\n", len(payload))
		fmt.Printf("instance UUID field [%s] \n", cnciAdded.CNCIAdded.InstanceUUID)
		fmt.Printf("tenant UUID field [%s]", cnciAdded.CNCIAdded.TenantUUID)
		fmt.Printf("CNCI IP field [%s]", cnciAdded.CNCIAdded.ConcentratorIP)
		fmt.Printf("CNCI MAC field [%s]", cnciAdded.CNCIAdded.ConcentratorMAC)
	}
}

func (server *ssntpTestServer) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
	server.nErrors++
	fmt.Printf("%s: ERROR (#%d)from %s\n", server.name, server.nErrors, uuid)
}

func main() {
	var cert = flag.String("cert", "/var/lib/ciao/cert-server-localhost.pem", "Client certificate")
	var CAcert = flag.String("cacert", "/var/lib/ciao/CAcert-server-localhost.pem", "CA certificate")
	var cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
	var config ssntp.Config

	flag.Parse()
	server := &ssntpTestServer{
		name:         "Network Test Server",
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
		_ = pprof.StartCPUProfile(f)
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

	_ = server.ssntp.Serve(&config, server)
}
