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

func TestConcentratorAddedUnmarshal(t *testing.T) {
	var cnciAdded EventConcentratorInstanceAdded

	err := yaml.Unmarshal([]byte(testutil.CNCIAddedYaml), &cnciAdded)
	if err != nil {
		t.Error(err)
	}

	if cnciAdded.CNCIAdded.InstanceUUID != testutil.CNCIUUID {
		t.Errorf("Wrong instance UUID field [%s]", cnciAdded.CNCIAdded.InstanceUUID)
	}

	if cnciAdded.CNCIAdded.TenantUUID != testutil.TenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", cnciAdded.CNCIAdded.TenantUUID)
	}

	if cnciAdded.CNCIAdded.ConcentratorIP != testutil.CNCIIP {
		t.Errorf("Wrong CNCI IP field [%s]", cnciAdded.CNCIAdded.ConcentratorIP)
	}

	if cnciAdded.CNCIAdded.ConcentratorMAC != testutil.CNCIMAC {
		t.Errorf("Wrong CNCI MAC field [%s]", cnciAdded.CNCIAdded.ConcentratorMAC)
	}
}

func TestConcentratorAddedMarshal(t *testing.T) {
	var cnciAdded EventConcentratorInstanceAdded

	cnciAdded.CNCIAdded.InstanceUUID = testutil.CNCIUUID
	cnciAdded.CNCIAdded.TenantUUID = testutil.TenantUUID
	cnciAdded.CNCIAdded.ConcentratorIP = testutil.CNCIIP
	cnciAdded.CNCIAdded.ConcentratorMAC = testutil.CNCIMAC

	y, err := yaml.Marshal(&cnciAdded)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.CNCIAddedYaml {
		t.Errorf("ConcentratorInstanceAdded marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.CNCIAddedYaml)
	}
}
