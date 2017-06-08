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
	"encoding/json"
	"sync"
	"testing"

	"github.com/clearcontainers/proxy/api"
	"github.com/containers/virtcontainers/pkg/hyperstart"
	"github.com/containers/virtcontainers/pkg/hyperstart/mock"

	"github.com/stretchr/testify/assert"
)

// vmRig maintains a test environment for vm objects
type vmRig struct {
	t  *testing.T
	wg sync.WaitGroup

	// hyperstart mocking
	Hyperstart *mock.Hyperstart
}

func newVMRig(t *testing.T) *vmRig {
	return &vmRig{
		t: t,
	}
}

func (rig *vmRig) Start() {
	// Start hyperstart go routine
	rig.Hyperstart = mock.NewHyperstart(rig.t)
	rig.Hyperstart.Start()

	// Explicitly send READY message from hyperstart mock
	rig.wg.Add(1)
	go func() {
		rig.Hyperstart.SendMessage(int(hyperstart.ReadyCode), []byte{})
		rig.wg.Done()
	}()

}

func (rig *vmRig) Stop() {
	rig.Hyperstart.Stop()
	rig.wg.Wait()
}

const testVM = "testVM"

// CreateVM creates a vm instance that is connected to the rig's Hyperstart
// mock object.
func (rig *vmRig) CreateVM() *vm {
	ctlSocketPath, ioSocketPath := rig.Hyperstart.GetSocketPaths()

	vm := newVM(testVM, ctlSocketPath, ioSocketPath)
	assert.NotNil(rig.t, vm)

	err := vm.Connect()
	assert.Nil(rig.t, err)

	return vm
}

func (rig *vmRig) createBaseProcess() *hyperstart.Process {
	return &hyperstart.Process{
		Args: []string{"/bin/sh"},
		Envs: []hyperstart.EnvironmentVar{
			{
				Env:   "PATH",
				Value: "/sbin:/usr/sbin:/bin:/usr/bin",
			},
		},
		Workdir: "/",
	}

}

func (rig *vmRig) createHyperCmd(vm *vm, cmdName string, numTokens int, data []byte) *api.Hyper {
	tokens := make([]string, numTokens)
	for i := 0; i < numTokens; i++ {
		token, err := vm.AllocateToken()
		assert.Nil(rig.t, err)
		tokens[i] = string(token)
	}

	return &api.Hyper{
		HyperName: cmdName,
		Tokens:    tokens,
		Data:      data,
	}
}

func (rig *vmRig) createNewcontainer(vm *vm, numTokens int) *api.Hyper {
	process := rig.createBaseProcess()
	cmd := hyperstart.Container{
		Process: process,
	}

	data, err := json.Marshal(&cmd)
	assert.Nil(rig.t, err)

	return rig.createHyperCmd(vm, "newcontainer", numTokens, data)
}

func TestHyperRelocationNewcontainer(t *testing.T) {
	rig := newVMRig(t)
	rig.Start()

	vm := rig.CreateVM()

	// Relocate an execcmd command, giving 1 token as it should be (only
	// 1 process can be spawned using execcmd.
	cmd := rig.createNewcontainer(vm, 1)
	token := cmd.Tokens[0]
	// associate a dummy shim
	vm.AssociateShim(Token(token), 1, nil)
	// relocate
	_, err := vm.relocateHyperCommand(cmd)
	assert.Nil(t, err)

	// Check that the relocated command contains the seq numbers
	// corresponding to the token.
	session := vm.findSessionByToken(Token(token))
	cmdOut := hyperstart.Container{}
	err = json.Unmarshal(cmd.Data, &cmdOut)
	assert.Nil(t, err)
	assert.Equal(t, session.ioBase, cmdOut.Process.Stdio)
	assert.Equal(t, session.ioBase+1, cmdOut.Process.Stderr)

	// Giving more than 1 token should result in an error
	cmd = rig.createExecmd(vm, 2)
	_, err = vm.relocateHyperCommand(cmd)
	assert.NotNil(t, err)

	rig.Stop()

	vm.Close()
}

// In some case, we want to create containers without caring about the session
// between the process inside the VM and the host. One of those cases is the
// pause container created as we create a pod in virtcontainers.
func TestHyperRelocationNewcontainerNoToken(t *testing.T) {
	rig := newVMRig(t)
	rig.Start()

	vm := rig.CreateVM()

	// Relocate an execcmd command, no token given!
	cmd := rig.createNewcontainer(vm, 0)

	// don't associate a dummy shim, we should wait for one when no token is
	// given.

	// relocate
	_, err := vm.relocateHyperCommand(cmd)
	assert.Nil(t, err)

	// Check that the relocated command contains the seq numbers
	// of the null session
	session := &vm.nullSession
	cmdOut := hyperstart.ExecCommand{}
	err = json.Unmarshal(cmd.Data, &cmdOut)
	assert.Nil(t, err)
	assert.Equal(t, session.ioBase, cmdOut.Process.Stdio)
	assert.Equal(t, session.ioBase+1, cmdOut.Process.Stderr)

	rig.Stop()
	vm.Close()
}

func (rig *vmRig) createExecmd(vm *vm, numTokens int) *api.Hyper {
	process := rig.createBaseProcess()
	cmd := hyperstart.ExecCommand{
		Process: *process,
	}

	data, err := json.Marshal(&cmd)
	assert.Nil(rig.t, err)

	return rig.createHyperCmd(vm, "execcmd", numTokens, data)
}

func TestHyperRelocationExeccmd(t *testing.T) {
	rig := newVMRig(t)
	rig.Start()

	vm := rig.CreateVM()

	// Relocate an execcmd command, giving 1 token as it should be (only
	// 1 process can be spawned using execcmd.
	cmd := rig.createExecmd(vm, 1)
	token := cmd.Tokens[0]
	// associate a dummy shim
	vm.AssociateShim(Token(token), 1, nil)
	// relocate
	_, err := vm.relocateHyperCommand(cmd)
	assert.Nil(t, err)

	// Check that the relocated command contains the seq numbers
	// corresponding to the token.
	session := vm.findSessionByToken(Token(token))
	cmdOut := hyperstart.ExecCommand{}
	err = json.Unmarshal(cmd.Data, &cmdOut)
	assert.Nil(t, err)
	assert.Equal(t, session.ioBase, cmdOut.Process.Stdio)
	assert.Equal(t, session.ioBase+1, cmdOut.Process.Stderr)

	// Giving more than 1 token should result in an error
	cmd = rig.createExecmd(vm, 2)
	_, err = vm.relocateHyperCommand(cmd)
	assert.NotNil(t, err)

	rig.Stop()

	vm.Close()
}

// A command that doesn't require tokens should error out if one is supplied
// anyway.
func TestHyperRelocationPing(t *testing.T) {
	rig := newVMRig(t)
	rig.Start()

	vm := rig.CreateVM()

	// Relocate a ping command, giving 1 token as it should be (only
	// 1 process can be spawned using execcmd.
	cmd := rig.createHyperCmd(vm, "ping", 1, nil)
	originalCmd := *cmd
	_, err := vm.relocateHyperCommand(cmd)
	assert.NotNil(t, err)
	assert.Equal(t, originalCmd, *cmd)

	rig.Stop()

	vm.Close()
}

func TestRelocateProcessNonZeroSequenceNumbers(t *testing.T) {
	process := &hyperstart.Process{
		Args: []string{"/bin/sh"},
	}
	session := &ioSession{
		ioBase: 64,
	}

	process.Stdio = 128
	process.Stderr = 0
	err := relocateProcess(process, session)
	assert.NotNil(t, err)

	process.Stdio = 0
	process.Stderr = 128
	err = relocateProcess(process, session)
	assert.NotNil(t, err)
}

func TestRelocateInteractiveProcess(t *testing.T) {
	process := &hyperstart.Process{
		Args:     []string{"/bin/sh"},
		Terminal: true,
	}
	session := &ioSession{
		ioBase: 64,
	}

	err := relocateProcess(process, session)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), process.Stdio)
	assert.Equal(t, uint64(0), process.Stderr)
}
