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
	"math/rand"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/01org/ciao/ssntp"
)

type logger struct{}

func (l logger) Infof(format string, args ...interface{}) {
	fmt.Printf("INFO: Client example: "+format, args...)
}

func (l logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: Client example: "+format, args...)
}

func (l logger) Warningf(format string, args ...interface{}) {
	fmt.Printf("WARNING: Client example: "+format, args...)
}

type ssntpClient struct {
	ssntp     ssntp.Client
	name      string
	nCommands int
}

func (client *ssntpClient) ConnectNotify() {
	fmt.Printf("%s connected\n", client.name)
}

func (client *ssntpClient) DisconnectNotify() {
	fmt.Printf("%s disconnected\n", client.name)
}

func (client *ssntpClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
	fmt.Printf("STATUS %s for %s\n", status, client.name)
}

func (client *ssntpClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	client.nCommands++
}

func (client *ssntpClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	fmt.Printf("EVENT %s for %s\n", event, client.name)
}

func (client *ssntpClient) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
	fmt.Printf("ERROR (%s) for %s\n", error, client.name)
}

func clientThread(config *ssntp.Config, n int, threads int, nFrames int, delay int, wg *sync.WaitGroup, payloadLen int) {
	defer wg.Done()

	client := &ssntpClient{
		name:      "CIAO module",
		nCommands: 0,
	}

	payload := make([]byte, payloadLen)

	fmt.Printf("----- Client [%d] delay [%d] frames [%d] payload [%d bytes] -----\n", n, delay, nFrames, payloadLen)

	if threads > 1 {
		source := rand.NewSource(time.Now().UnixNano())
		r := rand.New(source)
		time.Sleep(time.Duration(r.Int63n(2000)) * time.Millisecond)
	}

	if client.ssntp.Dial(config, client) != nil {
		fmt.Printf("Could not connect to an SSNTP server\n")
		return
	}
	fmt.Printf("Client [%d]: Connected\n", n)

	sentFrames := 0
	for i := 0; i < nFrames; i++ {
		_, err := client.ssntp.SendCommand(ssntp.STATS, payload)
		if err != nil {
			fmt.Printf("Could not send STATS: %s\n", err)
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if err == nil {
			sentFrames++
		}
	}

	fmt.Printf("Client [%d]: Done\n", n)

	client.ssntp.Close()

	fmt.Printf("Sent %d commands, received %d\n", sentFrames, client.nCommands)
}

func main() {
	var serverURL = flag.String("url", "localhost", "Server URL")
	var cert = flag.String("cert", "/etc/pki/ciao/client.pem", "Client certificate")
	var CAcert = flag.String("cacert", "/etc/pki/ciao/ca_cert.crt", "CA certificate")
	var nFrames = flag.Int("frames", 10, "Number of frames to send")
	var delay = flag.Int("delay", 500, "Delay(ms) between frames")
	var threads = flag.Int("threads", 1, "Number of client threads")
	var cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
	var payloadLen = flag.Int("payload-len", 0, "Frame payload length")
	var config ssntp.Config
	var wg sync.WaitGroup

	flag.Parse()

	config.URI = *serverURL
	config.CAcert = *CAcert
	config.Cert = *cert
	config.Log = logger{}

	if len(*cpuprofile) != 0 {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Print(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go clientThread(&config, i, *threads, *nFrames, *delay, &wg, *payloadLen)
	}

	wg.Wait()
}
