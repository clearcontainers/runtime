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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
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

func testProcessRunning(t *testing.T, pid int, expected bool) {
	running, err := processRunning(pid)
	if err != nil {
		t.Fatal(err)
	}

	if running != expected {
		t.Fatalf("Expecting PID %d to be 'running == %v'", pid, expected)
	}
}

func TestProcessRunningFailure(t *testing.T) {
	testProcessRunning(t, 99999, false)
}

func TestProcessRunningSuccessful(t *testing.T) {
	pid := os.Getpid()
	testProcessRunning(t, pid, true)
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
