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
	"os/exec"
)

const cpBinaryName = "cp"

func fileCopy(srcPath, dstPath string) error {
	if srcPath == "" {
		return fmt.Errorf("Source path cannot be empty")
	}

	if dstPath == "" {
		return fmt.Errorf("Destination path cannot be empty")
	}

	binPath, err := exec.LookPath(cpBinaryName)
	if err != nil {
		return err
	}

	cmd := exec.Command(binPath, srcPath, dstPath)

	return cmd.Run()
}
