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

import "testing"

func TestIsSystemMount(t *testing.T) {
	tests := []struct {
		mnt      string
		expected bool
	}{
		{"/sys", true},
		{"/sys/", true},
		{"/sys//", true},
		{"/sys/fs", true},
		{"/sys/fs/", true},
		{"/sys/fs/cgroup", true},
		{"/sysfoo", false},
		{"/home", false},
		{"/dev/block/", true},
	}

	for _, test := range tests {
		result := isSystemMount(test.mnt)
		if result != test.expected {
			t.Fatalf("Expected result for path %s : %v, got %v", test.mnt, test.expected, result)
		}
	}
}
