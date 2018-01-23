//
// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/stretchr/testify/assert"
)

const (
	testExecID      = "testExecID"
	testContainerID = "testContainerID"
)

func TestClosePostStartFDsAllNil(t *testing.T) {
	p := &process{}

	p.closePostStartFDs()
}

func TestClosePostStartFDsAllInitialized(t *testing.T) {
	rStdin, wStdin, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer wStdin.Close()

	rStdout, wStdout, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer rStdout.Close()

	rStderr, wStderr, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer rStderr.Close()

	rConsoleSocket, wConsoleSocket, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer wConsoleSocket.Close()

	rConsoleSock, wConsoleSock, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer wConsoleSock.Close()

	p := &process{
		process: libcontainer.Process{
			Stdin:         rStdin,
			Stdout:        wStdout,
			Stderr:        wStderr,
			ConsoleSocket: rConsoleSocket,
		},
		consoleSock: rConsoleSock,
	}

	p.closePostStartFDs()
}

func TestClosePostExitFDsAllNil(t *testing.T) {
	p := &process{}

	p.closePostExitFDs()
}

func TestClosePostExitFDsAllInitialized(t *testing.T) {
	rTermMaster, wTermMaster, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer wTermMaster.Close()

	rStdin, wStdin, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer rStdin.Close()

	rStdout, wStdout, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer wStdout.Close()

	rStderr, wStderr, err := os.Pipe()
	assert.Nil(t, err, "%v", err)
	defer wStderr.Close()

	p := &process{
		termMaster: rTermMaster,
		stdin:      wStdin,
		stdout:     rStdout,
		stderr:     rStderr,
	}

	p.closePostExitFDs()
}

func TestSetProcess(t *testing.T) {
	c := &container{
		processes: make(map[string]*process),
	}

	p := &process{
		id: testExecID,
	}

	c.setProcess(p)

	proc, exist := c.processes[testExecID]
	assert.True(t, exist, "Process entry should exist")

	assert.True(t, reflect.DeepEqual(p, proc),
		"Process structures should be identical: got %+v, expecting %+v",
		proc, p)
}

func TestDeleteProcess(t *testing.T) {
	c := &container{
		processes: make(map[string]*process),
	}

	p := &process{
		id: testExecID,
	}

	c.processes[testExecID] = p

	c.deleteProcess(testExecID)

	_, exist := c.processes[testExecID]
	assert.False(t, exist, "Process entry should not exist")
}

func TestGetProcessEntryExist(t *testing.T) {
	c := &container{
		processes: make(map[string]*process),
	}

	p := &process{
		id: testExecID,
	}

	c.processes[testExecID] = p

	proc, err := c.getProcess(testExecID)
	assert.Nil(t, err, "%v", err)

	assert.True(t, reflect.DeepEqual(p, proc),
		"Process structures should be identical: got %+v, expecting %+v",
		proc, p)
}

func TestGetProcessNoEntry(t *testing.T) {
	c := &container{
		processes: make(map[string]*process),
	}

	_, err := c.getProcess(testExecID)
	assert.Error(t, err, "Should fail because no entry has been created")
}

func TestGetContainerEntryExist(t *testing.T) {
	s := &sandbox{
		containers: make(map[string]*container),
	}

	c := &container{
		id: testContainerID,
	}

	s.containers[testContainerID] = c

	cont, err := s.getContainer(testContainerID)
	assert.Nil(t, err, "%v", err)

	assert.True(t, reflect.DeepEqual(c, cont),
		"Container structures should be identical: got %+v, expecting %+v",
		cont, c)
}

func TestGetContainerNoEntry(t *testing.T) {
	s := &sandbox{
		containers: make(map[string]*container),
	}

	_, err := s.getContainer(testContainerID)
	assert.Error(t, err, "Should fail because no entry has been created")
}

func TestSetContainer(t *testing.T) {
	s := &sandbox{
		containers: make(map[string]*container),
	}

	c := &container{
		id: testContainerID,
	}

	s.setContainer(testContainerID, c)

	cont, exist := s.containers[testContainerID]
	assert.True(t, exist, "Container entry should exist")

	assert.True(t, reflect.DeepEqual(c, cont),
		"Container structures should be identical: got %+v, expecting %+v",
		cont, c)
}

func TestDeleteContainer(t *testing.T) {
	s := &sandbox{
		containers: make(map[string]*container),
	}

	c := &container{
		id: testContainerID,
	}

	s.containers[testContainerID] = c

	s.deleteContainer(testContainerID)

	_, exist := s.containers[testContainerID]
	assert.False(t, exist, "Process entry should not exist")
}
