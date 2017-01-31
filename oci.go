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

	vc "github.com/containers/virtcontainers"
)

func containerExists(containerID string) (bool, error) {
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
		return fmt.Errorf("Missing container ID")
	}

	// container ID MUST be unique.
	exist, err := containerExists(containerID)
	if err != nil {
		return err
	}
	if exist == true {
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
