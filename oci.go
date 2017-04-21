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
	"os"
	"path/filepath"
	"syscall"

	vc "github.com/containers/virtcontainers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Contants related to cgroup memory directory
const (
	cgroupsTasksFile = "tasks"
	cgroupsProcsFile = "cgroup.procs"
	cgroupsDirMode   = os.FileMode(0750)
	cgroupsFileMode  = os.FileMode(0640)
	cgroupsMountType = "cgroup"
)

var cgroupsMemDirPath = "/sys/fs/cgroup"

func containerExists(containerID string) (bool, error) {
	if containerID == "" {
		return false, errNeedContainerID
	}

	podStatusList, err := vc.ListPod()
	if err != nil {
		return false, err
	}

	for _, podStatus := range podStatusList {
		if podStatus.ID == containerID {
			return true, nil
		}
	}

	return false, nil
}

func validCreateParams(containerID, bundlePath string) error {
	// container ID MUST be provided.
	if containerID == "" {
		return errNeedContainerID
	}

	// bundle path MUST be provided.
	if bundlePath == "" {
		return errNeedBundlePath
	}

	// container ID MUST be unique.
	exist, err := containerExists(containerID)
	if err != nil {
		return err
	}
	if exist == true {
		return fmt.Errorf("ID already in use, unique ID should be provided")
	}

	// bundle path MUST be valid.
	fileInfo, err := os.Stat(bundlePath)
	if err != nil {
		return fmt.Errorf("Invalid bundle path '%s': %s", bundlePath, err)
	}
	if fileInfo.IsDir() == false {
		return fmt.Errorf("Invalid bundle path '%s', it should be a directory", bundlePath)
	}

	return nil
}

func validContainer(containerID string) error {
	// container ID MUST be provided.
	if containerID == "" {
		return errNeedContainerID
	}

	// container ID MUST exist.
	exist, err := containerExists(containerID)
	if err != nil {
		return err
	}
	if exist == false {
		return fmt.Errorf("Container ID does not exist")
	}

	return nil
}

func processRunning(pid int) (bool, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, nil
	}

	return true, nil
}

func stopContainer(podStatus vc.PodStatus) error {
	if len(podStatus.ContainersStatus) != 1 {
		return fmt.Errorf("ContainerStatus list from PodStatus is wrong, expecting only one ContainerStatus")
	}

	// Calling StopContainer allows to make sure the container is properly
	// stopped and removed from the pod. That way, the container's state is
	// updated to the expected "stopped" state.
	if _, err := vc.StopContainer(podStatus.ID, podStatus.ContainersStatus[0].ID); err != nil {
		return err
	}

	return nil
}

// processCgroupsPath process the cgroups path as expected from the
// OCI runtime specification. It returns a list of complete paths
// that should be created and used for every specified resource.
func processCgroupsPath(ociSpec specs.Spec) ([]string, error) {
	var cgroupsPathList []string

	if ociSpec.Linux.CgroupsPath == "" {
		return []string{}, nil
	}

	if ociSpec.Linux.Resources == nil {
		return []string{}, nil
	}

	if ociSpec.Linux.Resources.Memory != nil {
		memCgroupsPath, err := processCgroupsPathForResource(ociSpec, "memory")
		if err != nil {
			return []string{}, err
		}

		cgroupsPathList = append(cgroupsPathList, memCgroupsPath)
	}

	if ociSpec.Linux.Resources.CPU != nil {
		cpuCgroupsPath, err := processCgroupsPathForResource(ociSpec, "cpu")
		if err != nil {
			return []string{}, err
		}

		cgroupsPathList = append(cgroupsPathList, cpuCgroupsPath)
	}

	if ociSpec.Linux.Resources.Pids != nil {
		pidsCgroupsPath, err := processCgroupsPathForResource(ociSpec, "pids")
		if err != nil {
			return []string{}, err
		}

		cgroupsPathList = append(cgroupsPathList, pidsCgroupsPath)
	}

	if ociSpec.Linux.Resources.BlockIO != nil {
		blkIOCgroupsPath, err := processCgroupsPathForResource(ociSpec, "blkio")
		if err != nil {
			return []string{}, err
		}

		cgroupsPathList = append(cgroupsPathList, blkIOCgroupsPath)
	}

	return cgroupsPathList, nil
}

func processCgroupsPathForResource(ociSpec specs.Spec, resource string) (string, error) {
	if resource == "" {
		return "", errNeedLinuxResource
	}

	// Relative cgroups path provided.
	if filepath.IsAbs(ociSpec.Linux.CgroupsPath) == false {
		return filepath.Join(cgroupsMemDirPath, resource, ociSpec.Linux.CgroupsPath), nil
	}

	// Absolute cgroups path provided.
	var cgroupMount specs.Mount
	cgroupMountFound := false
	for _, mount := range ociSpec.Mounts {
		if mount.Type == "cgroup" {
			cgroupMount = mount
			cgroupMountFound = true
			break
		}
	}

	if cgroupMountFound == false {
		return "", fmt.Errorf("cgroupsPath is absolute, cgroup mount MUST exist")
	}

	if cgroupMount.Destination == "" {
		return "", fmt.Errorf("cgroupsPath is absolute, cgroup mount destination cannot be empty")
	}

	return filepath.Join(cgroupMount.Destination, resource, ociSpec.Linux.CgroupsPath), nil
}
