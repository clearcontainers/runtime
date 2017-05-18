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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Contants related to cgroup memory directory
const (
	cgroupsTasksFile = "tasks"
	cgroupsProcsFile = "cgroup.procs"
	cgroupsDirMode   = os.FileMode(0750)
	cgroupsFileMode  = os.FileMode(0640)
	cgroupsMountType = "cgroup"

	// Filesystem type corresponding to CGROUP_SUPER_MAGIC as listed
	// here: http://man7.org/linux/man-pages/man2/statfs.2.html
	cgroupFsType = 0x27e0eb
)

var (
	errNeedLinuxResource = errors.New("Linux resource cannot be empty")
)

var cgroupsMemDirPath = "/sys/fs/cgroup"

// getContainerByPrefix returns the full containerID for a container
// whose ID matches the specified prefix.
//
// An error is returned if >1 containers are found with the specified
// prefix.
func getContainerIDByPrefix(containerID string) (string, error) {
	if containerID == "" {
		return "", fmt.Errorf("Missing container ID")
	}

	podStatusList, err := vc.ListPod()
	if err != nil {
		return "", err
	}

	var matches []string

	for _, podStatus := range podStatusList {
		if podStatus.ID == containerID {
			return containerID, nil
		}

		if strings.HasPrefix(podStatus.ID, containerID) {
			matches = append(matches, podStatus.ID)
		}
	}

	l := len(matches)

	if l == 1 {
		return matches[0], nil
	} else if l > 1 {
		return "", fmt.Errorf("Partial container ID not unique")
	}

	return "", nil
}

func validCreateParams(containerID, bundlePath string) error {
	// container ID MUST be provided.
	if containerID == "" {
		return fmt.Errorf("Missing container ID")
	}

	// container ID MUST be unique.
	fullID, err := getContainerIDByPrefix(containerID)
	if err != nil {
		return err
	}

	if fullID != "" {
		return fmt.Errorf("ID already in use, unique ID should be provided")
	}

	// bundle path MUST be provided.
	if bundlePath == "" {
		return fmt.Errorf("Missing bundle path")
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

func expandContainerID(containerID string) (fullID string, err error) {
	// container ID MUST be provided.
	if containerID == "" {
		return "", fmt.Errorf("Missing container ID")
	}

	// container ID MUST exist.
	fullID, err = getContainerIDByPrefix(containerID)
	if err != nil {
		return "", err
	}
	if fullID == "" {
		return "", fmt.Errorf("Container ID does not exist")
	}

	return fullID, nil
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

func stopContainer(status vc.ContainerStatus) error {
	// Calling StopContainer allows to make sure the container is properly
	// stopped and removed from the pod. That way, the container's state is
	// updated to the expected "stopped" state.
	if _, err := vc.StopContainer(status.ID, status.ID); err != nil {
		return err
	}

	return nil
}

// processCgroupsPath process the cgroups path as expected from the
// OCI runtime specification. It returns a list of complete paths
// that should be created and used for every specified resource.
func processCgroupsPath(ociSpec oci.CompatOCISpec) ([]string, error) {
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

		if memCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, memCgroupsPath)
		}
	}

	if ociSpec.Linux.Resources.CPU != nil {
		cpuCgroupsPath, err := processCgroupsPathForResource(ociSpec, "cpu")
		if err != nil {
			return []string{}, err
		}

		if cpuCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, cpuCgroupsPath)
		}
	}

	if ociSpec.Linux.Resources.Pids != nil {
		pidsCgroupsPath, err := processCgroupsPathForResource(ociSpec, "pids")
		if err != nil {
			return []string{}, err
		}

		if pidsCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, pidsCgroupsPath)
		}
	}

	if ociSpec.Linux.Resources.BlockIO != nil {
		blkIOCgroupsPath, err := processCgroupsPathForResource(ociSpec, "blkio")
		if err != nil {
			return []string{}, err
		}

		if blkIOCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, blkIOCgroupsPath)
		}
	}

	return cgroupsPathList, nil
}

func processCgroupsPathForResource(ociSpec oci.CompatOCISpec, resource string) (string, error) {
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

	cgroupPath := filepath.Join(cgroupMount.Destination, resource)

	// It is not an error to have this cgroup not mounted. It is usually
	// due to an old kernel version with missing support for specific
	// cgroups.
	if !isCgroupMounted(cgroupPath) {
		ccLog.Infof("cgroup path %s not mounted", cgroupPath)
		return "", nil
	}

	ccLog.Infof("cgroup path %s mounted", cgroupPath)

	return filepath.Join(cgroupPath, ociSpec.Linux.CgroupsPath), nil
}

func isCgroupMounted(cgroupPath string) bool {
	var statFs syscall.Statfs_t

	if err := syscall.Statfs(cgroupPath, &statFs); err != nil {
		return false
	}

	if statFs.Type != int64(cgroupFsType) {
		return false
	}

	return true
}
