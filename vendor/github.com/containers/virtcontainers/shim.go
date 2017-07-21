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
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"
)

// ShimType describes a shim type.
type ShimType string

const (
	// CCShimType is the ccShim.
	CCShimType ShimType = "ccShim"

	// NoopShimType is the noopShim.
	NoopShimType ShimType = "noopShim"
)

var waitForShimTimeout = 5.0

// ShimParams is the structure providing specific parameters needed
// for the execution of the shim binary.
type ShimParams struct {
	Token   string
	URL     string
	Console string
	Detach  bool
}

// Set sets a shim type based on the input string.
func (pType *ShimType) Set(value string) error {
	switch value {
	case "noopShim":
		*pType = NoopShimType
		return nil
	case "ccShim":
		*pType = CCShimType
		return nil
	default:
		return fmt.Errorf("Unknown shim type %s", value)
	}
}

// String converts a shim type to a string.
func (pType *ShimType) String() string {
	switch *pType {
	case NoopShimType:
		return string(NoopShimType)
	case CCShimType:
		return string(CCShimType)
	default:
		return ""
	}
}

// newShim returns a shim from a shim type.
func newShim(pType ShimType) (shim, error) {
	switch pType {
	case NoopShimType:
		return &noopShim{}, nil
	case CCShimType:
		return &ccShim{}, nil
	default:
		return &noopShim{}, nil
	}
}

// newShimConfig returns a shim config from a generic PodConfig interface.
func newShimConfig(config PodConfig) interface{} {
	switch config.ShimType {
	case NoopShimType:
		return nil
	case CCShimType:
		var ccConfig CCShimConfig
		err := mapstructure.Decode(config.ShimConfig, &ccConfig)
		if err != nil {
			return err
		}
		return ccConfig
	default:
		return nil
	}
}

func stopShim(pid int) error {
	if pid <= 0 {
		return nil
	}

	virtLog.Infof("Stopping shim PID %d", pid)

	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		return err
	}

	return nil
}

func isShimRunning(pid int) (bool, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, nil
	}

	return true, nil
}

// waitForShim waits for the end of the shim unless it reaches the timeout
// first, returning an error in that case.
func waitForShim(pid int) error {
	if pid <= 0 {
		return nil
	}

	tInit := time.Now()
	for {
		running, err := isShimRunning(pid)
		if err != nil {
			return err
		}

		if !running {
			break
		}

		if time.Since(tInit).Seconds() >= waitForShimTimeout {
			return fmt.Errorf("Shim still running, timeout %f s has been reached", waitForShimTimeout)
		}

		// Let's avoid to run a too busy loop
		time.Sleep(time.Duration(100) * time.Millisecond)
	}

	return nil
}

// shim is the virtcontainers shim interface.
type shim interface {
	// start starts the shim relying on its configuration and on
	// parameters provided.
	start(pod Pod, params ShimParams) (int, error)
}
