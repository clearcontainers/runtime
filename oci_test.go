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

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/opencontainers/runc/libcontainer/utils"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var (
	consolePathTest       = "console-test"
	consoleSocketPathTest = "console-socket-test"
)

func TestGetContainerInfoContainerIDEmptyFailure(t *testing.T) {
	status, _, err := getContainerInfo("")
	if err == nil {
		t.Fatalf("This test should fail because containerID is empty")
	}

	if status.ID != "" {
		t.Fatalf("Expected blank fullID, but got %v", status.ID)
	}
}

func TestValidCreateParamsContainerIDEmptyFailure(t *testing.T) {
	_, err := validCreateParams("", "")
	if err == nil {
		t.Fatalf("This test should fail because containerID is empty")
	}
}

func TestGetExistingContainerInfoContainerIDEmptyFailure(t *testing.T) {
	status, _, err := getExistingContainerInfo("")

	if err == nil {
		t.Fatalf("This test should fail because containerID is empty")
	}

	if status.ID != "" {
		t.Fatalf("Expected blank fullID, but got %v", status.ID)
	}
}

func TestStopContainerPodStatusEmptyFailure(t *testing.T) {
	if err := stopContainer("", vc.ContainerStatus{}); err == nil {
		t.Fatalf("This test should fail because PodStatus is empty")
	}
}

func TestStopContainerTooManyContainerStatusesFailure(t *testing.T) {
	podStatus := vc.PodStatus{}

	for i := 0; i < 2; i++ {
		podStatus.ContainersStatus = append(podStatus.ContainersStatus, vc.ContainerStatus{})
	}

	if err := stopContainer("", vc.ContainerStatus{}); err == nil {
		t.Fatalf("This test should fail because PodStatus has too many container statuses")
	}
}

func testProcessCgroupsPath(t *testing.T, ociSpec oci.CompatOCISpec, expected []string) {
	result, err := processCgroupsPath(ociSpec, true)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Result path %q should match the expected one %q", result, expected)
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
	absoluteCgroupsPath := "/absolute/cgroups/path"

	ociSpec := oci.CompatOCISpec{}

	ociSpec.Linux = &specs.Linux{
		Resources: &specs.LinuxResources{
			Memory: &specs.LinuxMemory{},
		},
		CgroupsPath: absoluteCgroupsPath,
	}

	_, err := processCgroupsPath(ociSpec, true)
	if err == nil {
		t.Fatalf("This test should fail because no cgroup mount provided")
	}
}

func TestProcessCgroupsPathAbsoluteNoCgroupMountDestinationFailure(t *testing.T) {
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
	if err == nil {
		t.Fatalf("This test should fail because no cgroup mount destination provided")
	}
}

func TestProcessCgroupsPathAbsoluteSuccessful(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledNeedRoot)
	}

	memoryResource := "memory"
	absoluteCgroupsPath := "/cgroup/mount/destination"

	cgroupMountDest, err := ioutil.TempDir("", "cgroup-memory-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cgroupMountDest)

	resourceMountPath := filepath.Join(cgroupMountDest, memoryResource)
	if err := os.MkdirAll(resourceMountPath, cgroupsDirMode); err != nil {
		t.Fatal(err)
	}

	if err := syscall.Mount("go-test", resourceMountPath, "cgroup", 0, memoryResource); err != nil {
		t.Fatal(err)
	}
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
	console, err := setupConsole(consolePathTest, "")
	if err != nil {
		t.Fatal(err)
	}

	if console != consolePathTest {
		t.Fatalf("Got %q, Expecting %q", console, consolePathTest)
	}
}

func TestSetupConsoleExistingConsolePathAndConsoleSocketPathSuccessful(t *testing.T) {
	console, err := setupConsole(consolePathTest, consoleSocketPathTest)
	if err != nil {
		t.Fatal(err)
	}

	if console != consolePathTest {
		t.Fatalf("Got %q, Expecting %q", console, consolePathTest)
	}
}

func TestSetupConsoleEmptyPathsSuccessful(t *testing.T) {
	console, err := setupConsole("", "")
	if err != nil {
		t.Fatal(err)
	}

	if console != "" {
		t.Fatalf("Console path should be empty, got %q instead", console)
	}
}

func TestSetupConsoleExistingConsoleSocketPath(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-socket")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	sockName := filepath.Join(dir, "console.sock")

	l, err := net.Listen("unix", sockName)
	if err != nil {
		t.Fatal(err)
	}

	console, err := setupConsole("", sockName)
	if err != nil {
		t.Fatal(err)
	}

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

	if console == "" {
		t.Fatal("Console socket path should not be empty")
	}

	if err := <-waitCh; err != nil {
		t.Fatal(err)
	}
}

func TestSetupConsoleNotExistingSocketPathFailure(t *testing.T) {
	console, err := setupConsole("", "unknown-sock-path")
	if err == nil && console != "" {
		t.Fatalf("This test should fail because the console socket path does not exist")
	}
}

func testNoNeedForOutput(t *testing.T, detach bool, tty bool, expected bool) {
	result := noNeedForOutput(detach, tty)
	if result != expected {
		t.Fatalf("Expecting %t, Got %t", expected, result)
	}
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
