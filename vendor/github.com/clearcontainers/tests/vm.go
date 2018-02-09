/// Copyright (c) 2017 Intel Corporation
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

package tests

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

const procPath = "/proc"

var errFound = errors.New("found")

// IsVMRunning looks in /proc for a hypervisor process that contains
// the containerID in its command line
func IsVMRunning(containerID string) bool {
	err := filepath.Walk(procPath, func(path string, _ os.FileInfo, _ error) error {
		if path == "" {
			return filepath.SkipDir
		}

		info, err := os.Stat(path)
		if err != nil {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			return filepath.SkipDir
		}

		content, err := ioutil.ReadFile(filepath.Join(path, "cmdline"))
		if err != nil {
			return filepath.SkipDir
		}

		hypervisorRegexs := []string{".*/qemu.*-name.*" + containerID + ".*-qmp.*unix:.*/" + containerID + "/.*"}

		for _, regex := range hypervisorRegexs {
			matcher := regexp.MustCompile(regex)
			if matcher.MatchString(string(content)) {
				return errFound
			}
		}

		return nil
	})

	return err == errFound
}
