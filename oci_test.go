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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/opencontainers/runc/libcontainer/utils"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

var (
	consolePathTest       = "console-test"
	consoleSocketPathTest = "console-socket-test"
)

func TestGetContainerInfoContainerIDEmptyFailure(t *testing.T) {
	assert := assert.New(t)
	status, _, err := getContainerInfo("")

	assert.Error(err, "This test should fail because containerID is empty")
	assert.Empty(status.ID, "Expected blank fullID, but got %v", status.ID)
}

func TestValidCreateParamsContainerIDEmptyFailure(t *testing.T) {
	assert := assert.New(t)
	_, err := validCreateParams("", "")

	assert.Error(err, "This test should fail because containerID is empty")
}

func TestGetExistingContainerInfoContainerIDEmptyFailure(t *testing.T) {
	assert := assert.New(t)
	status, _, err := getExistingContainerInfo("")

	assert.Error(err, "This test should fail because containerID is empty")
	assert.Empty(status.ID, "Expected blank fullID, but got %v", status.ID)
}

func testProcessCgroupsPath(t *testing.T, ociSpec oci.CompatOCISpec, expected []string) {
	assert := assert.New(t)
	result, err := processCgroupsPath(ociSpec, true)

	assert.NoError(err)

	if reflect.DeepEqual(result, expected) == false {
		assert.FailNow("DeepEqual failed", "Result path %q should match the expected one %q", result, expected)
	}
}

func TestProcessCgroupsPathEmptyPathSuccessful(t *testing.T) {
	ociSpec := oci.CompatOCISpec{}

	ociSpec.Linux = &specs.Linux{
		CgroupsPath: "",
	}

	testProcessCgroupsPath(t, ociSpec, []string{})
}

func TestProcessCgroupsPathRelativePathSuccessful(t *testing.T) {
	relativeCgroupsPath := "relative/cgroups/path"
	cgroupsDirPath = "/foo/runtime/base"

	ociSpec := oci.CompatOCISpec{}

	ociSpec.Linux = &specs.Linux{
		Resources: &specs.LinuxResources{
			Memory: &specs.LinuxMemory{},
		},
		CgroupsPath: relativeCgroupsPath,
	}

	testProcessCgroupsPath(t, ociSpec, []string{filepath.Join(cgroupsDirPath, "memory", relativeCgroupsPath)})
}

func TestProcessCgroupsPathAbsoluteNoCgroupMountFailure(t *testing.T) {
	assert := assert.New(t)
	absoluteCgroupsPath := "/absolute/cgroups/path"

	ociSpec := oci.CompatOCISpec{}

	ociSpec.Linux = &specs.Linux{
		Resources: &specs.LinuxResources{
			Memory: &specs.LinuxMemory{},
		},
		CgroupsPath: absoluteCgroupsPath,
	}

	_, err := processCgroupsPath(ociSpec, true)
	assert.Error(err, "This test should fail because no cgroup mount provided")
}

func TestProcessCgroupsPathAbsoluteNoCgroupMountDestinationFailure(t *testing.T) {
	assert := assert.New(t)
	absoluteCgroupsPath := "/absolute/cgroups/path"

	ociSpec := oci.CompatOCISpec{}

	ociSpec.Linux = &specs.Linux{
		Resources: &specs.LinuxResources{
			Memory: &specs.LinuxMemory{},
		},
		CgroupsPath: absoluteCgroupsPath,
	}

	ociSpec.Mounts = []specs.Mount{
		{
			Type: "cgroup",
		},
	}

	_, err := processCgroupsPath(ociSpec, true)
	assert.Error(err, "This test should fail because no cgroup mount destination provided")
}

func TestProcessCgroupsPathAbsoluteSuccessful(t *testing.T) {
	assert := assert.New(t)

	if os.Geteuid() != 0 {
		t.Skip(testDisabledNeedRoot)
	}

	memoryResource := "memory"
	absoluteCgroupsPath := "/cgroup/mount/destination"

	cgroupMountDest, err := ioutil.TempDir("", "cgroup-memory-")
	assert.NoError(err)
	defer os.RemoveAll(cgroupMountDest)

	resourceMountPath := filepath.Join(cgroupMountDest, memoryResource)
	err = os.MkdirAll(resourceMountPath, cgroupsDirMode)
	assert.NoError(err)

	err = syscall.Mount("go-test", resourceMountPath, "cgroup", 0, memoryResource)
	assert.NoError(err)
	defer syscall.Unmount(resourceMountPath, 0)

	ociSpec := oci.CompatOCISpec{}

	ociSpec.Linux = &specs.Linux{
		Resources: &specs.LinuxResources{
			Memory: &specs.LinuxMemory{},
		},
		CgroupsPath: absoluteCgroupsPath,
	}

	ociSpec.Mounts = []specs.Mount{
		{
			Type:        "cgroup",
			Destination: cgroupMountDest,
		},
	}

	testProcessCgroupsPath(t, ociSpec, []string{filepath.Join(resourceMountPath, absoluteCgroupsPath)})
}

func TestSetupConsoleExistingConsolePathSuccessful(t *testing.T) {
	assert := assert.New(t)
	console, err := setupConsole(consolePathTest, "")

	assert.NoError(err)
	assert.Equal(console, consolePathTest, "Got %q, Expecting %q", console, consolePathTest)
}

func TestSetupConsoleExistingConsolePathAndConsoleSocketPathSuccessful(t *testing.T) {
	assert := assert.New(t)
	console, err := setupConsole(consolePathTest, consoleSocketPathTest)

	assert.NoError(err)
	assert.Equal(console, consolePathTest, "Got %q, Expecting %q", console, consolePathTest)
}

func TestSetupConsoleEmptyPathsSuccessful(t *testing.T) {
	assert := assert.New(t)

	console, err := setupConsole("", "")
	assert.NoError(err)
	assert.Empty(console, "Console path should be empty, got %q instead", console)
}

func TestSetupConsoleExistingConsoleSocketPath(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "test-socket")
	assert.NoError(err)
	defer os.RemoveAll(dir)

	sockName := filepath.Join(dir, "console.sock")

	l, err := net.Listen("unix", sockName)
	assert.NoError(err)

	console, err := setupConsole("", sockName)
	assert.NoError(err)

	waitCh := make(chan error)
	go func() {
		conn, err1 := l.Accept()
		if err != nil {
			waitCh <- err1
		}

		uConn, ok := conn.(*net.UnixConn)
		if !ok {
			waitCh <- fmt.Errorf("casting to *net.UnixConn failed")
		}

		f, err1 := uConn.File()
		if err != nil {
			waitCh <- err1
		}

		_, err1 = utils.RecvFd(f)
		waitCh <- err1
	}()

	assert.NotEmpty(console, "Console socket path should not be empty")

	err = <-waitCh
	assert.NoError(err)
}

func TestSetupConsoleNotExistingSocketPathFailure(t *testing.T) {
	assert := assert.New(t)

	console, err := setupConsole("", "unknown-sock-path")
	assert.Error(err, "This test should fail because the console socket path does not exist")
	assert.Empty(console, "This test should fail because the console socket path does not exist")
}

func testNoNeedForOutput(t *testing.T, detach bool, tty bool, expected bool) {
	assert := assert.New(t)
	result := noNeedForOutput(detach, tty)

	assert.Equal(result, expected)
}

func TestNoNeedForOutputDetachTrueTtyTrue(t *testing.T) {
	testNoNeedForOutput(t, true, true, true)
}

func TestNoNeedForOutputDetachFalseTtyTrue(t *testing.T) {
	testNoNeedForOutput(t, false, true, false)
}

func TestNoNeedForOutputDetachFalseTtyFalse(t *testing.T) {
	testNoNeedForOutput(t, false, false, false)
}

func TestNoNeedForOutputDetachTrueTtyFalse(t *testing.T) {
	testNoNeedForOutput(t, true, false, false)
}
