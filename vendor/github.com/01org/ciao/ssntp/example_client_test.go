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
	"time"
)

type ssntpEchoClient struct {
	ssntp Client
	name  string
}

func (client *ssntpEchoClient) ConnectNotify() {
	fmt.Printf("%s Connected\n", client.name)
}

func (client *ssntpEchoClient) DisconnectNotify() {
	fmt.Printf("%s disconnected\n", client.name)
}

func (client *ssntpEchoClient) StatusNotify(status Status, frame *Frame) {
	n, err := client.ssntp.SendStatus(status, frame.Payload)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	fmt.Printf("Echoed %d status bytes\n", n)
}

func (client *ssntpEchoClient) CommandNotify(command Command, frame *Frame) {
	n, err := client.ssntp.SendCommand(command, frame.Payload)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	fmt.Printf("Echoed %d command bytes\n", n)
}

func (client *ssntpEchoClient) EventNotify(event Event, frame *Frame) {
	n, err := client.ssntp.SendEvent(event, frame.Payload)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	fmt.Printf("Echoed %d event bytes\n", n)
}

func (client *ssntpEchoClient) ErrorNotify(error Error, frame *Frame) {
	fmt.Printf("ERROR %s\n", error)
}

func ExampleClient_Dial() {
	var config Config

	client := &ssntpEchoClient{
		name: "CIAO Agent",
	}

	config.URI = "myCIAOserver.local"
	config.CAcert = "CIAOCA.crt"
	config.Cert = "agent.pem"

	if client.ssntp.Dial(&config, client) != nil {
		fmt.Printf("Could not connect to an SSNTP server\n")
		return
	}

	// Loop and wait for notifications
	for {
		time.Sleep(time.Duration(10) * time.Second)
	}
}
