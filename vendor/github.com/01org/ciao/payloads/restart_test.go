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

func TestRestartUnmarshal(t *testing.T) {
	var cmd Restart
	err := yaml.Unmarshal([]byte(testutil.RestartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestRestartMarshal(t *testing.T) {
	reqVcpus := RequestedResource{
		Type:      "vcpus",
		Value:     2,
		Mandatory: true,
	}
	reqMem := RequestedResource{
		Type:      "mem_mb",
		Value:     4096,
		Mandatory: true,
	}
	estVcpus := EstimatedResource{
		Type:  "vcpus",
		Value: 1,
	}
	estMem := EstimatedResource{
		Type:  "mem_mb",
		Value: 128,
	}

	var cmd Restart
	cmd.Restart.TenantUUID = testutil.TenantUUID
	cmd.Restart.InstanceUUID = testutil.InstanceUUID
	cmd.Restart.ImageUUID = testutil.ImageUUID
	cmd.Restart.WorkloadAgentUUID = testutil.AgentUUID
	cmd.Restart.RequestedResources = append(cmd.Restart.RequestedResources, reqVcpus)
	cmd.Restart.RequestedResources = append(cmd.Restart.RequestedResources, reqMem)
	cmd.Restart.EstimatedResources = append(cmd.Restart.EstimatedResources, estVcpus)
	cmd.Restart.EstimatedResources = append(cmd.Restart.EstimatedResources, estMem)
	cmd.Restart.FWType = EFI
	cmd.Restart.InstancePersistence = Host

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.RestartYaml {
		t.Errorf("Restart marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.RestartYaml)
	}
}

// make sure the yaml can be unmarshaled into the Restart struct with
// optional data not present
func TestRestartUnmarshalPartial(t *testing.T) {
	var cmd Restart
	err := yaml.Unmarshal([]byte(testutil.PartialRestartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	var expectedCmd Restart
	expectedCmd.Restart.InstanceUUID = testutil.InstanceUUID
	expectedCmd.Restart.WorkloadAgentUUID = testutil.AgentUUID
	expectedCmd.Restart.FWType = EFI
	expectedCmd.Restart.InstancePersistence = Host
	vcpus := RequestedResource{
		Type:      "vcpus",
		Value:     2,
		Mandatory: true,
	}
	expectedCmd.Restart.RequestedResources = append(expectedCmd.Restart.RequestedResources, vcpus)

	if cmd.Restart.InstanceUUID != expectedCmd.Restart.InstanceUUID ||
		cmd.Restart.WorkloadAgentUUID != expectedCmd.Restart.WorkloadAgentUUID ||
		cmd.Restart.FWType != expectedCmd.Restart.FWType ||
		cmd.Restart.InstancePersistence != expectedCmd.Restart.InstancePersistence ||
		len(cmd.Restart.RequestedResources) != 1 ||
		len(expectedCmd.Restart.RequestedResources) != 1 ||
		cmd.Restart.RequestedResources[0].Type != expectedCmd.Restart.RequestedResources[0].Type ||
		cmd.Restart.RequestedResources[0].Value != expectedCmd.Restart.RequestedResources[0].Value ||
		cmd.Restart.RequestedResources[0].Mandatory != expectedCmd.Restart.RequestedResources[0].Mandatory {
		t.Error("Unexpected values in Restart")
	}
}
