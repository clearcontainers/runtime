/*
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
*/

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestDetachVolumeFailureUnmarshal(t *testing.T) {
	var error ErrorDetachVolumeFailure
	err := yaml.Unmarshal([]byte(testutil.DetachVolumeFailureYaml), &error)
	if err != nil {
		t.Error(err)
	}

	if error.InstanceUUID != testutil.InstanceUUID {
		t.Error("Wrong UUID field")
	}

	if error.VolumeUUID != testutil.VolumeUUID {
		t.Error("Wrong UUID field")
	}

	if error.Reason != DetachVolumeDetachFailure {
		t.Error("Wrong Error field")
	}
}

func TestDetachVolumeFailureMarshal(t *testing.T) {
	error := ErrorDetachVolumeFailure{
		InstanceUUID: testutil.InstanceUUID,
		VolumeUUID:   testutil.VolumeUUID,
		Reason:       DetachVolumeDetachFailure,
	}

	y, err := yaml.Marshal(&error)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.DetachVolumeFailureYaml {
		t.Errorf("DetachVolumeFailure marshalling failed\n[%s]\n vs\n[%s]",
			string(y), testutil.DetachVolumeFailureYaml)
	}
}

func TestDetachVolmeFailureString(t *testing.T) {
	var stringTests = []struct {
		r        DetachVolumeFailureReason
		expected string
	}{
		{DetachVolumeNoInstance, "Instance does not exist"},
		{DetachVolumeInvalidPayload, "YAML payload is corrupt"},
		{DetachVolumeInvalidData, "Command section of YAML payload is corrupt or missing required information"},
		{DetachVolumeDetachFailure, "Failed to detach volume from instance"},
		{DetachVolumeNotAttached, "Volume not attached"},
		{DetachVolumeStateFailure, "State failure"},
		{DetachVolumeInstanceFailure, "Instance failure"},
		{DetachVolumeNotSupported, "Not Supported"},
	}
	error := ErrorDetachVolumeFailure{
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
