/*
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
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path"
	"reflect"
	"sync"
	"testing"
	"time"
)

var imageInfoTestGood = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 865M (907018240 bytes)
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoTestMissingBytes = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 865M
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoTestMissingLine = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoTooBig = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 18,446,744,073,710M (18446744073709551615 bytes)
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

var imageInfoBadBytes = `
image: /var/lib/ciao/images/clear-8780-cloud.img
file format: qcow2
virtual size: 865M (9aaaa07018240 bytes)
disk size: 113M
cluster_size: 65536
Format specific information:
    compat: 1.1
    lazy refcounts: false
    refcount bits: 16
    corrupt: false
`

func TestExtractImageInfo(t *testing.T) {
	tests := []struct {
		name   string
		result int
		data   string
	}{
		{
			"imageInfoTestGood",
			865,
			imageInfoTestGood,
		},
		{
			"imageInfoTestMissingBytes",
			-1,
			imageInfoTestMissingBytes,
		},
		{
			"imageInfoTestMissingLine",
			-1,
			imageInfoTestMissingLine,
		},
		{
			"imageInfoTooBig",
			-1,
			imageInfoTooBig,
		},
		{
			"imageInfoBadBytes",
			-1,
			imageInfoBadBytes,
		},
	}

	for _, ti := range tests {
		b := bytes.NewBuffer([]byte(ti.data))
		result := extractImageInfo(b)
		if result != ti.result {
			t.Fatalf("%s failed. %d != %d", ti.name, result, ti.result)
		}
	}
}

func genQEMUParams(networkParams []string) []string {
	baseParams := []string{
		"-drive",
		"file=/var/lib/ciao/instance/1/seed.iso,if=virtio,media=cdrom",
	}
	baseParams = append(baseParams, networkParams...)
	baseParams = append(baseParams, "-enable-kvm", "-cpu", "host", "-daemonize",
		"-qmp", "unix:/var/lib/ciao/instance/1/socket,server,nowait")

	return baseParams
}

func TestGenerateQEMULaunchParams(t *testing.T) {
	var cfg vmConfig

	params := genQEMUParams(nil)
	cfg.Legacy = false
	cfg.Mem = 0
	cfg.Cpus = 0
	params = append(params, "-bios", qemuEfiFw)
	genParams := generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil, "ciao")
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	params = genQEMUParams(nil)
	cfg.Mem = 100
	cfg.Cpus = 0
	cfg.Legacy = true
	params = append(params, "-m", "100")
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil, "ciao")
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	params = genQEMUParams(nil)
	cfg.Mem = 0
	cfg.Cpus = 4
	cfg.Legacy = true
	params = append(params, "-smp", "cpus=4")
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", nil, "ciao")
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}

	netParams := []string{"-net", "nic,model=virtio", "-net", "user"}
	params = genQEMUParams(netParams)
	cfg.Mem = 0
	cfg.Cpus = 0
	cfg.Legacy = true
	genParams = generateQEMULaunchParams(&cfg, "/var/lib/ciao/instance/1/seed.iso",
		"/var/lib/ciao/instance/1", netParams, "ciao")
	if !reflect.DeepEqual(params, genParams) {
		t.Fatalf("%s and %s do not match", params, genParams)
	}
}

func TestQmpConnectBadSocket(t *testing.T) {
	var wg sync.WaitGroup
	qmpChannel := make(chan interface{})
	closedCh := make(chan struct{})
	connectedCh := make(chan struct{})
	instance := "testInstance"
	instanceDir := path.Join("/tmp", instance)

	wg.Add(1)
	go qmpConnect(qmpChannel, instance, instanceDir, closedCh, connectedCh, &wg, false)
	wg.Wait()
	select {
	case <-closedCh:
	case <-time.After(time.Second):
		t.Fatalf("Timed out waiting for closedCh to close")
	}
}

func setupQmpSocket(t *testing.T, runTest func(net.Conn, *bufio.Scanner, chan interface{}, *testing.T) bool) {
	var wg sync.WaitGroup
	qmpChannel := make(chan interface{})
	closedCh := make(chan struct{})
	connectedCh := make(chan struct{})
	instance := "testInstance"
	instanceDir := path.Join("/tmp", instance)

	err := os.MkdirAll(instanceDir, 0755)
	if err != nil {
		t.Fatalf("Unable to create %s: %v", instanceDir, err)
	}
	defer func() {
		_ = os.RemoveAll(instanceDir)
	}()

	socketPath := path.Join(instanceDir, "socket")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Unable to open domain socket %s: %v", socketPath, err)
	}
	defer ln.Close()
	wg.Add(1)
	go qmpConnect(qmpChannel, instance, instanceDir, closedCh, connectedCh, &wg, false)
	fd, err := ln.Accept()
	if err != nil {
		t.Fatalf("Unable to accept client %v", err)
	}

	fd.SetDeadline(time.Now().Add(5. * time.Second))
	const qmpHello = `{ "QMP": { "version": { "qemu": { "micro": 0, "minor": 5, "major": 2}, "package": ""}, "capabilities": []}}`
	_, err = fmt.Fprintln(fd, qmpHello)
	if err != nil {
		fd.Close()
		t.Fatalf("Unable to write to qmpChannel %v", err)
	}

	sc := bufio.NewScanner(fd)
	if !sc.Scan() {
		fd.Close()
		t.Fatalf("query capabilities not received")
	}

	_, err = fmt.Fprintln(fd, `{ "return": {}}`)
	if err != nil {
		fd.Close()
		t.Fatalf("Unable to write to qmpChannel %v", err)
	}

	select {
	case <-connectedCh:
	case <-time.After(time.Second):
		fd.Close()
		t.Fatalf("Timed out waiting for connectedCh to close")
	}

	if runTest(fd, sc, qmpChannel, t) {
		defer fd.Close()
	}
	close(qmpChannel)
	wg.Wait()
	select {
	case <-closedCh:
	case <-time.After(time.Second):
		t.Fatalf("Timed out waiting for closedCh to close")
	}
}

func TestQmpConnect(t *testing.T) {
	setupQmpSocket(t, func(fd net.Conn, sc *bufio.Scanner, qmpChannel chan interface{}, t *testing.T) bool {
		return true
	})
}

func TestQmpShutdown(t *testing.T) {
	setupQmpSocket(t, func(fd net.Conn, sc *bufio.Scanner, qmpChannel chan interface{}, t *testing.T) bool {
		qmpChannel <- virtualizerStopCmd{}
		if !sc.Scan() {
			t.Fatalf("power down command expected")
		}
		_, err := fmt.Fprintln(fd, `{ "return": {}}
{"timestamp": {"seconds": 1487084520, "microseconds": 332329}, "event": "SHUTDOWN"}
		`)
		if err != nil {
			t.Fatalf("Unable to write to domain socket: %v", err)
		}

		return true
	})
}

func TestQmpLost(t *testing.T) {
	setupQmpSocket(t, func(fd net.Conn, sc *bufio.Scanner, qmpChannel chan interface{}, t *testing.T) bool {
		qmpChannel <- virtualizerStopCmd{}
		if !sc.Scan() {
			t.Fatalf("power down command expected")
		}
		fd.Close()

		return false
	})
}
