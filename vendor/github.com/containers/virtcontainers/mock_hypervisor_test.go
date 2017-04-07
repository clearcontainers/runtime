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

package virtcontainers

import (
	"fmt"
	"testing"
	"time"
)

func TestMockHypervisorInit(t *testing.T) {
	var m *mockHypervisor

	wrongConfig := HypervisorConfig{
		KernelPath:     "",
		ImagePath:      "",
		HypervisorPath: "",
	}

	err := m.init(wrongConfig)
	if err == nil {
		t.Fatal()
	}

	rightConfig := HypervisorConfig{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	err = m.init(rightConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorCreatePod(t *testing.T) {
	var m *mockHypervisor

	config := PodConfig{}

	err := m.createPod(config)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorStartPod(t *testing.T) {
	var m *mockHypervisor

	startCh := make(chan struct{})
	stopCh := make(chan struct{})

	go m.startPod(startCh, stopCh)

	select {
	case <-startCh:
		break
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for start notification")
	}
}

func TestMockHypervisorStopPod(t *testing.T) {
	var m *mockHypervisor

	err := m.stopPod()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorAddDevice(t *testing.T) {
	var m *mockHypervisor

	err := m.addDevice(nil, imgDev)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorGetPodConsole(t *testing.T) {
	var m *mockHypervisor

	expected := ""

	if result := m.getPodConsole("testPodID"); result != expected {
		t.Fatalf("Got %s\nExpecting %s", result, expected)
	}
}
