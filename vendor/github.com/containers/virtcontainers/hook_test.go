//
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
//

package virtcontainers

import (
	"os"
	"reflect"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Important to keep these values in sync with hook test binary
var testKeyHook = "test-key"
var testContainerIDHook = "test-container-id"
var testControllerIDHook = "test-controller-id"
var testProcessIDHook = 12345
var testBinHookPath = "/tmp/bin/hook"

func TestBuildHookState(t *testing.T) {
	expected := specs.State{
		Pid: testProcessIDHook,
	}

	hookState := buildHookState(testProcessIDHook)

	if reflect.DeepEqual(hookState, expected) == false {
		t.Fatal()
	}
}

func createHook(timeout int) *Hook {
	return &Hook{
		Path:    testBinHookPath,
		Args:    []string{testKeyHook, testContainerIDHook, testControllerIDHook},
		Env:     os.Environ(),
		Timeout: timeout,
	}
}

func createWrongHook() *Hook {
	return &Hook{
		Path: testBinHookPath,
		Args: []string{"wrong-args"},
		Env:  os.Environ(),
	}
}

func testRunHook(t *testing.T, timeout int) {
	hook := createHook(timeout)

	err := hook.runHook()
	if err != nil {
		t.Fatal()
	}
}

func TestRunHook(t *testing.T) {
	cleanUp()

	testRunHook(t, 0)
}

func TestRunHookTimeout(t *testing.T) {
	testRunHook(t, 1)
}

func TestRunHookExitFailure(t *testing.T) {
	hook := createWrongHook()

	err := hook.runHook()
	if err == nil {
		t.Fatal()
	}
}

func TestRunHookTimeoutFailure(t *testing.T) {
	hook := createHook(1)

	hook.Args = append(hook.Args, "2")

	err := hook.runHook()
	if err == nil {
		t.Fatal()
	}
}

func testHooks(t *testing.T, hook *Hook) {
	hooks := &Hooks{
		PreStartHooks:  []Hook{*hook},
		PostStartHooks: []Hook{*hook},
		PostStopHooks:  []Hook{*hook},
	}

	err := hooks.preStartHooks()
	if err != nil {
		t.Fatal(err)
	}

	err = hooks.postStartHooks()
	if err != nil {
		t.Fatal(err)
	}

	err = hooks.postStopHooks()
	if err != nil {
		t.Fatal(err)
	}
}

func testFailingHooks(t *testing.T, hook *Hook) {
	hooks := &Hooks{
		PreStartHooks:  []Hook{*hook},
		PostStartHooks: []Hook{*hook},
		PostStopHooks:  []Hook{*hook},
	}

	err := hooks.preStartHooks()
	if err == nil {
		t.Fatal(err)
	}

	err = hooks.postStartHooks()
	if err != nil {
		t.Fatal(err)
	}

	err = hooks.postStopHooks()
	if err != nil {
		t.Fatal(err)
	}
}

func TestHooks(t *testing.T) {
	testHooks(t, createHook(0))
}

func TestHooksTimeout(t *testing.T) {
	testHooks(t, createHook(1))
}

func TestFailingHooks(t *testing.T) {
	testFailingHooks(t, createWrongHook())
}

func TestEmptyHooks(t *testing.T) {
	hooks := &Hooks{}

	err := hooks.preStartHooks()
	if err != nil {
		t.Fatal(err)
	}

	err = hooks.postStartHooks()
	if err != nil {
		t.Fatal(err)
	}

	err = hooks.postStopHooks()
	if err != nil {
		t.Fatal(err)
	}
}
