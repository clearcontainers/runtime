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
	"strconv"
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestTenantAddedUnmarshal(t *testing.T) {
	var tenantAdded EventTenantAdded

	err := yaml.Unmarshal([]byte(testutil.TenantAddedYaml), &tenantAdded)
	if err != nil {
		t.Error(err)
	}

	if tenantAdded.TenantAdded.AgentUUID != testutil.AgentUUID {
		t.Errorf("Wrong agent UUID field [%s]", tenantAdded.TenantAdded.AgentUUID)
	}

	if tenantAdded.TenantAdded.AgentIP != testutil.AgentIP {
		t.Errorf("Wrong agent IP field [%s]", tenantAdded.TenantAdded.AgentIP)
	}

	if tenantAdded.TenantAdded.TenantUUID != testutil.TenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", tenantAdded.TenantAdded.TenantUUID)
	}

	if tenantAdded.TenantAdded.TenantSubnet != testutil.TenantSubnet {
		t.Errorf("Wrong tenant subnet field [%s]", tenantAdded.TenantAdded.TenantSubnet)
	}

	if tenantAdded.TenantAdded.ConcentratorUUID != testutil.CNCIUUID {
		t.Errorf("Wrong CNCI UUID field [%s]", tenantAdded.TenantAdded.ConcentratorUUID)
	}

	if tenantAdded.TenantAdded.ConcentratorIP != testutil.CNCIIP {
		t.Errorf("Wrong CNCI IP field [%s]", tenantAdded.TenantAdded.ConcentratorIP)
	}

}

func TestTenantRemovedUnmarshal(t *testing.T) {
	var tenantRemoved EventTenantRemoved

	err := yaml.Unmarshal([]byte(testutil.TenantRemovedYaml), &tenantRemoved)
	if err != nil {
		t.Error(err)
	}

	if tenantRemoved.TenantRemoved.AgentUUID != testutil.AgentUUID {
		t.Errorf("Wrong agent UUID field [%s]", tenantRemoved.TenantRemoved.AgentUUID)
	}

	if tenantRemoved.TenantRemoved.AgentIP != testutil.AgentIP {
		t.Errorf("Wrong agent IP field [%s]", tenantRemoved.TenantRemoved.AgentIP)
	}

	if tenantRemoved.TenantRemoved.TenantUUID != testutil.TenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", tenantRemoved.TenantRemoved.TenantUUID)
	}

	if tenantRemoved.TenantRemoved.TenantSubnet != testutil.TenantSubnet {
		t.Errorf("Wrong tenant subnet field [%s]", tenantRemoved.TenantRemoved.TenantSubnet)
	}

	if tenantRemoved.TenantRemoved.ConcentratorUUID != testutil.CNCIUUID {
		t.Errorf("Wrong CNCI UUID field [%s]", tenantRemoved.TenantRemoved.ConcentratorUUID)
	}

	if tenantRemoved.TenantRemoved.ConcentratorIP != testutil.CNCIIP {
		t.Errorf("Wrong CNCI IP field [%s]", tenantRemoved.TenantRemoved.ConcentratorIP)
	}

}

func TestTenantAddedMarshal(t *testing.T) {
	var tenantAdded EventTenantAdded

	tenantAdded.TenantAdded.AgentUUID = testutil.AgentUUID
	tenantAdded.TenantAdded.AgentIP = testutil.AgentIP
	tenantAdded.TenantAdded.TenantUUID = testutil.TenantUUID
	tenantAdded.TenantAdded.TenantSubnet = testutil.TenantSubnet
	tenantAdded.TenantAdded.ConcentratorUUID = testutil.CNCIUUID
	tenantAdded.TenantAdded.ConcentratorIP = testutil.CNCIIP
	k, _ := strconv.Atoi(testutil.SubnetKey)
	tenantAdded.TenantAdded.SubnetKey = k

	y, err := yaml.Marshal(&tenantAdded)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.TenantAddedYaml {
		t.Errorf("TenantAdded marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.TenantAddedYaml)
	}
}

func TestTenantRemovedMarshal(t *testing.T) {
	var tenantRemoved EventTenantRemoved

	tenantRemoved.TenantRemoved.AgentUUID = testutil.AgentUUID
	tenantRemoved.TenantRemoved.AgentIP = testutil.AgentIP
	tenantRemoved.TenantRemoved.TenantUUID = testutil.TenantUUID
	tenantRemoved.TenantRemoved.TenantSubnet = testutil.TenantSubnet
	tenantRemoved.TenantRemoved.ConcentratorUUID = testutil.CNCIUUID
	tenantRemoved.TenantRemoved.ConcentratorIP = testutil.CNCIIP
	k, _ := strconv.Atoi(testutil.SubnetKey)
	tenantRemoved.TenantRemoved.SubnetKey = k

	y, err := yaml.Marshal(&tenantRemoved)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.TenantRemovedYaml {
		t.Errorf("TenantRemoved marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.TenantRemovedYaml)
	}
}
