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
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestContainerExistsContainerIDEmptyFailure(t *testing.T) {
	if _, err := containerExists(""); err == nil {
		t.Fatalf("This test should fail because containerID is empty")
	}
}

func TestValidCreateParamsContainerIDEmptyFailure(t *testing.T) {
	if err := validCreateParams("", ""); err == nil {
		t.Fatalf("This test should fail because containerID is empty")
	}
}

func TestValidContainerContainerIDEmptyFailure(t *testing.T) {
	if err := validContainer(""); err == nil {
		t.Fatalf("This test should fail because containerID is empty")
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
	if err := stopContainer(vc.PodStatus{}); err == nil {
		t.Fatalf("This test should fail because PodStatus is empty")
	}
}

func TestStopContainerTooManyContainerStatusesFailure(t *testing.T) {
	podStatus := vc.PodStatus{}

	for i := 0; i < 2; i++ {
		podStatus.ContainersStatus = append(podStatus.ContainersStatus, vc.ContainerStatus{})
	}

	if err := stopContainer(vc.PodStatus{}); err == nil {
		t.Fatalf("This test should fail because PodStatus has too many container statuses")
	}
}

func testProcessCgroupsPath(t *testing.T, ociSpec specs.Spec, expected []string) {
	result, err := processCgroupsPath(ociSpec)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Result path %q should match the expected one %q", result, expected)
	}
}

func TestProcessCgroupsPathEmptyPathSuccessful(t *testing.T) {
	ociSpec := specs.Spec{
		Linux: &specs.Linux{
			CgroupsPath: "",
		},
	}

	testProcessCgroupsPath(t, ociSpec, []string{})
}

func TestProcessCgroupsPathRelativePathSuccessful(t *testing.T) {
	relativeCgroupsPath := "relative/cgroups/path"
	cgroupsMemDirPath = "/foo/runtime/base"

	ociSpec := specs.Spec{
		Linux: &specs.Linux{
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{},
			},
			CgroupsPath: relativeCgroupsPath,
		},
	}

	testProcessCgroupsPath(t, ociSpec, []string{filepath.Join(cgroupsMemDirPath, "memory", relativeCgroupsPath)})
}

func TestProcessCgroupsPathAbsoluteNoCgroupMountFailure(t *testing.T) {
	absoluteCgroupsPath := "/absolute/cgroups/path"

	ociSpec := specs.Spec{
		Linux: &specs.Linux{
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{},
			},
			CgroupsPath: absoluteCgroupsPath,
		},
	}

	_, err := processCgroupsPath(ociSpec)
	if err == nil {
		t.Fatalf("This test should fail because no cgroup mount provided")
	}
}

func TestProcessCgroupsPathAbsoluteNoCgroupMountDestinationFailure(t *testing.T) {
	absoluteCgroupsPath := "/absolute/cgroups/path"

	ociSpec := specs.Spec{
		Linux: &specs.Linux{
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{},
			},
			CgroupsPath: absoluteCgroupsPath,
		},
		Mounts: []specs.Mount{
			{
				Type: "cgroup",
			},
		},
	}

	_, err := processCgroupsPath(ociSpec)
	if err == nil {
		t.Fatalf("This test should fail because no cgroup mount destination provided")
	}
}

func TestProcessCgroupsPathAbsoluteSuccessful(t *testing.T) {
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

	ociSpec := specs.Spec{
		Linux: &specs.Linux{
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{},
			},
			CgroupsPath: absoluteCgroupsPath,
		},
		Mounts: []specs.Mount{
			{
				Type:        "cgroup",
				Destination: cgroupMountDest,
			},
		},
	}

	testProcessCgroupsPath(t, ociSpec, []string{filepath.Join(resourceMountPath, absoluteCgroupsPath)})
}
