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
	"os/exec"

	vc "github.com/containers/virtcontainers"
)

// ShimInfo gathers information needed by the shim.
type ShimInfo struct {
	Path  string
	Token string
	IP    string
	Port  string
}

func startShim(pod *vc.Pod) (int, error) {
	containers := pod.GetContainers()

	if len(containers) != 1 {
		return -1, fmt.Errorf("Container list from pod is wrong, expecting only one container")
	}
	container := containers[0]

	shimInfo, err := getShimInfo(container)
	if err != nil {
		return -1, err
	}

	cmd := exec.Cmd{
		Path: shimInfo.Path,
		Args: []string{"-t", shimInfo.Token, "-s", shimInfo.IP, "-p", shimInfo.Port},
		Env:  os.Environ(),
	}

	if err := cmd.Start(); err != nil {
		return -1, err
	}
	pid := cmd.Process.Pid

	if err := container.SetPid(pid); err != nil {
		return -1, err
	}

	return pid, nil
}

func getShimInfo(container *vc.Container) (ShimInfo, error) {
	token := container.GetToken()
	if token == "" {
		return ShimInfo{}, fmt.Errorf("Invalid token")
	}

	shimInfo := ShimInfo{
		Path:  defaultShimPath,
		Token: token,
		IP:    defaultProxyIP,
		Port:  defaultProxyPort,
	}

	return shimInfo, nil
}
