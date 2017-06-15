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

func TestNewSsntpTestControllerConnectionArgs(t *testing.T) {
	_, err := NewSsntpTestControllerConnection("Controller Client", "")
	if err == nil {
		t.Fatalf("NewSsntpTestControllerConnection incorrectly accepted empty uuid")
	}
}

func TestControllerErrorChan(t *testing.T) {
	controllerCh := controller.AddErrorChan(ssntp.StopFailure)

	var result Result
	result.Err = errors.New("foo")
	go controller.SendResultAndDelErrorChan(ssntp.StopFailure, result)

	r, err := controller.GetErrorChanResult(controllerCh, ssntp.StopFailure)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestControllerErrorChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	controllerCh := controller.AddErrorChan(ssntp.StopFailure)

	// should time out
	_, err := controller.GetErrorChanResult(controllerCh, ssntp.StopFailure)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go controller.SendResultAndDelErrorChan(ssntp.StopFailure, result)
	_, err = controller.GetErrorChanResult(controllerCh, ssntp.StopFailure)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerEventChan(t *testing.T) {
	controllerCh := controller.AddEventChan(ssntp.TraceReport)

	var result Result
	result.Err = errors.New("foo")
	go controller.SendResultAndDelEventChan(ssntp.TraceReport, result)

	r, err := controller.GetEventChanResult(controllerCh, ssntp.TraceReport)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestControllerEventChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	controllerCh := controller.AddEventChan(ssntp.TraceReport)

	// should time out
	_, err := controller.GetEventChanResult(controllerCh, ssntp.TraceReport)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go controller.SendResultAndDelEventChan(ssntp.TraceReport, result)
	_, err = controller.GetEventChanResult(controllerCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerCmdChan(t *testing.T) {
	controllerCh := controller.AddCmdChan(ssntp.START)

	var result Result
	result.Err = errors.New("foo")
	go controller.SendResultAndDelCmdChan(ssntp.START, result)

	r, err := controller.GetCmdChanResult(controllerCh, ssntp.START)
	if err == nil {
		t.Fatal(err)
	}
	if r.Err != result.Err {
		t.Fatalf("channel returned wrong result: expected \"%s\", got \"%s\"\n", result.Err, r.Err)
	}
}

func TestControllerCmdChanTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	controllerCh := controller.AddCmdChan(ssntp.START)

	// should time out
	_, err := controller.GetCmdChanResult(controllerCh, ssntp.START)
	if err == nil {
		t.Fatal(err)
	}

	// don't leave the result on the channel
	var result Result
	go controller.SendResultAndDelCmdChan(ssntp.START, result)
	_, err = controller.GetCmdChanResult(controllerCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerCloseChans(t *testing.T) {
	var result Result

	_ = controller.AddCmdChan(ssntp.START)
	go controller.SendResultAndDelCmdChan(ssntp.START, result)

	_ = controller.AddEventChan(ssntp.TraceReport)
	go controller.SendResultAndDelEventChan(ssntp.TraceReport, result)

	_ = controller.AddErrorChan(ssntp.StopFailure)
	go controller.SendResultAndDelErrorChan(ssntp.StopFailure, result)

	CloseControllerChans(controller)
	OpenControllerChans(controller)
}
