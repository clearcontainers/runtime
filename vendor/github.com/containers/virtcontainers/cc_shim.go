//
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
//

package virtcontainers

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type ccShim struct{}

// CCShimConfig is the structure providing specific configuration
// for ccShim implementation.
type CCShimConfig struct {
	Path  string
	Debug bool
}

var consoleFileMode = os.FileMode(0660)

// start is the ccShim start implementation.
// It starts the cc-shim binary with URL and token flags provided by
// the proxy.
func (s *ccShim) start(pod Pod, params ShimParams) (int, error) {
	if pod.config == nil {
		return -1, fmt.Errorf("Pod config cannot be nil")
	}

	config, ok := newShimConfig(*(pod.config)).(CCShimConfig)
	if !ok {
		return -1, fmt.Errorf("Wrong shim config type, should be CCShimConfig type")
	}

	if config.Path == "" {
		return -1, fmt.Errorf("Shim path cannot be empty")
	}

	if params.Token == "" {
		return -1, fmt.Errorf("Token cannot be empty")
	}

	if params.URL == "" {
		return -1, fmt.Errorf("URL cannot be empty")
	}

	if params.Container == "" {
		return -1, fmt.Errorf("Container cannot be empty")
	}

	args := []string{config.Path, "-c", params.Container, "-t", params.Token, "-u", params.URL}
	if config.Debug {
		args = append(args, "-d")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()

	if !params.Detach {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	var f *os.File
	var err error
	if params.Console != "" {
		f, err = os.OpenFile(params.Console, os.O_RDWR, consoleFileMode)
		if err != nil {
			return -1, err
		}

		cmd.Stdin = f
		cmd.Stdout = f
		cmd.Stderr = f
		cmd.SysProcAttr = &syscall.SysProcAttr{
			// Create Session
			Setsid: true,

			// Set Controlling terminal to Ctty
			Setctty: true,
			Ctty:    int(f.Fd()),
		}

	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	if err := cmd.Start(); err != nil {
		return -1, err
	}

	return cmd.Process.Pid, nil
}
