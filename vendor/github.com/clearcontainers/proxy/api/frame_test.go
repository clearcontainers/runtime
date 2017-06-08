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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrameTypeString(t *testing.T) {
	tests := []struct {
		t FrameType
		s string
	}{
		{TypeCommand, "command"},
		{TypeResponse, "response"},
		{TypeStream, "stream"},
		{TypeNotification, "notification"},
		{TypeMax, "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.s, test.t.String())
	}
}

func TestCommandString(t *testing.T) {
	tests := []struct {
		c Command
		s string
	}{
		{CmdRegisterVM, "RegisterVM"},
		{CmdUnregisterVM, "UnregisterVM"},
		{CmdAttachVM, "AttachVM"},
		{CmdHyper, "Hyper"},
		{CmdConnectShim, "ConnectShim"},
		{CmdDisconnectShim, "DisconnectShim"},
		{CmdSignal, "Signal"},
		{CmdMax, "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.s, test.c.String())
	}
}

func TestStreamString(t *testing.T) {
	tests := []struct {
		io Stream
		s  string
	}{
		{StreamStdin, "stdin"},
		{StreamStdout, "stdout"},
		{StreamStderr, "stderr"},
		{StreamLog, "log"},
		{StreamMax, "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.s, test.io.String())
	}
}

func TestNotificationString(t *testing.T) {
	tests := []struct {
		n Notification
		s string
	}{
		{NotificationProcessExited, "ProcessExited"},
		{NotificationMax, "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.s, test.n.String())
	}
}

type jsonPayload struct {
	Foo string `json:"foo"`
}

func TestNewFrameJson(t *testing.T) {
	payload := &jsonPayload{
		Foo: "bar",
	}

	frame, err := NewFrameJSON(TypeCommand, int(CmdSignal), payload)
	assert.Nil(t, err)

	decoded := jsonPayload{}
	err = json.Unmarshal(frame.Payload, &decoded)
	assert.Nil(t, err)
	assert.Equal(t, payload.Foo, decoded.Foo)
}

func TestNewFrameJsonNilPayload(t *testing.T) {
	frame, err := NewFrameJSON(TypeCommand, int(CmdSignal), nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, frame.Header.PayloadLength)

}
