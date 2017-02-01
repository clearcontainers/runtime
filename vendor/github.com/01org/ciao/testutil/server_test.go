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

func TestServerStatusChan(t *testing.T) {
	serverCh := server.AddStatusChan(ssntp.READY)

	var result Result
	result.Err = errors.New("foo")
	go server.SendResultAndDelStatusChan(ssntp.READY, result)

	r, err := server.GetStatusChanResult(serverCh, ssntp.READY)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestServerStatusChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	serverCh := server.AddStatusChan(ssntp.READY)

	_, err := server.GetStatusChanResult(serverCh, ssntp.READY)
	if err == nil {
		t.Fatal(err)
	}
}

func TestServerErrorChan(t *testing.T) {
	serverCh := server.AddErrorChan(ssntp.StopFailure)

	var result Result
	result.Err = errors.New("foo")
	go server.SendResultAndDelErrorChan(ssntp.StopFailure, result)

	r, err := server.GetErrorChanResult(serverCh, ssntp.StopFailure)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestServerErrorChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	serverCh := server.AddErrorChan(ssntp.StopFailure)

	_, err := server.GetErrorChanResult(serverCh, ssntp.StopFailure)
	if err == nil {
		t.Fatal(err)
	}
}

func TestServerEventChan(t *testing.T) {
	serverCh := server.AddEventChan(ssntp.TraceReport)

	var result Result
	result.Err = errors.New("foo")
	go server.SendResultAndDelEventChan(ssntp.TraceReport, result)

	r, err := server.GetEventChanResult(serverCh, ssntp.TraceReport)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestServerEventChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	serverCh := server.AddEventChan(ssntp.TraceReport)

	_, err := server.GetEventChanResult(serverCh, ssntp.TraceReport)
	if err == nil {
		t.Fatal(err)
	}
}

func TestServerCmdChan(t *testing.T) {
	serverCh := server.AddCmdChan(ssntp.START)

	var result Result
	result.Err = errors.New("foo")
	go server.SendResultAndDelCmdChan(ssntp.START, result)

	r, err := server.GetCmdChanResult(serverCh, ssntp.START)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestServerCmdChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	serverCh := server.AddCmdChan(ssntp.START)

	_, err := server.GetCmdChanResult(serverCh, ssntp.START)
	if err == nil {
		t.Fatal(err)
	}
}

func TestServerCloseChans(t *testing.T) {
	var result Result

	_ = server.AddCmdChan(ssntp.START)
	go server.SendResultAndDelCmdChan(ssntp.START, result)

	_ = server.AddEventChan(ssntp.TraceReport)
	go server.SendResultAndDelEventChan(ssntp.TraceReport, result)

	_ = server.AddErrorChan(ssntp.StopFailure)
	go server.SendResultAndDelErrorChan(ssntp.StopFailure, result)

	_ = server.AddStatusChan(ssntp.READY)
	go server.SendResultAndDelStatusChan(ssntp.READY, result)

	CloseServerChans(server)
	OpenServerChans(server)
}
