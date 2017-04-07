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

package testutil_test

import (
	"errors"
	"testing"

	"github.com/01org/ciao/ssntp"
	. "github.com/01org/ciao/testutil"
)

func TestNewSsntpTestClientonnectionArgs(t *testing.T) {
	_, err := NewSsntpTestClientConnection("AGENT Client", ssntp.UNKNOWN, AgentUUID)
	if err == nil {
		t.Fatalf("NewSsntpTestClientConnection incorrectly accepted unknown role")
	}

	_, err = NewSsntpTestClientConnection("AGENT Client", ssntp.AGENT, "")
	if err == nil {
		t.Fatalf("NewSsntpTestClientConnection incorrectly accepted empty uuid")
	}
}

func TestAgentStatusChan(t *testing.T) {
	agentCh := agent.AddStatusChan(ssntp.READY)

	var result Result
	result.Err = errors.New("foo")
	go agent.SendResultAndDelStatusChan(ssntp.READY, result)

	r, err := agent.GetStatusChanResult(agentCh, ssntp.READY)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestAgentStatusChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	agentCh := agent.AddStatusChan(ssntp.READY)

	// should time out
	_, err := agent.GetStatusChanResult(agentCh, ssntp.READY)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go agent.SendResultAndDelStatusChan(ssntp.READY, result)
	_, err = agent.GetStatusChanResult(agentCh, ssntp.READY)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentErrorChan(t *testing.T) {
	agentCh := agent.AddErrorChan(ssntp.StopFailure)

	var result Result
	result.Err = errors.New("foo")
	go agent.SendResultAndDelErrorChan(ssntp.StopFailure, result)

	r, err := agent.GetErrorChanResult(agentCh, ssntp.StopFailure)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestAgentErrorChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	agentCh := agent.AddErrorChan(ssntp.StopFailure)

	// should time out
	_, err := agent.GetErrorChanResult(agentCh, ssntp.StopFailure)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go agent.SendResultAndDelErrorChan(ssntp.StopFailure, result)
	_, err = agent.GetErrorChanResult(agentCh, ssntp.StopFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentEventChan(t *testing.T) {
	agentCh := agent.AddEventChan(ssntp.TraceReport)

	var result Result
	result.Err = errors.New("foo")
	go agent.SendResultAndDelEventChan(ssntp.TraceReport, result)

	r, err := agent.GetEventChanResult(agentCh, ssntp.TraceReport)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestAgentEventChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	agentCh := agent.AddEventChan(ssntp.TraceReport)

	// should time out
	_, err := agent.GetEventChanResult(agentCh, ssntp.TraceReport)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go agent.SendResultAndDelEventChan(ssntp.TraceReport, result)
	_, err = agent.GetEventChanResult(agentCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentCmdChan(t *testing.T) {
	agentCh := agent.AddCmdChan(ssntp.START)

	var result Result
	result.Err = errors.New("foo")
	go agent.SendResultAndDelCmdChan(ssntp.START, result)

	r, err := agent.GetCmdChanResult(agentCh, ssntp.START)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestAgentCmdChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	agentCh := agent.AddCmdChan(ssntp.START)

	// should time out
	_, err := agent.GetCmdChanResult(agentCh, ssntp.START)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go agent.SendResultAndDelCmdChan(ssntp.START, result)
	_, err = agent.GetCmdChanResult(agentCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentCloseChans(t *testing.T) {
	var result Result

	_ = agent.AddCmdChan(ssntp.START)
	go agent.SendResultAndDelCmdChan(ssntp.START, result)

	_ = agent.AddEventChan(ssntp.TraceReport)
	go agent.SendResultAndDelEventChan(ssntp.TraceReport, result)

	_ = agent.AddErrorChan(ssntp.StopFailure)
	go agent.SendResultAndDelErrorChan(ssntp.StopFailure, result)

	_ = agent.AddStatusChan(ssntp.READY)
	go agent.SendResultAndDelStatusChan(ssntp.READY, result)

	CloseClientChans(agent)
	OpenClientChans(agent)
}
