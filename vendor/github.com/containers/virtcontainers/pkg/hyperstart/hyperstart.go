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

package hyperstart

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	hyper "github.com/hyperhq/runv/hyperstart/api/json"
)

// Control command IDs
// Need to be in sync with hyperstart/src/api.h
const (
	Version         = "version"
	StartPod        = "startpod"
	DestroyPod      = "destroypod"
	ExecCmd         = "execcmd"
	Ready           = "ready"
	Ack             = "ack"
	Error           = "error"
	WinSize         = "winsize"
	Ping            = "ping"
	FinishPod       = "finishpod"
	Next            = "next"
	WriteFile       = "writefile"
	ReadFile        = "readfile"
	NewContainer    = "newcontainer"
	KillContainer   = "killcontainer"
	RemoveContainer = "removecontainer"
	OnlineCPUMem    = "onlinecpumem"
	SetupInterface  = "setupinterface"
	SetupRoute      = "setuproute"
)

var codeList = map[string]uint32{
	Version:         hyper.INIT_VERSION,
	StartPod:        hyper.INIT_STARTPOD,
	DestroyPod:      hyper.INIT_DESTROYPOD,
	ExecCmd:         hyper.INIT_EXECCMD,
	Ready:           hyper.INIT_READY,
	Ack:             hyper.INIT_ACK,
	Error:           hyper.INIT_ERROR,
	WinSize:         hyper.INIT_WINSIZE,
	Ping:            hyper.INIT_PING,
	Next:            hyper.INIT_NEXT,
	WriteFile:       hyper.INIT_WRITEFILE,
	ReadFile:        hyper.INIT_READFILE,
	NewContainer:    hyper.INIT_NEWCONTAINER,
	KillContainer:   hyper.INIT_KILLCONTAINER,
	RemoveContainer: hyper.INIT_REMOVECONTAINER,
	OnlineCPUMem:    hyper.INIT_ONLINECPUMEM,
	SetupInterface:  hyper.INIT_SETUPINTERFACE,
	SetupRoute:      hyper.INIT_SETUPROUTE,
}

// Values related to the communication on control channel.
const (
	ctlHdrSize      = 8
	ctlHdrLenOffset = 4
)

// Values related to the communication on tty channel.
const (
	ttyHdrSize      = 12
	ttyHdrLenOffset = 8
)

type connState struct {
	sync.Mutex
	opened bool
}

func (c *connState) close() {
	c.Lock()
	defer c.Unlock()

	c.opened = false
}

func (c *connState) open() {
	c.Lock()
	defer c.Unlock()

	c.opened = true
}

func (c *connState) closed() bool {
	c.Lock()
	defer c.Unlock()

	return !c.opened
}

// Hyperstart is the base structure for hyperstart.
type Hyperstart struct {
	ctlSerial, ioSerial string
	sockType            string
	ctl, io             net.Conn
	ctlState, ioState   connState

	// ctl access is arbitrated by ctlMutex. We can only allow a single
	// "transaction" (write command + read answer) at a time
	ctlMutex sync.Mutex

	ctlMulticast *multicast
}

// NewHyperstart returns a new hyperstart structure.
func NewHyperstart(ctlSerial, ioSerial, sockType string) *Hyperstart {
	return &Hyperstart{
		ctlSerial: ctlSerial,
		ioSerial:  ioSerial,
		sockType:  sockType,
	}
}

// OpenSockets opens both CTL and IO sockets.
func (h *Hyperstart) OpenSockets() error {
	var err error

	h.ctl, err = net.Dial(h.sockType, h.ctlSerial)
	if err != nil {
		return err
	}
	h.ctlState.open()

	h.io, err = net.Dial(h.sockType, h.ioSerial)
	if err != nil {
		h.ctl.Close()
		return err
	}
	h.ioState.open()

	h.ctlMulticast = startCtlMonitor(h.ctl)

	return nil
}

// CloseSockets closes both CTL and IO sockets.
func (h *Hyperstart) CloseSockets() error {
	if !h.ctlState.closed() {
		err := h.ctl.Close()
		if err != nil {
			return err
		}

		h.ctlState.close()
	}

	if !h.ioState.closed() {
		err := h.io.Close()
		if err != nil {
			return err
		}

		h.ioState.close()
	}

	h.ctlMulticast = nil

	return nil
}

// SetDeadline sets a timeout for CTL connection.
func (h *Hyperstart) SetDeadline(t time.Time) error {
	err := h.ctl.SetDeadline(t)
	if err != nil {
		return err
	}

	return nil
}

// IsStarted returns about connection status.
func (h *Hyperstart) IsStarted() bool {
	ret := false
	timeoutDuration := 1 * time.Second

	if h.ctlState.closed() {
		return ret
	}

	h.SetDeadline(time.Now().Add(timeoutDuration))

	_, err := h.SendCtlMessage(Ping, nil)
	if err == nil {
		ret = true
	}

	h.SetDeadline(time.Time{})

	if ret == false {
		h.CloseSockets()
	}

	return ret
}

// FormatMessage formats hyperstart messages.
func FormatMessage(payload interface{}) ([]byte, error) {
	var payloadSlice []byte
	var err error

	if payload != nil {
		switch p := payload.(type) {
		case string:
			payloadSlice = []byte(p)
		default:
			payloadSlice, err = json.Marshal(p)
			if err != nil {
				return nil, err
			}
		}
	}

	return payloadSlice, nil
}

// ReadCtlMessage reads an hyperstart message from conn and returns a decoded message.
//
// This is a low level function, for a full and safe transaction on the
// hyperstart control serial link, use SendCtlMessage.
func ReadCtlMessage(conn net.Conn) (*hyper.DecodedMessage, error) {
	needRead := ctlHdrSize
	length := 0
	read := 0
	buf := make([]byte, 512)
	res := []byte{}
	for read < needRead {
		want := needRead - read
		if want > 512 {
			want = 512
		}
		nr, err := conn.Read(buf[:want])
		if err != nil {
			return nil, err
		}

		res = append(res, buf[:nr]...)
		read = read + nr

		if length == 0 && read >= ctlHdrSize {
			length = int(binary.BigEndian.Uint32(res[ctlHdrLenOffset:ctlHdrSize]))
			if length > ctlHdrSize {
				needRead = length
			}
		}
	}

	return &hyper.DecodedMessage{
		Code:    binary.BigEndian.Uint32(res[:ctlHdrLenOffset]),
		Message: res[ctlHdrSize:],
	}, nil
}

// WriteCtlMessage writes an hyperstart message to conn.
//
// This is a low level function, for a full and safe transaction on the
// hyperstart control serial link, use SendCtlMessage.
func (h *Hyperstart) WriteCtlMessage(conn net.Conn, m *hyper.DecodedMessage) error {
	length := len(m.Message) + ctlHdrSize
	// XXX: Support sending messages by chunks to support messages over
	// 10240 bytes. That limit is from hyperstart src/init.c,
	// hyper_channel_ops, rbuf_size.
	if length > 10240 {
		return fmt.Errorf("message too long %d", length)
	}
	msg := make([]byte, length)
	binary.BigEndian.PutUint32(msg[:], uint32(m.Code))
	binary.BigEndian.PutUint32(msg[ctlHdrLenOffset:], uint32(length))
	copy(msg[ctlHdrSize:], m.Message)

	_, err := conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}

// ReadIoMessageWithConn returns data coming from the specified IO channel.
func ReadIoMessageWithConn(conn net.Conn) (*hyper.TtyMessage, error) {
	needRead := ttyHdrSize
	length := 0
	read := 0
	buf := make([]byte, 512)
	res := []byte{}
	for read < needRead {
		want := needRead - read
		if want > 512 {
			want = 512
		}
		nr, err := conn.Read(buf[:want])
		if err != nil {
			return nil, err
		}

		res = append(res, buf[:nr]...)
		read = read + nr

		if length == 0 && read >= ttyHdrSize {
			length = int(binary.BigEndian.Uint32(res[ttyHdrLenOffset:ttyHdrSize]))
			if length > ttyHdrSize {
				needRead = length
			}
		}
	}

	return &hyper.TtyMessage{
		Session: binary.BigEndian.Uint64(res[:ttyHdrLenOffset]),
		Message: res[ttyHdrSize:],
	}, nil
}

// ReadIoMessage returns data coming from the IO channel.
func (h *Hyperstart) ReadIoMessage() (*hyper.TtyMessage, error) {
	return ReadIoMessageWithConn(h.io)
}

// SendIoMessageWithConn sends data to the specified IO channel.
func SendIoMessageWithConn(conn net.Conn, ttyMsg *hyper.TtyMessage) error {
	length := len(ttyMsg.Message) + ttyHdrSize
	// XXX: Support sending messages by chunks to support messages over
	// 10240 bytes. That limit is from hyperstart src/init.c,
	// hyper_channel_ops, rbuf_size.
	if length > 10240 {
		return fmt.Errorf("message too long %d", length)
	}
	msg := make([]byte, length)
	binary.BigEndian.PutUint64(msg[:], ttyMsg.Session)
	binary.BigEndian.PutUint32(msg[ttyHdrLenOffset:], uint32(length))
	copy(msg[ttyHdrSize:], ttyMsg.Message)

	n, err := conn.Write(msg)
	if err != nil {
		return err
	}

	if n != length {
		return fmt.Errorf("%d bytes written out of %d expected", n, length)
	}

	return nil
}

// SendIoMessage sends data to the IO channel.
func (h *Hyperstart) SendIoMessage(ttyMsg *hyper.TtyMessage) error {
	return SendIoMessageWithConn(h.io, ttyMsg)
}

func codeFromCmd(cmd string) (uint32, error) {
	_, ok := codeList[cmd]
	if ok == false {
		return math.MaxUint32, fmt.Errorf("unknown command '%s'", cmd)
	}

	return codeList[cmd], nil
}

func (h *Hyperstart) checkReturnedCode(recvCode, expectedCode uint32) error {
	if recvCode != expectedCode {
		if recvCode == hyper.INIT_ERROR {
			return fmt.Errorf("ERROR received from Hyperstart")
		}

		return fmt.Errorf("CMD ID received %d not matching expected %d", recvCode, expectedCode)
	}

	return nil
}

// WaitForReady waits for a READY message on CTL channel.
func (h *Hyperstart) WaitForReady() error {
	if h.ctlMulticast == nil {
		return fmt.Errorf("No multicast available for CTL channel")
	}

	channel, err := h.ctlMulticast.listen("", "", replyType)
	if err != nil {
		return err
	}

	msg := <-channel

	err = h.checkReturnedCode(msg.Code, hyper.INIT_READY)
	if err != nil {
		return err
	}

	return nil
}

// WaitForPAE waits for a PROCESSASYNCEVENT message on CTL channel.
func (h *Hyperstart) WaitForPAE(containerID, processID string) (*hyper.ProcessAsyncEvent, error) {
	if h.ctlMulticast == nil {
		return nil, fmt.Errorf("No multicast available for CTL channel")
	}

	channel, err := h.ctlMulticast.listen(containerID, processID, eventType)
	if err != nil {
		return nil, err
	}

	msg := <-channel

	var paeData hyper.ProcessAsyncEvent
	err = json.Unmarshal(msg.Message, paeData)
	if err != nil {
		return nil, err
	}

	return &paeData, nil
}

// SendCtlMessage sends a message to the CTL channel.
//
// This function does a full transaction over the CTL channel: it will rely on the
// multicaster to register a listener reading over the CTL channel. Then it writes
// a command and waits for the multicaster to send hyperstart's answer back before
// it can return.
// Several concurrent calls to SendCtlMessage are allowed, the function ensuring
// proper serialization of the communication by making the listener registration
// and the command writing an atomic operation protected by a mutex.
// Waiting for the reply from multicaster doesn't need to be protected by this mutex.
func (h *Hyperstart) SendCtlMessage(cmd string, data []byte) (*hyper.DecodedMessage, error) {
	if h.ctlMulticast == nil {
		return nil, fmt.Errorf("No multicast available for CTL channel")
	}

	h.ctlMutex.Lock()

	channel, err := h.ctlMulticast.listen("", "", replyType)
	if err != nil {
		h.ctlMutex.Unlock()
		return nil, err
	}

	code, err := codeFromCmd(cmd)
	if err != nil {
		h.ctlMutex.Unlock()
		return nil, err
	}

	msgSend := &hyper.DecodedMessage{
		Code:    code,
		Message: data,
	}
	err = h.WriteCtlMessage(h.ctl, msgSend)
	if err != nil {
		h.ctlMutex.Unlock()
		return nil, err
	}

	h.ctlMutex.Unlock()

	msgRecv := <-channel

	err = h.checkReturnedCode(msgRecv.Code, hyper.INIT_ACK)
	if err != nil {
		return nil, err
	}

	return msgRecv, nil
}
