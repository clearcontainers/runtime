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

package main

import (
	"os"
	"syscall"
	"testing"

	vc "github.com/containers/virtcontainers"
)

const mockShimPath = "/tmp/cc-runtime/bin/cc-shim"

var testToken = "test-token"
var testURL = "sheme://ipAddress:8080/path"

func TestStartShimTokenEmptyFailure(t *testing.T) {
	process := &vc.Process{
		URL: testURL,
	}

	if _, err := startShim(process, ShimConfig{}); err == nil {
		t.Fatalf("This test should fail because process.Token is empty")
	}
}

func TestStartShimURLEmptyFailure(t *testing.T) {
	process := &vc.Process{
		Token: testToken,
	}

	if _, err := startShim(process, ShimConfig{}); err == nil {
		t.Fatalf("This test should fail because process.Token is empty")
	}
}

func testStartShimSuccessful(t *testing.T, process *vc.Process, shimConfig ShimConfig) {
	pid, err := startShim(process, shimConfig)
	if err != nil {
		t.Fatal(err)
	}

	if pid < 0 {
		t.Fatalf("Invalid PID %d", pid)
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Could not find shim PID %d: %s", pid, err)
	}

	if err := p.Signal(syscall.SIGUSR1); err != nil {
		t.Fatalf("Could not stop shim PID %d: %s", pid, err)
	}
}

func TestStartShimDefaultShimPathSuccessful(t *testing.T) {
	defaultShimPath = mockShimPath

	process := &vc.Process{
		URL:   testURL,
		Token: testToken,
	}

	testStartShimSuccessful(t, process, ShimConfig{})
}

func TestStartShimSuccessful(t *testing.T) {
	process := &vc.Process{
		URL:   testURL,
		Token: testToken,
	}

	shimConfig := ShimConfig{
		Path: mockShimPath,
	}

	testStartShimSuccessful(t, process, shimConfig)
}

func TestStartContainerShimContainerEmptyFailure(t *testing.T) {
	if _, err := startContainerShim(&vc.Container{}, ShimConfig{}); err == nil {
		t.Fatal("This test should fail because container is empty")
	}
}
