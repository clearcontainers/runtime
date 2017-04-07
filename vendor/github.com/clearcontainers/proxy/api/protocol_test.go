// Copyright (c) 2017 Intel Corporation
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

package api

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// hl is in bytes
func makeFrame(v, hl int, t FrameType, op, pl int) []byte {
	buf := make([]byte, hl+pl)
	binary.BigEndian.PutUint16(buf[0:2], uint16(v))
	buf[2] = byte(hl / 4)
	buf[6] = byte(t)
	buf[7] = byte(op)
	binary.BigEndian.PutUint32(buf[8:8+4], uint32(pl))

	return buf
}

func setFlags(data []byte, flags uint8) {
	b := data[6]
	b |= flags
	data[6] = b
}

func getFlags(data []byte) uint8 {
	return data[6] & 0xf0
}

func makeStreamFrame(v, hl int, op Stream, pl int) []byte {
	return makeFrame(v, hl, TypeStream, int(op), pl)
}

func makeFrameReader(t FrameType, op, pl int) io.Reader {
	frame := makeFrame(Version, minHeaderLength, t, op, pl)
	return bytes.NewReader(frame)
}

func TestReadFrame(t *testing.T) {
	reader := makeFrameReader(TypeStream, int(StreamStderr), 1024)
	frame, err := ReadFrame(reader)
	assert.Nil(t, err)

	header := &frame.Header
	assert.Equal(t, Version, header.Version)
	assert.Equal(t, minHeaderLength, header.HeaderLength)
	assert.Equal(t, TypeStream, FrameType(header.Type))
	assert.Equal(t, StreamStderr, Stream(header.Opcode))
	assert.Equal(t, 1024, header.PayloadLength)
	assert.Equal(t, 1024, len(frame.Payload))
}

func TestReadFrameFlags(t *testing.T) {
	buf := makeFrame(Version, minHeaderLength, TypeResponse, int(CmdSignal), 16)
	setFlags(buf, flagInError)
	r := bytes.NewReader(buf)
	frame, err := ReadFrame(r)
	assert.Nil(t, err)
	assert.Equal(t, true, frame.Header.InError)
}

func TestReadFrameErrorPaths(t *testing.T) {
	// EOF.
	frame, err := ReadFrame(bytes.NewReader(nil))
	assert.Nil(t, frame)
	assert.NotNil(t, err)

	// Truncated input, header too short.
	buf := makeStreamFrame(Version, minHeaderLength, StreamStderr, 1024)
	frame, err = ReadFrame(bytes.NewReader(buf[0:10]))
	assert.Nil(t, frame)
	assert.NotNil(t, err)

	// Truncated input, payload too short.
	buf = makeStreamFrame(Version, minHeaderLength, StreamStderr, 1024)
	frame, err = ReadFrame(bytes.NewReader(buf[0:512]))
	assert.Nil(t, frame)
	assert.NotNil(t, err)

	// Bad version
	buf = makeStreamFrame(0x8fff, minHeaderLength, StreamStderr, 1024)
	frame, err = ReadFrame(bytes.NewReader(buf))
	assert.Nil(t, frame)
	assert.NotNil(t, err)

	buf = makeStreamFrame(0, minHeaderLength, StreamStderr, 1024)
	frame, err = ReadFrame(bytes.NewReader(buf))
	assert.Nil(t, frame)
	assert.NotNil(t, err)

	// Bad type
	buf = makeFrame(Version, minHeaderLength, TypeMax, 0, 1024)
	frame, err = ReadFrame(bytes.NewReader(buf))
	assert.Nil(t, frame)
	assert.NotNil(t, err)
}

// TestLargerHeader makes sure we can read a larger header than minHeaderLength
// without any issue.
func TestReadFrameLargerHeader(t *testing.T) {
	buf := makeFrame(Version, minHeaderLength+12, TypeStream,
		int(StreamStderr), 1024)
	frame, err := ReadFrame(bytes.NewReader(buf))
	assert.Nil(t, err)

	header := &frame.Header
	assert.Equal(t, Version, header.Version)
	assert.Equal(t, minHeaderLength+12, header.HeaderLength)
	assert.Equal(t, TypeStream, FrameType(header.Type))
	assert.Equal(t, StreamStderr, Stream(header.Opcode))
	assert.Equal(t, 1024, header.PayloadLength)
	assert.Equal(t, 1024, len(frame.Payload))
}

func newBuffer(payloadLength int) *bytes.Buffer {
	buf := make([]byte, 0, minHeaderLength+payloadLength)
	return bytes.NewBuffer(buf)
}

func newTestFrame(t FrameType, op, pl int) *Frame {
	payload := make([]byte, pl)
	for i := range payload {
		payload[i] = 0xaa
	}

	return NewFrame(t, op, payload)
}

func newStreamFrame(op Stream, pl int) *Frame {
	return newTestFrame(TypeStream, int(op), pl)
}

func newCommandFrame(op Stream, pl int) *Frame {
	return newTestFrame(TypeStream, int(op), pl)
}

func newResponseFrame(op Stream, pl int) *Frame {
	return newTestFrame(TypeStream, int(op), pl)
}

func TestWriteFrame(t *testing.T) {
	w := newBuffer(1024)
	frame := newStreamFrame(StreamStderr, 1024)

	err := WriteFrame(w, frame)
	assert.Nil(t, err)
	buf := w.Bytes()

	version := int(binary.BigEndian.Uint16(buf[0:2]))
	assert.Equal(t, Version, version)
	assert.Equal(t, uint8(minHeaderLength/4), buf[2])
	assert.Equal(t, byte(TypeStream), buf[6]&0xf)
	assert.Equal(t, byte(StreamStderr), buf[7])
	pl := int(binary.BigEndian.Uint32(buf[8 : 8+4]))
	assert.Equal(t, 1024, pl)

	for i := range buf[minHeaderLength : minHeaderLength+1024] {
		assert.Equal(t, uint8(0xaa), buf[minHeaderLength+i])
	}
}

func TestWriteFrameFlags(t *testing.T) {
	frame := newStreamFrame(StreamStderr, 1024)
	frame.Header.InError = true

	w := newBuffer(1024)
	err := WriteFrame(w, frame)
	assert.Nil(t, err)
	buf := w.Bytes()
	assert.Equal(t, uint8(flagInError), getFlags(buf))
}

func TestWriteFrameErrorPaths(t *testing.T) {
	// Header.PayloadLength to large compared to len(Payload.)
	w := newBuffer(1024)
	frame := newStreamFrame(StreamStderr, 1024)
	frame.Header.PayloadLength = 1025

	err := WriteFrame(w, frame)
	assert.NotNil(t, err)
	assert.Equal(t, 0, w.Len())
}

func TestWriteCommand(t *testing.T) {
	w := newBuffer(1024)
	err := WriteCommand(w, CmdSignal, nil)
	assert.Nil(t, err)

	buf := w.Bytes()
	assert.Equal(t, byte(TypeCommand), buf[6]&0xf)
}

func TestWriteResponse(t *testing.T) {
	w := newBuffer(1024)
	err := WriteResponse(w, CmdSignal, false, nil)
	assert.Nil(t, err)

	buf := w.Bytes()
	assert.Equal(t, byte(TypeResponse), buf[6]&0xf)
}

func TestWriteStream(t *testing.T) {
	w := newBuffer(1024)
	err := WriteStream(w, StreamStderr, nil)
	assert.Nil(t, err)

	buf := w.Bytes()
	assert.Equal(t, byte(TypeStream), buf[6]&0xf)
}

func TestWriteNotification(t *testing.T) {
	w := newBuffer(1024)
	err := WriteNotification(w, NotificationProcessExited, nil)
	assert.Nil(t, err)

	buf := w.Bytes()
	assert.Equal(t, byte(TypeNotification), buf[6]&0xf)
}
