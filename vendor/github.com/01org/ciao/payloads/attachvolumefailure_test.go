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

func TestAttachVolumeFailureUnmarshal(t *testing.T) {
	var error ErrorAttachVolumeFailure
	err := yaml.Unmarshal([]byte(testutil.AttachVolumeFailureYaml), &error)
	if err != nil {
		t.Error(err)
	}

	if error.InstanceUUID != testutil.InstanceUUID {
		t.Error("Wrong UUID field")
	}

	if error.VolumeUUID != testutil.VolumeUUID {
		t.Error("Wrong UUID field")
	}

	if error.Reason != AttachVolumeAttachFailure {
		t.Error("Wrong Error field")
	}
}

func TestAttachVolumeFailureMarshal(t *testing.T) {
	error := ErrorAttachVolumeFailure{
		InstanceUUID: testutil.InstanceUUID,
		VolumeUUID:   testutil.VolumeUUID,
		Reason:       AttachVolumeAttachFailure,
	}

	y, err := yaml.Marshal(&error)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.AttachVolumeFailureYaml {
		t.Errorf("AttachVolumeFailure marshalling failed\n[%s]\n vs\n[%s]",
			string(y), testutil.AttachVolumeFailureYaml)
	}
}

func TestAttachVolmeFailureString(t *testing.T) {
	var stringTests = []struct {
		r        AttachVolumeFailureReason
		expected string
	}{
		{AttachVolumeNoInstance, "Instance does not exist"},
		{AttachVolumeInvalidPayload, "YAML payload is corrupt"},
		{AttachVolumeInvalidData, "Command section of YAML payload is corrupt or missing required information"},
		{AttachVolumeAttachFailure, "Failed to attach volume to instance"},
		{AttachVolumeAlreadyAttached, "Volume already attached"},
		{AttachVolumeStateFailure, "State failure"},
		{AttachVolumeInstanceFailure, "Instance failure"},
		{AttachVolumeNotSupported, "Not Supported"},
	}
	error := ErrorAttachVolumeFailure{
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
