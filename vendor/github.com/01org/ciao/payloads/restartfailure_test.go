/* // Copyright (c) 2016 Intel Corporation
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
*/

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestRestartFailureUnmarshal(t *testing.T) {
	var error ErrorRestartFailure
	err := yaml.Unmarshal([]byte(testutil.RestartFailureYaml), &error)
	if err != nil {
		t.Error(err)
	}

	if error.InstanceUUID != testutil.InstanceUUID {
		t.Error("Wrong UUID field")
	}

	if error.Reason != RestartAlreadyRunning {
		t.Error("Wrong Error field")
	}
}

func TestRestartFailureMarshal(t *testing.T) {
	error := ErrorRestartFailure{
		InstanceUUID: testutil.InstanceUUID,
		Reason:       RestartAlreadyRunning,
	}

	y, err := yaml.Marshal(&error)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.RestartFailureYaml {
		t.Errorf("RestartFailure marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.RestartFailureYaml)
	}
}

func TestRestartFailureString(t *testing.T) {
	var stringTests = []struct {
		r        RestartFailureReason
		expected string
	}{
		{RestartNoInstance, "Instance does not exist"},
		{RestartInvalidPayload, "YAML payload is corrupt"},
		{RestartInvalidData, "Command section of YAML payload is corrupt or missing required information"},
		{RestartAlreadyRunning, "Instance is already running"},
		{RestartInstanceCorrupt, "Instance is corrupt"},
		{RestartLaunchFailure, "Failed to launch instance"},
		{RestartNetworkFailure, "Failed to locate VNIC for instance"},
	}
	error := ErrorRestartFailure{
		InstanceUUID: testutil.InstanceUUID,
	}
	for _, test := range stringTests {
		error.Reason = test.r
		s := error.Reason.String()
		if s != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, s)
		}
	}
}
