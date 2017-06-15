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

func TestStartUnmarshal(t *testing.T) {
	var cmd Start
	err := yaml.Unmarshal([]byte(testutil.StartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestStartMarshal(t *testing.T) {
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

	var cmd Start
	cmd.Start.TenantUUID = testutil.TenantUUID
	cmd.Start.InstanceUUID = testutil.InstanceUUID
	cmd.Start.ImageUUID = testutil.ImageUUID
	cmd.Start.DockerImage = testutil.DockerImage
	cmd.Start.RequestedResources = append(cmd.Start.RequestedResources, reqVcpus)
	cmd.Start.RequestedResources = append(cmd.Start.RequestedResources, reqMem)
	cmd.Start.EstimatedResources = append(cmd.Start.EstimatedResources, estVcpus)
	cmd.Start.EstimatedResources = append(cmd.Start.EstimatedResources, estMem)
	cmd.Start.FWType = EFI
	cmd.Start.InstancePersistence = Host
	cmd.Start.VMType = QEMU

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.StartYaml {
		t.Errorf("Start marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.StartYaml)
	}
}

// make sure the yaml can be unmarshaled into the Start struct with
// optional data not present
func TestStartUnmarshalPartial(t *testing.T) {
	var cmd Start
	err := yaml.Unmarshal([]byte(testutil.PartialStartYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	var expectedCmd Start
	expectedCmd.Start.InstanceUUID = testutil.InstanceUUID
	expectedCmd.Start.ImageUUID = testutil.ImageUUID
	expectedCmd.Start.DockerImage = testutil.DockerImage
	expectedCmd.Start.FWType = EFI
	expectedCmd.Start.InstancePersistence = Host
	expectedCmd.Start.VMType = QEMU
	vcpus := RequestedResource{
		Type:      "vcpus",
		Value:     2,
		Mandatory: true,
	}
	expectedCmd.Start.RequestedResources = append(expectedCmd.Start.RequestedResources, vcpus)

	if cmd.Start.InstanceUUID != expectedCmd.Start.InstanceUUID ||
		cmd.Start.ImageUUID != expectedCmd.Start.ImageUUID ||
		cmd.Start.DockerImage != expectedCmd.Start.DockerImage ||
		cmd.Start.FWType != expectedCmd.Start.FWType ||
		cmd.Start.InstancePersistence != expectedCmd.Start.InstancePersistence ||
		cmd.Start.VMType != expectedCmd.Start.VMType ||
		len(cmd.Start.RequestedResources) != 1 ||
		len(expectedCmd.Start.RequestedResources) != 1 ||
		cmd.Start.RequestedResources[0].Type != expectedCmd.Start.RequestedResources[0].Type ||
		cmd.Start.RequestedResources[0].Value != expectedCmd.Start.RequestedResources[0].Value ||
		cmd.Start.RequestedResources[0].Mandatory != expectedCmd.Start.RequestedResources[0].Mandatory {
		t.Error("Unexpected values in Start")
	}
}
