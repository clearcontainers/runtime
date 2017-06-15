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

func TestInstanceDeletedUnmarshal(t *testing.T) {
	var insDel EventInstanceDeleted
	err := yaml.Unmarshal([]byte(testutil.InsDelYaml), &insDel)
	if err != nil {
		t.Error(err)
	}

	if insDel.InstanceDeleted.InstanceUUID != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", insDel.InstanceDeleted.InstanceUUID)
	}
}

func TestInstanceDeletedMarshal(t *testing.T) {
	var insDel EventInstanceDeleted

	insDel.InstanceDeleted.InstanceUUID = testutil.InstanceUUID

	y, err := yaml.Marshal(&insDel)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.InsDelYaml {
		t.Errorf("InstanceDeleted marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.InsDelYaml)
	}
}
