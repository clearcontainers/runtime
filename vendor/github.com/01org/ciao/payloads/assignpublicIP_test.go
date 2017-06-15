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
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestAssignPublicIPUnmarshal(t *testing.T) {
	var assignIP CommandAssignPublicIP

	err := yaml.Unmarshal([]byte(testutil.AssignIPYaml), &assignIP)
	if err != nil {
		t.Error(err)
	}

	if assignIP.AssignIP.ConcentratorUUID != testutil.CNCIUUID {
		t.Errorf("Wrong concentrator UUID field [%s]", assignIP.AssignIP.ConcentratorUUID)
	}

	if assignIP.AssignIP.TenantUUID != testutil.TenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", assignIP.AssignIP.TenantUUID)
	}

	if assignIP.AssignIP.InstanceUUID != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", assignIP.AssignIP.InstanceUUID)
	}

	if assignIP.AssignIP.PublicIP != testutil.InstancePublicIP {
		t.Errorf("Wrong public IP field [%s]", assignIP.AssignIP.PublicIP)
	}

	if assignIP.AssignIP.PrivateIP != testutil.InstancePrivateIP {
		t.Errorf("Wrong private IP field [%s]", assignIP.AssignIP.PrivateIP)
	}

	if assignIP.AssignIP.VnicMAC != testutil.VNICMAC {
		t.Errorf("Wrong VNIC MAC field [%s]", assignIP.AssignIP.VnicMAC)
	}
}

func TestReleasePublicIPUnmarshal(t *testing.T) {
	var releaseIP CommandReleasePublicIP

	err := yaml.Unmarshal([]byte(testutil.ReleaseIPYaml), &releaseIP)
	if err != nil {
		t.Error(err)
	}

	if releaseIP.ReleaseIP.ConcentratorUUID != testutil.CNCIUUID {
		t.Errorf("Wrong concentrator UUID field [%s]", releaseIP.ReleaseIP.ConcentratorUUID)
	}

	if releaseIP.ReleaseIP.TenantUUID != testutil.TenantUUID {
		t.Errorf("Wrong tenant UUID field [%s]", releaseIP.ReleaseIP.TenantUUID)
	}

	if releaseIP.ReleaseIP.InstanceUUID != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID field [%s]", releaseIP.ReleaseIP.InstanceUUID)
	}

	if releaseIP.ReleaseIP.PublicIP != testutil.InstancePublicIP {
		t.Errorf("Wrong public IP field [%s]", releaseIP.ReleaseIP.PublicIP)
	}

	if releaseIP.ReleaseIP.PrivateIP != testutil.InstancePrivateIP {
		t.Errorf("Wrong private IP field [%s]", releaseIP.ReleaseIP.PrivateIP)
	}

	if releaseIP.ReleaseIP.VnicMAC != testutil.VNICMAC {
		t.Errorf("Wrong VNIC MAC field [%s]", releaseIP.ReleaseIP.VnicMAC)
	}
}

func TestAssignPublicIPMarshal(t *testing.T) {
	var assignIP CommandAssignPublicIP

	assignIP.AssignIP.ConcentratorUUID = testutil.CNCIUUID
	assignIP.AssignIP.TenantUUID = testutil.TenantUUID
	assignIP.AssignIP.InstanceUUID = testutil.InstanceUUID
	assignIP.AssignIP.PublicIP = testutil.InstancePublicIP
	assignIP.AssignIP.PrivateIP = testutil.InstancePrivateIP
	assignIP.AssignIP.VnicMAC = testutil.VNICMAC

	y, err := yaml.Marshal(&assignIP)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.AssignIPYaml {
		t.Errorf("AssignPublicIP marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.AssignIPYaml)
	}
}

func TestReleasePublicIPMarshal(t *testing.T) {
	var releaseIP CommandReleasePublicIP

	releaseIP.ReleaseIP.ConcentratorUUID = testutil.CNCIUUID
	releaseIP.ReleaseIP.TenantUUID = testutil.TenantUUID
	releaseIP.ReleaseIP.InstanceUUID = testutil.InstanceUUID
	releaseIP.ReleaseIP.PublicIP = testutil.InstancePublicIP
	releaseIP.ReleaseIP.PrivateIP = testutil.InstancePrivateIP
	releaseIP.ReleaseIP.VnicMAC = testutil.VNICMAC

	y, err := yaml.Marshal(&releaseIP)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.ReleaseIPYaml {
		t.Errorf("ReleasePublicIP marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.ReleaseIPYaml)
	}
}

func TestPublicIPFailureString(t *testing.T) {
	var stringTests = []struct {
		r        PublicIPFailureReason
		expected string
	}{
		{PublicIPNoInstance, "Instance does not exist"},
		{PublicIPInvalidPayload, "YAML payload is corrupt"},
		{PublicIPInvalidData, "Command section of YAML payload is corrupt or missing required information"},
		{PublicIPAssignFailure, "Public IP assignment operation_failed"},
		{PublicIPReleaseFailure, "Public IP release operation_failed"},
	}
	error := ErrorPublicIPFailure{
		ConcentratorUUID: uuid.Generate().String(),
		TenantUUID:       uuid.Generate().String(),
		InstanceUUID:     uuid.Generate().String(),
		PublicIP:         "10.1.2.3",
		PrivateIP:        "192.168.1.2",
		VnicMAC:          "aa:bb:cc:01:02:03",
	}
	for _, test := range stringTests {
		error.Reason = test.r
		s := error.Reason.String()
		if s != test.expected {
			t.Errorf("expected \"%s\", got \"%s\"", test.expected, s)
		}
	}
}
