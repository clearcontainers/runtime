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

// ShimConfig describes a shim configuration.
type ShimConfig struct {
	Path string
	IP   string
	Port string
}

func startShim(config ShimConfig, pod *vc.Pod) (int, error) {
	containers := pod.GetContainers()

	if len(containers) != 1 {
		return -1, fmt.Errorf("Container list from pod is wrong, expecting only one container")
	}
	container := containers[0]

	token := container.GetToken()
	if token == "" {
		return -1, fmt.Errorf("Invalid token")
	}

	cmd := exec.Cmd{
		Path: config.Path,
		Args: []string{"-t", token, "-s", config.IP, "-p", config.Port},
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
