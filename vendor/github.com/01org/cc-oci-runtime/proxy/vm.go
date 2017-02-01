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
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/containers/virtcontainers/hyperstart"
	"github.com/golang/glog"
)

// Represents a single qemu/hyperstart instance on the system
type vm struct {
	sync.Mutex

	containerID string

	hyperHandler *hyperstart.Hyperstart

	// Socket to the VM console
	console struct {
		socketPath string
		conn       net.Conn
	}

	// Used to allocate globally unique IO sequence numbers
	nextIoBase uint64

	// ios are hashed by their sequence numbers. If 2 sequence numbers are
	// allocated for one process (stdin/stdout and stderr) both sequence
	// numbers appear in this map.
	ioSessions map[uint64]*ioSession

	// Used to wait for all VM-global goroutines to finish on Close()
	wg sync.WaitGroup

	// Channel to signal qemu has terminated.
	vmLost chan interface{}
}

// A set of I/O streams between a client and a process running inside the VM
type ioSession struct {
	nStreams int
	ioBase   uint64

	// id  of the client owning that ioSession
	clientID uint64

	// socket connected to the fd sent over to the client
	client net.Conn

	// Used to wait for per-ioSession goroutines. Currently there's only
	// one such goroutine, the one reading stdin data from client socket.
	wg sync.WaitGroup
}

func newVM(id, ctlSerial, ioSerial string) *vm {
	h := hyperstart.NewHyperstart(ctlSerial, ioSerial, "unix")

	return &vm{
		containerID:  id,
		hyperHandler: h,
		nextIoBase:   1,
		ioSessions:   make(map[uint64]*ioSession),
		vmLost:       make(chan interface{}),
	}
}

// setConsole() will make the proxy output the console data on stderr
func (vm *vm) setConsole(path string) {
	vm.console.socketPath = path
}

func (vm *vm) shortName() string {
	length := 8
	if len(vm.containerID) < 8 {
		length = len(vm.containerID)
	}
	return vm.containerID[0:length]
}

func (vm *vm) info(lvl glog.Level, channel string, msg string) {
	if !glog.V(lvl) {
		return
	}
	glog.Infof("[vm %s %s] %s", vm.shortName(), channel, msg)
}

func (vm *vm) infof(lvl glog.Level, channel string, fmt string, a ...interface{}) {
	if !glog.V(lvl) {
		return
	}
	a = append(a, 0, 0)
	copy(a[2:], a[0:])
	a[0] = vm.shortName()
	a[1] = channel
	glog.Infof("[vm %s %s] "+fmt, a...)
}

func (vm *vm) dump(lvl glog.Level, data []byte) {
	if !glog.V(lvl) {
		return
	}
	glog.Infof("\n%s", hex.Dump(data))
}

func (vm *vm) findSession(seq uint64) *ioSession {
	vm.Lock()
	defer vm.Unlock()

	return vm.ioSessions[seq]
}

// This function runs in a goroutine, reading data from the io channel and
// dispatching it to the right client (the one with matching seq number)
// There's only one instance of this goroutine per-VM
func (vm *vm) ioHyperToClients() {
	for {
		msg, err := vm.hyperHandler.ReadIoMessage()
		if err != nil {
			// VM process is gone
			vm.signalVMLost()
			break
		}

		session := vm.findSession(msg.Session)
		if session == nil {
			fmt.Fprintf(os.Stderr,
				"couldn't find client with seq number %d\n", msg.Session)
			continue
		}

		vm.infof(1, "io", "<- writing to client #%d", session.clientID)
		vm.dump(2, msg.Message)

		err = hyperstart.SendIoMessageWithConn(session.client, msg)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"error writing I/O data to client: %v\n", err)
			break
		}
	}

	vm.wg.Done()
}

// Stream the VM console to stderr
func (vm *vm) consoleToLog() {
	reader := bufio.NewReader(vm.console.conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		vm.infof(3, "hyperstart", line)
	}

	vm.wg.Done()
}

func (vm *vm) Connect() error {
	if vm.console.socketPath != "" {
		var err error

		vm.console.conn, err = net.Dial("unix", vm.console.socketPath)
		if err != nil {
			return err
		}

		vm.wg.Add(1)
		go vm.consoleToLog()
	}

	if err := vm.hyperHandler.OpenSockets(); err != nil {
		return err
	}

	if err := vm.hyperHandler.WaitForReady(); err != nil {
		vm.hyperHandler.CloseSockets()
		return err
	}

	vm.wg.Add(1)
	go vm.ioHyperToClients()

	return nil
}

func (vm *vm) SendMessage(cmd string, data []byte) error {
	_, err := vm.hyperHandler.SendCtlMessage(cmd, data)
	return err
}

// This function runs in a goroutine, reading data from the client socket and
// writing data to the hyperstart I/O chanel.
// There's one instance of this goroutine per client having done an allocateIO.
func (vm *vm) ioClientToHyper(session *ioSession) {
	for {
		msg, err := hyperstart.ReadIoMessageWithConn(session.client)
		if err != nil {
			// client process is gone
			break
		}

		if msg.Session != session.ioBase {
			fmt.Fprintf(os.Stderr, "stdin seq %d not matching ioBase %d\n", msg.Session, session.ioBase)
			session.client.Close()
			break
		}

		vm.infof(1, "io", "-> writing to hyper from #%d", session.clientID)
		vm.dump(2, msg.Message)

		err = vm.hyperHandler.SendIoMessage(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"error writing I/O data to hyperstart: %v\n", err)
			break
		}
	}

	session.wg.Done()
}

func (vm *vm) AllocateIo(n int, clientID uint64, c net.Conn) uint64 {
	// Allocate ioBase
	vm.Lock()
	ioBase := vm.nextIoBase
	vm.nextIoBase += uint64(n)

	session := &ioSession{
		nStreams: n,
		ioBase:   ioBase,
		clientID: clientID,
		client:   c,
	}

	for i := 0; i < n; i++ {
		vm.ioSessions[ioBase+uint64(i)] = session
	}
	vm.Unlock()

	// Starts stdin forwarding between client and hyper
	session.wg.Add(1)
	go vm.ioClientToHyper(session)

	return ioBase
}

func (session *ioSession) Close() {
	session.client.Close()
	session.wg.Wait()
}

func (vm *vm) CloseIo(seq uint64) {
	vm.Lock()
	session := vm.ioSessions[seq]
	if session == nil {
		vm.Unlock()
		return
	}
	for i := 0; i < session.nStreams; i++ {
		delete(vm.ioSessions, seq+uint64(i))
	}
	vm.Unlock()

	session.Close()
}

func (vm *vm) Close() {
	vm.hyperHandler.CloseSockets()
	if vm.console.conn != nil {
		vm.console.conn.Close()
	}

	// Wait for per-client goroutines
	vm.Lock()
	for seq, session := range vm.ioSessions {
		delete(vm.ioSessions, seq)
		if seq != session.ioBase {
			continue
		}

		session.Close()
	}
	vm.Unlock()

	// Wait for VM global goroutines
	vm.wg.Wait()
}

// OnVmLost returns a channel can be waited on to signal the end of the qemu
// process.
func (vm *vm) OnVMLost() <-chan interface{} {
	return vm.vmLost
}

func (vm *vm) signalVMLost() {
	close(vm.vmLost)
}
