//
// Copyright © 2016 Intel Corporation
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

package osprepare

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/01org/ciao/clogger"
)

func getCommandOutput(command string, logger clogger.CiaoLog) string {
	splits := strings.Split(command, " ")
	c := exec.Command(splits[0], splits[1:]...)
	c.Env = os.Environ()
	// Force C locale
	c.Env = append(c.Env, "LC_ALL=C")
	c.Env = append(c.Env, "LANG=C")
	c.Stderr = os.Stderr

	out, err := c.Output()
	if err != nil {
		logger.Warningf("Failed to run %s: %s", splits[0], err)
		return ""
	}

	return string(out)
}

func getDockerVersion(logger clogger.CiaoLog) string {
	ret := getCommandOutput("docker --version", logger)
	var version string

	if n, _ := fmt.Sscanf(ret, "Docker version %s, build", &version); n != 1 {
		return ""
	}

	if strings.HasSuffix(version, ",") {
		return string(version[0 : len(version)-1])
	}
	return version
}

func getQemuVersion(logger clogger.CiaoLog) string {
	ret := getCommandOutput("qemu-system-x86_64 --version", logger)
	var version string

	if n, _ := fmt.Sscanf(ret, "QEMU emulator version %s, Copyright (c)", &version); n != 1 {
		return ""
	}

	if strings.HasSuffix(version, ",") {
		return string(version[0 : len(version)-1])
	}
	return version
}

// Determine if the given current version is less than the test version
// Note: Can only compare equal version schemas (i.e. same level of dots)
func versionLessThan(currentVer string, testVer string) bool {
	curSplits := strings.Split(currentVer, ".")
	testSplits := strings.Split(testVer, ".")

	max := len(curSplits)

	if l2 := len(testSplits); l2 < max {
		max = l2
	}

	iSplits := make([]int, max)
	tSplits := make([]int, max)

	for i := 0; i < max; i++ {
		iSplits[i], _ = strconv.Atoi(curSplits[i])
		tSplits[i], _ = strconv.Atoi(testSplits[i])
	}

	for i := 0; i < max; i++ {
		if i == 0 {
			if iSplits[i] < tSplits[i] {
				return true
			}
		} else {
			match := true
			for j := 0; j < i; j++ {
				if iSplits[j] != tSplits[j] {
					match = false
					break
				}
			}
			if match && iSplits[i] < tSplits[i] {
				return true
			}
		}
	}
	return false
}
