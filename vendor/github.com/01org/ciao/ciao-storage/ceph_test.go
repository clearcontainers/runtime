// Copyright (c) 2016 Intel Corporation
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

package storage

import "testing"

var cephDriver = CephDriver{
	ID: "unittest",
}

func TestCephIsValidSnapshotUUID(t *testing.T) {
	var stringTests = []struct {
		uuid     string
		expected string
	}{
		{"", "missing '@'"},
		{"a2dec44c-e1b5-40c0-a2b1-bc700d12cfde", "missing '@'"},
		{"a@b", "uuid not of form \"{UUID}@{UUID}\""},
	}
	for _, test := range stringTests {
		err := cephDriver.IsValidSnapshotUUID(test.uuid)
		if err.Error() != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, err)
		}
	}

	err := cephDriver.IsValidSnapshotUUID("dc1d3e23-e32a-49f5-8c59-402c13031d49@e1f4834b-af32-46d9-8ec3-e4cea3de78cb")
	if err != nil {
		t.Errorf("expected nil, got \"%s\"", err)
	}
}
