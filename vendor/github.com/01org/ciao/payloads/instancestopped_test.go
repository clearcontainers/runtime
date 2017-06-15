/*
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
*/

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestInstanceStoppedUnmarshal(t *testing.T) {
	var insStop EventInstanceStopped
	err := yaml.Unmarshal([]byte(testutil.InsStopYaml), &insStop)
	if err != nil {
		t.Error(err)
	}

	if insStop.InstanceStopped.InstanceUUID != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", insStop.InstanceStopped.InstanceUUID)
	}
}

func TestInstanceStoppedMarshal(t *testing.T) {
	var insStop EventInstanceStopped

	insStop.InstanceStopped.InstanceUUID = testutil.InstanceUUID

	y, err := yaml.Marshal(&insStop)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.InsStopYaml {
		t.Errorf("InstanceStopped marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.InsStopYaml)
	}
}
