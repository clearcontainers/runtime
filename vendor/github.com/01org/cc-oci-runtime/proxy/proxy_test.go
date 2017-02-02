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
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/01org/cc-oci-runtime/proxy/api"
	"github.com/containers/virtcontainers/hyperstart/mock"

	hyper "github.com/hyperhq/runv/hyperstart/api/json"
	"github.com/stretchr/testify/assert"
)

type testRig struct {
	t  *testing.T
	wg sync.WaitGroup

	// hyperstart mocking
	Hyperstart      *mock.Hyperstart
	ctlPath, ioPath string

	// Control if we start the proxy in the test process or as a separate
	// process
	proxyFork bool

	// proxy, in process
	proxy     *proxy
	protocol  *protocol
	proxyConn net.Conn // socket used by proxy to communicate with Client

	// proxy, forked
	proxySocketPath string
	proxyCommand    *exec.Cmd

	// client
	Client *api.Client

	// fd leak detection
	detector          *FdLeakDetector
	startFds, stopFds *FdSnapshot
}

func newTestRig(t *testing.T, proto *protocol) *testRig {
	return &testRig{
		t:        t,
		protocol: proto,
		detector: NewFdLeadDetector(),
	}
}

func (rig *testRig) SetFork(fork bool) {
	rig.proxyFork = fork
}

func (rig *testRig) Start() {
	var err error

	rig.startFds, err = rig.detector.Snapshot()
	assert.Nil(rig.t, err)

	initLogging()
	flag.Parse()

	// Start hyperstart go routine
	rig.Hyperstart = mock.NewHyperstart(rig.t)
	rig.Hyperstart.Start()

	// Explicitly send READY message from hyperstart mock
	rig.wg.Add(1)
	go func() {
		rig.Hyperstart.SendMessage(int(hyper.INIT_READY), []byte{})
		rig.wg.Done()
	}()

	// we can either "start" the proxy in process or spawn a proxy process.
	// Spawning the process (through TestLaunchProxy).
	// Passing a file descriptor through connected AF_UNIX sockets in the
	// same thread has a slight behaviour difference which breaks the
	// barrier between two reads(), so we spawn a process in that case.
	var clientConn net.Conn

	if rig.proxyFork {
		rig.proxySocketPath = mock.GetTmpPath("test-proxy.%s.sock")
		rig.proxyCommand = proxyCommand(rig.proxySocketPath)
		err = rig.proxyCommand.Start()
		assert.Nil(rig.t, err)
		//output, err := rig.proxyCommand.CombinedOutput()
		//fmt.Fprintln(os.Stderr, hex.Dump(output))
		for i := 0; i < 2000; i++ {
			// XXX: We might want a mode where the proxy forks as
			// soon as it listens on its socket so we have a way to
			// known when we can connect to the proxy socket.
			time.Sleep(1 * time.Millisecond)
			clientConn, err = net.Dial("unix", rig.proxySocketPath)
			if err == nil {
				break
			}
		}
		assert.NotNil(rig.t, clientConn)
		rig.wg.Add(1)
		go func() {
			rig.proxyCommand.Wait()
			rig.wg.Done()
		}()
	} else {
		// client <-> proxy connection
		clientConn, rig.proxyConn, err = Socketpair()
		assert.Nil(rig.t, err)
		// Start proxy main go routine
		rig.proxy = newProxy()
		rig.wg.Add(1)
		go func() {
			rig.proxy.serveNewClient(rig.protocol, rig.proxyConn)
			rig.wg.Done()
		}()
	}

	// Client object that can be used to issue proxy commands
	rig.Client = api.NewClient(clientConn.(*net.UnixConn))
}

// A fake test we use to lauch a full proxy process
func proxyCommand(socketPath string) *exec.Cmd {
	cs := []string{"-test.run=TestLaunchProxy"}
	cmd := exec.Command(os.Args[0], cs...)

	socketEnv := fmt.Sprintf("CC_TEST_SOCKET_PATH=%s", socketPath)
	cmd.Env = []string{"CC_TEST_PROXY_PROCESS=1", socketEnv}

	return cmd
}

func TestLaunchProxy(t *testing.T) {
	if os.Getenv("CC_TEST_PROXY_PROCESS") != "1" {
		return
	}

	// used in proxy.go for the non socket-activated case
	DefaultSocketPath = os.Getenv("CC_TEST_SOCKET_PATH")

	proxyMain()
}

func (rig *testRig) Stop() {
	var err error

	rig.Client.Close()

	if rig.proxyCommand != nil {
		cmd := rig.proxyCommand
		if cmd.Process != nil {
			syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)
		}
	}
	if rig.proxyConn != nil {
		rig.proxyConn.Close()
	}
	if rig.proxySocketPath != "" {
		//os.Remove(rig.proxySocketPath)

	}

	rig.Hyperstart.Stop()

	rig.wg.Wait()

	if rig.proxy != nil {
		rig.proxy.wg.Wait()
	}

	// We shouldn't have leaked a fd between the beginning of Start() and
	// the end of Stop().
	rig.stopFds, err = rig.detector.Snapshot()
	assert.Nil(rig.t, err)

	assert.True(rig.t,
		rig.detector.Compare(os.Stdout, rig.startFds, rig.stopFds))
}

const testContainerID = "0987654321"

func TestHello(t *testing.T) {
	proto := newProtocol()
	proto.Handle("hello", helloHandler)

	rig := newTestRig(t, proto)
	rig.Start()

	// Register new VM
	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()
	ret, err := rig.Client.Hello(testContainerID, ctlSocketPath, ioSocketPath, nil)
	assert.Nil(t, err)
	assert.NotNil(t, ret)

	// Check that Hello returns the protocol version
	assert.Equal(t, api.Version, ret.Version)

	// A new Hello message with the same containerID should error out
	_, err = rig.Client.Hello(testContainerID, "fooCtl", "fooIo", nil)
	assert.NotNil(t, err)

	// Hello should register a new vm object
	proxy := rig.proxy
	proxy.Lock()
	vm := proxy.vms[testContainerID]
	proxy.Unlock()

	assert.NotNil(t, vm)
	assert.Equal(t, testContainerID, vm.containerID)

	// This test shouldn't send anything to hyperstart
	msgs := rig.Hyperstart.GetLastMessages()
	assert.Equal(t, 0, len(msgs))

	rig.Stop()
}

func TestBye(t *testing.T) {
	proto := newProtocol()
	proto.Handle("hello", helloHandler)
	proto.Handle("bye", byeHandler)

	rig := newTestRig(t, proto)
	rig.Start()

	// Register new VM
	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()
	_, err := rig.Client.Hello(testContainerID, ctlSocketPath, ioSocketPath, nil)
	assert.Nil(t, err)

	// Bye with a bad containerID
	err = rig.Client.Bye("foo")
	assert.NotNil(t, err)

	// Bye!
	err = rig.Client.Bye(testContainerID)
	assert.Nil(t, err)

	// A second Bye (client not attached anymore) should return an error
	err = rig.Client.Bye(testContainerID)
	assert.NotNil(t, err)

	// Bye should unregister the vm object
	proxy := rig.proxy
	proxy.Lock()
	vm := proxy.vms[testContainerID]
	proxy.Unlock()
	assert.Nil(t, vm)

	// This test shouldn't send anything to hyperstart
	msgs := rig.Hyperstart.GetLastMessages()
	assert.Equal(t, 0, len(msgs))

	rig.Stop()
}

func TestAttach(t *testing.T) {
	proto := newProtocol()
	proto.Handle("hello", helloHandler)
	proto.Handle("attach", attachHandler)
	proto.Handle("bye", byeHandler)

	rig := newTestRig(t, proto)
	rig.Start()

	// Register new VM
	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()
	_, err := rig.Client.Hello(testContainerID, ctlSocketPath, ioSocketPath, nil)
	assert.Nil(t, err)

	// Attaching to an unknown VM should return an error
	_, err = rig.Client.Attach("foo", nil)
	assert.NotNil(t, err)

	// Attaching to an existing VM should work. To test we are effectively
	// attached, we issue a bye that would error out if not attached.
	ret, err := rig.Client.Attach(testContainerID, nil)
	assert.Nil(t, err)

	// Check that Attach returns the protocol version
	assert.Equal(t, api.Version, ret.Version)

	err = rig.Client.Bye(testContainerID)
	assert.Nil(t, err)

	// This test shouldn't send anything with hyperstart
	msgs := rig.Hyperstart.GetLastMessages()
	assert.Equal(t, 0, len(msgs))

	rig.Stop()
}

func TestHyperPing(t *testing.T) {
	proto := newProtocol()
	proto.Handle("hello", helloHandler)
	proto.Handle("hyper", hyperHandler)

	rig := newTestRig(t, proto)
	rig.Start()

	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()
	_, err := rig.Client.Hello(testContainerID, ctlSocketPath, ioSocketPath, nil)
	assert.Nil(t, err)

	// Send ping and verify we have indeed received the message on the
	// hyperstart side. Ping is somewhat interesting because it's a case of
	// an hyper message without data.
	err = rig.Client.Hyper("ping", nil)
	assert.Nil(t, err)

	msgs := rig.Hyperstart.GetLastMessages()
	assert.Equal(t, 1, len(msgs))

	msg := msgs[0]
	assert.Equal(t, hyper.INIT_PING, int(msg.Code))
	assert.Equal(t, 0, len(msg.Message))

	rig.Stop()
}

func TestHyperStartpod(t *testing.T) {
	proto := newProtocol()
	proto.Handle("hello", helloHandler)
	proto.Handle("hyper", hyperHandler)

	rig := newTestRig(t, proto)
	rig.Start()

	// Register new VM
	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()
	_, err := rig.Client.Hello(testContainerID, ctlSocketPath, ioSocketPath, nil)
	assert.Nil(t, err)

	// Send startopd and verify we have indeed received the message on the
	// hyperstart side. startpod is interesting because it's a case of an
	// hyper message with JSON data.
	startpod := hyper.Pod{
		Hostname: "testhostname",
		ShareDir: "rootfs",
	}
	err = rig.Client.Hyper("startpod", &startpod)
	assert.Nil(t, err)

	msgs := rig.Hyperstart.GetLastMessages()
	assert.Equal(t, 1, len(msgs))

	msg := msgs[0]
	assert.Equal(t, hyper.INIT_STARTPOD, int(msg.Code))
	received := hyper.Pod{}
	err = json.Unmarshal(msg.Message, &received)
	assert.Nil(t, err)
	assert.Equal(t, startpod.Hostname, received.Hostname)
	assert.Equal(t, startpod.ShareDir, received.ShareDir)

	rig.Stop()
}

// header for hyperstart's I/O channel packets is 12 bytes
const ioHeaderLength = 12

// write a chunk of data to an I/O fd
func writeIo(t *testing.T, writer io.Writer, seq uint64, data []byte) {
	length := ioHeaderLength + len(data)
	header := make([]byte, ioHeaderLength)

	binary.BigEndian.PutUint64(header[:], uint64(seq))
	binary.BigEndian.PutUint32(header[8:], uint32(length))
	n, err := writer.Write(header)
	assert.Nil(t, err)
	assert.Equal(t, ioHeaderLength, n)

	n, err = writer.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)
}

// read a chunk of data from an I/O fd
func readIo(t *testing.T, reader io.Reader) (seq uint64, data []byte) {
	buf := make([]byte, ioHeaderLength)
	n, err := reader.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, ioHeaderLength, n)

	seq = binary.BigEndian.Uint64(buf[:8])
	length := binary.BigEndian.Uint32(buf[8:12]) - ioHeaderLength
	if length == 0 {
		return
	}

	received := 0
	need := int(length)
	data = make([]byte, need)
	for received < need {
		n, err := reader.Read(data[received:need])
		assert.Nil(t, err)

		received += n
	}

	return
}

func TestAllocateIo(t *testing.T) {
	proto := newProtocol()
	proto.Handle("hello", helloHandler)
	proto.Handle("allocateIO", allocateIoHandler)

	rig := newTestRig(t, proto)
	//rig.SetFork(true)
	rig.Start()

	// Register new VM
	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()
	_, err := rig.Client.Hello(testContainerID, ctlSocketPath, ioSocketPath, nil)
	assert.Nil(t, err)

	// Allocate 2 seq numbers and verify we can use the fd passed from
	// allocate I/O to send and receive data.
	ioBase, ioFile, err := rig.Client.AllocateIo(2)
	assert.Nil(t, err)

	// we always start our allocations from 1
	assert.Equal(t, uint64(1), ioBase)

	// make hyperstart send something on stdout/stderr and verify we
	// receive it
	streams := []struct {
		seq  uint64
		data string
	}{
		{ioBase, "stdout\n"},
		{ioBase + 1, "stderr\n"},
	}
	buf := make([]byte, 32)
	for _, stream := range streams {
		rig.Hyperstart.SendIoString(stream.seq, stream.data)
		n, err := ioFile.Read(buf)
		assert.Nil(t, err)
		// 12 is the length of the header on the hyperstart I/O channel
		assert.True(t, n > 12)
		assert.Equal(t, len(stream.data)+12, n)
		assert.Equal(t, stream.data, string(buf[12:n]))
	}

	// same thing for stdin
	const stdinData = "stdin\n"
	writeIo(t, ioFile, ioBase, []byte(stdinData))

	buf = make([]byte, 32)
	n, seq := rig.Hyperstart.ReadIo(buf)
	assert.Equal(t, ioBase, seq)
	assert.True(t, n > 12)
	assert.Equal(t, len(stdinData)+12, n)
	assert.Equal(t, stdinData, string(buf[12:n]))

	// Simulate the process exiting on the hyperstart end and check we
	// receive the exit status
	rig.Hyperstart.CloseIo(ioBase)
	rig.Hyperstart.SendExitStatus(ioBase, 17)

	// close
	seq, data := readIo(t, ioFile)
	assert.Equal(t, ioBase, seq)
	assert.Equal(t, 0, len(data))

	// exit status
	seq, data = readIo(t, ioFile)
	assert.Equal(t, ioBase, seq)
	assert.Equal(t, 1, len(data))
	assert.Equal(t, uint8(17), data[0])

	ioFile.Close()

	rig.Stop()
}
