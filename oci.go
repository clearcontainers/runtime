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
	errNeedLinuxResource     = errors.New("Linux resource cannot be empty")
	errPrefixContIDNotUnique = errors.New("Partial container ID not unique")
)

var cgroupsDirPath = "/sys/fs/cgroup"

// getContainerInfo returns the container status and its pod ID.
// It internally expands the container ID from the prefix provided.
// An error is returned if >1 containers are found with the specified
// prefix.
func getContainerInfo(containerID string) (vc.ContainerStatus, string, error) {
	var cStatus vc.ContainerStatus
	var podID string

	// container ID MUST be provided.
	if containerID == "" {
		return vc.ContainerStatus{}, "", fmt.Errorf("Missing container ID")
	}

	podStatusList, err := vc.ListPod()
	if err != nil {
		return vc.ContainerStatus{}, "", err
	}

	matchFound := false
	for _, podStatus := range podStatusList {
		for _, containerStatus := range podStatus.ContainersStatus {
			if containerStatus.ID == containerID {
				return containerStatus, podStatus.ID, nil
			}

			if strings.HasPrefix(containerStatus.ID, containerID) {
				if matchFound {
					return vc.ContainerStatus{}, "", errPrefixContIDNotUnique
				}

				matchFound = true
				cStatus = containerStatus
				podID = podStatus.ID
			}
		}
	}

	if matchFound {
		return cStatus, podID, nil
	}

	return vc.ContainerStatus{}, "", nil
}

func getExistingContainerInfo(containerID string) (vc.ContainerStatus, string, error) {
	cStatus, podID, err := getContainerInfo(containerID)
	if err != nil {
		return vc.ContainerStatus{}, "", err
	}

	// container ID MUST exist.
	if cStatus.ID == "" {
		return vc.ContainerStatus{}, "", fmt.Errorf("Container ID does not exist")
	}

	return cStatus, podID, nil
}

func validCreateParams(containerID, bundlePath string) (string, error) {
	// container ID MUST be provided.
	if containerID == "" {
		return "", fmt.Errorf("Missing container ID")
	}

	// container ID MUST be unique.
	cStatus, _, err := getContainerInfo(containerID)
	if err != nil {
		return "", err
	}

	if cStatus.ID != "" {
		return "", fmt.Errorf("ID already in use, unique ID should be provided")
	}

	// bundle path MUST be provided.
	if bundlePath == "" {
		return "", fmt.Errorf("Missing bundle path")
	}

	// bundle path MUST be valid.
	fileInfo, err := os.Stat(bundlePath)
	if err != nil {
		return "", fmt.Errorf("Invalid bundle path '%s': %s", bundlePath, err)
	}
	if fileInfo.IsDir() == false {
		return "", fmt.Errorf("Invalid bundle path '%s', it should be a directory", bundlePath)
	}

	resolved, err := resolvePath(bundlePath)
	if err != nil {
		return "", err
	}

	return resolved, nil
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

func stopContainer(podID string, status vc.ContainerStatus) error {
	containerType, err := oci.GetContainerType(status.Annotations)
	if err != nil {
		return err
	}

	switch containerType {
	case vc.PodSandbox:
		// Calling StopPod allows to make sure the pod is properly
		// stopped. That way, containers/pod states are updated to
		// the expected "stopped" state.
		if _, err := vc.StopPod(podID); err != nil {
			return err
		}
	case vc.PodContainer:
		// Calling StopContainer allows to make sure the container is
		// properly stopped and removed from the pod. That way, the
		// container's state is updated to the expected "stopped" state.
		if _, err := vc.StopContainer(podID, status.ID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Invalid container type found")
	}

	return nil
}

// processCgroupsPath process the cgroups path as expected from the
// OCI runtime specification. It returns a list of complete paths
// that should be created and used for every specified resource.
func processCgroupsPath(ociSpec oci.CompatOCISpec, isPod bool) ([]string, error) {
	var cgroupsPathList []string

	if ociSpec.Linux.CgroupsPath == "" {
		return []string{}, nil
	}

	if ociSpec.Linux.Resources == nil {
		return []string{}, nil
	}

	if ociSpec.Linux.Resources.Memory != nil {
		memCgroupsPath, err := processCgroupsPathForResource(ociSpec, "memory", isPod)
		if err != nil {
			return []string{}, err
		}

		if memCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, memCgroupsPath)
		}
	}

	if ociSpec.Linux.Resources.CPU != nil {
		cpuCgroupsPath, err := processCgroupsPathForResource(ociSpec, "cpu", isPod)
		if err != nil {
			return []string{}, err
		}

		if cpuCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, cpuCgroupsPath)
		}
	}

	if ociSpec.Linux.Resources.Pids != nil {
		pidsCgroupsPath, err := processCgroupsPathForResource(ociSpec, "pids", isPod)
		if err != nil {
			return []string{}, err
		}

		if pidsCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, pidsCgroupsPath)
		}
	}

	if ociSpec.Linux.Resources.BlockIO != nil {
		blkIOCgroupsPath, err := processCgroupsPathForResource(ociSpec, "blkio", isPod)
		if err != nil {
			return []string{}, err
		}

		if blkIOCgroupsPath != "" {
			cgroupsPathList = append(cgroupsPathList, blkIOCgroupsPath)
		}
	}

	return cgroupsPathList, nil
}

func processCgroupsPathForResource(ociSpec oci.CompatOCISpec, resource string, isPod bool) (string, error) {
	if resource == "" {
		return "", errNeedLinuxResource
	}

	// Relative cgroups path provided.
	if filepath.IsAbs(ociSpec.Linux.CgroupsPath) == false {
		return filepath.Join(cgroupsDirPath, resource, ociSpec.Linux.CgroupsPath), nil
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

	if !cgroupMountFound {
		if isPod {
			return "", fmt.Errorf("cgroupsPath %q is absolute, cgroup mount MUST exist",
				ociSpec.Linux.CgroupsPath)
		}

		// In case of container (CRI-O), if the mount point is not
		// provided, we assume this is a relative path.
		return filepath.Join(cgroupsDirPath, resource, ociSpec.Linux.CgroupsPath), nil
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
