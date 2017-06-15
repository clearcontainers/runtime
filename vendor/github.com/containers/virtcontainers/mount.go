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

import "strings"

// These mounts need to be created by the agent within the VM
var systemMounts = []string{"/proc", "/dev", "/dev/pts", "/dev/shm", "/dev/mqueue", "/sys", "/sys/fs/cgroup"}

var systemMountPrefixes = []string{"/proc", "/dev", "/sys"}

func isSystemMount(m string) bool {
	for _, p := range systemMountPrefixes {
		if m == p || strings.HasPrefix(m, p+"/") {
			return true
		}
	}

	return false
}
