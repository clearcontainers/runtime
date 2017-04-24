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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/clearcontainers/proxy/api"

	"github.com/containers/virtcontainers/pkg/hyperstart"
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

	// tokenToSession associate a token to the corresponding ioSession
	tokenToSession map[Token]*ioSession

	// nullSession is a special I/O session used for containers and execmd processes
	// when client of the proxy indicates they don't care about communicating with the
	// process inside the VM.
	nullSession ioSession

	// Used to wait for all VM-global goroutines to finish on Close()
	wg sync.WaitGroup

	// Channel to signal qemu has terminated.
	vmLost chan interface{}
}

// A set of I/O streams between a client and a process running inside the VM
type ioSession struct {
	// token is what identifies the I/O session to the external world
	token Token

	// Back pointer to the VM this session is attached to.
	vm *vm

	nStreams int
	ioBase   uint64
	// Have we received the EOF paquet from hyperstart for this session?
	terminated bool

	// id  of the client owning that ioSession (the shim process, usually).
	clientID uint64

	// socket connected to the fd sent over to the client
	client net.Conn

	// Channel to signal a shim has been associated with this session (hyper
	// commands newcontainer and execcmd will wait for the shim to be ready
	// before forwarding the command to hyperstart)
	shimConnected chan interface{}
}

const (
	nullSessionStdout = 1 + iota
	nullSessionStderr
	firstIoBase
)

func newVM(id, ctlSerial, ioSerial string) *vm {
	h := hyperstart.NewHyperstart(ctlSerial, ioSerial, "unix")

	vm := &vm{
		containerID:    id,
		hyperHandler:   h,
		nextIoBase:     firstIoBase,
		ioSessions:     make(map[uint64]*ioSession),
		tokenToSession: make(map[Token]*ioSession),
		vmLost:         make(chan interface{}),
	}

	vm.nullSession = ioSession{
		vm:            vm,
		nStreams:      2,
		ioBase:        nullSessionStdout,
		shimConnected: make(chan interface{}),
	}
	vm.ioSessions[nullSessionStdout] = &vm.nullSession
	vm.ioSessions[nullSessionStderr] = &vm.nullSession
	// Null sessions are always ready, there won't be a shim to wait for
	close(vm.nullSession.shimConnected)

	return vm
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

func (vm *vm) findSessionBySeq(seq uint64) *ioSession {
	vm.Lock()
	defer vm.Unlock()

	return vm.ioSessions[seq]
}

func (vm *vm) findSessionByToken(token Token) *ioSession {
	vm.Lock()
	defer vm.Unlock()

	return vm.tokenToSession[token]
}

func hyperstartTtyMessageToFrame(msg *hyperstart.TtyMessage, session *ioSession) *api.Frame {
	// Exit status
	if session.terminated && len(msg.Message) == 1 {
		return api.NewFrame(api.TypeNotification, int(api.NotificationProcessExited), msg.Message)
	}

	// Regular stdout/err data
	var stream api.Stream

	if msg.Session == session.ioBase {
		stream = api.StreamStdout
	} else {
		stream = api.StreamStderr
	}

	return api.NewFrame(api.TypeStream, int(stream), msg.Message)
}

// This function runs in a goroutine, reading data from the io channel and
// dispatching it to the right client (the one with matching seq number)
// There's only one instance of this goroutine per-VM
func (vm *vm) ioHyperToClients() {
	for {
		msg, err := vm.hyperHandler.ReadIoMessage()
		if err != nil {
			break
		}

		session := vm.findSessionBySeq(msg.Session)
		if session == nil {
			fmt.Fprintf(os.Stderr,
				"couldn't find client with seq number %d\n", msg.Session)
			continue
		}

		// The nullSession acts like /dev/null, discard data associated with it
		if session == &vm.nullSession {
			vm.info(1, "io", "data received for the null session, discarding")
			vm.dump(2, msg.Message)
			continue
		}

		// When the process corresponding to a session exits:
		//   1. hyperstart sends an EOF paquet, ie. data_length == 0
		//      session.terminated tracks that condition
		//   2. hyperstart sends the exit status paquet, ie. data_length == 1
		if len(msg.Message) == 0 {
			session.terminated = true
			continue
		}

		vm.infof(1, "io", "<- writing to client #%d", session.clientID)
		vm.dump(2, msg.Message)

		frame := hyperstartTtyMessageToFrame(msg, session)
		err = api.WriteFrame(session.client, frame)
		if err != nil {
			// When the shim is forcefully killed, it's possible we
			// still have data to write. Ignore errors for that case.
			vm.infof(1, "io", "error writing I/O data to client:", err)
			continue
		}
	}

	// Having an error on the IO channel read is interpreted as having lost
	// the VM.
	vm.signalVMLost()
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

type relocationHandler func(*vm, *api.Hyper, *ioSession) error

func relocateProcess(process *hyperstart.Process, session *ioSession) error {
	// Make sure clients don't prefill process.Stdio and proces.Stderr
	if process.Stdio != 0 {
		return fmt.Errorf("expected process.Stdio to be 0, got %d", process.Stdio)
	}
	if process.Stderr != 0 {
		return fmt.Errorf("expected process.Stderr to be 0, got %d", process.Stderr)
	}

	process.Stdio = session.ioBase

	// When relocating a process asking for a terminal, we need to make sure
	// Process.Stderr is 0. We only need the Stdio sequence number in that case and
	// hyperstart will be mad at us if we specify Stderr.
	if process.Terminal == false {
		process.Stderr = session.ioBase + 1
	}

	return nil
}

func execcmdHandler(vm *vm, hyper *api.Hyper, session *ioSession) error {
	cmdIn := hyperstart.ExecCommand{}
	if err := json.Unmarshal(hyper.Data, &cmdIn); err != nil {
		return err
	}

	if err := relocateProcess(&cmdIn.Process, session); err != nil {
		return err
	}

	newData, err := json.Marshal(&cmdIn)
	if err != nil {
		return err
	}

	hyper.Data = newData

	return nil
}

func newcontainerHandler(vm *vm, hyper *api.Hyper, session *ioSession) error {
	cmdIn := hyperstart.Container{}
	if err := json.Unmarshal(hyper.Data, &cmdIn); err != nil {
		return err
	}

	relocateProcess(cmdIn.Process, session)
	newData, err := json.Marshal(&cmdIn)
	if err != nil {
		return err
	}

	hyper.Data = newData

	return nil
}

// RelocateHyperCommand performs the sequence number relocation in the
// newcontainer and execmd hyper commands given the corresponding list of
// tokens. Starpod isn't handled as it's not currently use to start processes
// and indicated as deprecated in the hyperstart API.
func (vm *vm) relocateHyperCommand(hyper *api.Hyper) error {
	cmds := []struct {
		name    string
		handler relocationHandler
	}{
		{"newcontainer", newcontainerHandler},
		{"execcmd", execcmdHandler},
	}
	needsRelocation := false

	nTokens := len(hyper.Tokens)

	for _, cmd := range cmds {
		if hyper.HyperName == cmd.name {
			if nTokens > 1 {
				return fmt.Errorf("expected 0 or 1 token, got %d", nTokens)
			}

			var session *ioSession
			if nTokens == 0 {
				// When not given any token, we use the nullSession and will discard data
				// received from hyper
				session = &vm.nullSession
			} else {
				token := hyper.Tokens[0]
				session = vm.findSessionByToken(Token(token))
				if session == nil {
					return fmt.Errorf("unknown token %s", token)
				}
			}

			if err := cmd.handler(vm, hyper, session); err != nil {
				return err
			}

			// Wait for the corresponding shim to be registered with the proxy so we can
			// start forwarding data as soon as we receive it
			if err := session.WaitForShim(); err != nil {
				return err
			}

			needsRelocation = true
			break
		}
	}

	// If a hyper command doesn't need a token but one is given anyway, reject the
	// command.
	if !needsRelocation && nTokens > 0 {
		return fmt.Errorf("%s doesn't need tokens but %d token(s) were given",
			hyper.HyperName, nTokens)

	}

	return nil
}

func (vm *vm) SendMessage(hyper *api.Hyper) error {
	if err := vm.relocateHyperCommand(hyper); err != nil {
		return err
	}

	_, err := vm.hyperHandler.SendCtlMessage(hyper.HyperName, hyper.Data)
	return err
}

var waitForShimTimeout = 30 * time.Second

// WaitFormShim will wait until a shim claiming the ioSession has registered
// itself with the proxy. If the shim has already done so, WaitForSim will
// return immediately.
func (session *ioSession) WaitForShim() error {
	session.vm.infof(1, "session",
		"waiting for shim to register itself with token %s (timeout %s)",
		session.token, waitForShimTimeout)

	select {
	case <-session.shimConnected:
	case <-time.After(waitForShimTimeout):
		return fmt.Errorf("timeout waiting for shim with token %s", session.token)
	}
	return nil
}

// ForwardStdin forwards an api.Frame with stdin data to hyperstart
func (session *ioSession) ForwardStdin(frame *api.Frame) error {
	if frame.Header.Type != api.TypeStream {
		return fmt.Errorf("expected stream frame got %s", frame.Header.Type)
	}

	streamType := api.Stream(frame.Header.Opcode)
	if streamType != api.StreamStdin {
		return fmt.Errorf("expected stdin stream frame got %s", streamType)
	}

	vm := session.vm
	msg := &hyperstart.TtyMessage{
		Session: session.ioBase,
		Message: frame.Payload,
	}

	vm.infof(1, "io", "-> writing to hyper from #%d", session.clientID)
	vm.dump(2, msg.Message)

	return vm.hyperHandler.SendIoMessage(msg)
}

// windowSizeMessage07 is the hyperstart 0.7 winsize message payload for the
// winsize command. This payload has changed in 0.8 so we can't use the
// definition in the hyperstart package.
type windowSizeMessage07 struct {
	Seq    uint64 `json:"seq"`
	Row    uint16 `json:"row"`
	Column uint16 `json:"column"`
}

// SendTerminalSize sends a new terminal geometry to the process represented by
// session.
func (session *ioSession) SendTerminalSize(columns, rows int) error {
	msg := &windowSizeMessage07{
		Seq:    session.ioBase,
		Column: uint16(columns),
		Row:    uint16(rows),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = session.vm.hyperHandler.SendCtlMessage("winsize", data)
	return err
}

// SendSignal
func (session *ioSession) SendSignal(signal syscall.Signal) error {
	msg := &hyperstart.KillCommand{
		Container: session.vm.containerID,
		Signal:    signal,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = session.vm.hyperHandler.SendCtlMessage("killcontainer", data)
	return err
}

func (vm *vm) AllocateToken() (Token, error) {
	vm.Lock()
	defer vm.Unlock()

	// We always allocate 2 sequence numbers (1 for stdin/out + 1 for
	// stderr).
	nStreams := 2
	ioBase := vm.nextIoBase
	vm.nextIoBase += uint64(nStreams)

	token, err := GenerateToken(32)
	if err != nil {
		return nilToken, err
	}

	session := &ioSession{
		vm:            vm,
		token:         token,
		nStreams:      nStreams,
		ioBase:        ioBase,
		shimConnected: make(chan interface{}),
	}

	// This mapping is to get the session from the seq number in an
	// hyperstart I/O paquet.
	for i := 0; i < nStreams; i++ {
		vm.ioSessions[ioBase+uint64(i)] = session
	}

	// This mapping is to get the session from the I/O token
	vm.tokenToSession[token] = session

	return token, nil
}

// AssociateShim associates a shim given by the triplet (token, clientID,
// clientConn) to a vm (POD). After associating the shim, a hyper command can
// be issued to start the process inside the VM and data can flow between shim
// and containerized process through the shim.
func (vm *vm) AssociateShim(token Token, clientID uint64, clientConn net.Conn) (*ioSession, error) {
	vm.Lock()
	defer vm.Unlock()

	session := vm.tokenToSession[token]
	if session == nil {
		return nil, fmt.Errorf("vm: unknown token %s", token)
	}

	session.clientID = clientID
	session.client = clientConn

	// Signal a runtime waiting that the shim is connected
	close(session.shimConnected)

	return session, nil
}

func (vm *vm) freeTokenUnlocked(token Token) error {
	session := vm.tokenToSession[token]
	if session == nil {
		return fmt.Errorf("vm: unknown token %s", token)
	}

	delete(vm.tokenToSession, token)

	for i := 0; i < session.nStreams; i++ {
		delete(vm.ioSessions, session.ioBase+uint64(i))
	}

	session.Close()

	return nil
}

func (vm *vm) FreeToken(token Token) error {
	vm.Lock()
	defer vm.Unlock()

	return vm.freeTokenUnlocked(token)
}

func (session *ioSession) Close() {
	// We can have a session created, but no shim associated with just yet.
	// In that case, client is nil.
	if session.client != nil {
		session.client.Close()
	}
}

func (vm *vm) Close() {
	vm.hyperHandler.CloseSockets()
	if vm.console.conn != nil {
		vm.console.conn.Close()
	}

	// Garbage collect I/O sessions in case Close() was called without
	// properly cleaning up all sessions.
	vm.Lock()
	for token := range vm.tokenToSession {
		vm.freeTokenUnlocked(token)
		delete(vm.tokenToSession, token)
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
