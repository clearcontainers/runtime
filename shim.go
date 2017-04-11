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

// ShimConfig holds configuration data related to a shim.
type ShimConfig struct {
	Path string
}

func startShim(process *vc.Process, config ShimConfig) (int, error) {
	if process.Token == "" {
		return -1, fmt.Errorf("Token cannot be empty")
	}

	if process.URL == "" {
		return -1, fmt.Errorf("URL cannot be empty")
	}

	if config.Path == "" {
		config.Path = defaultShimPath
	}
	ccLog.Infof("Shim binary path: %s", config.Path)

	cmd := exec.Command(config.Path, "-t", process.Token, "-u", process.URL)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return -1, err
	}

	return cmd.Process.Pid, nil
}

func startContainerShim(container *vc.Container, config ShimConfig) (int, error) {
	process := container.Process()

	pid, err := startShim(&process, config)
	if err != nil {
		return -1, err
	}

	if err := container.SetPid(pid); err != nil {
		return -1, err
	}

	return pid, nil
}
