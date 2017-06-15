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

package payloads

// VolumeCmd contains all the information needed to attach a volume
// to or detach a volume from an existing instance.
type VolumeCmd struct {
	// InstanceUUID is the UUID of the instance to which the volume is to be
	// attached.
	InstanceUUID string `yaml:"instance_uuid"`

	// VolumeUUID is the UUID of the volume to attach.
	VolumeUUID string `yaml:"volume_uuid"`

	// WorkloadAgentUUID identifies the node on which the instance is
	// running.  This information is needed by the scheduler to route
	// the command to the correct CN/NN.
	WorkloadAgentUUID string `yaml:"workload_agent_uuid"`
}

// AttachVolume represents the unmarshalled version of the contents of a SSNTP
// AttachVolume payload.  The structure contains enough information to attach a
// volume to an existing instance.
type AttachVolume struct {
	Attach VolumeCmd `yaml:"attach_volume"`
}

// DetachVolume represents the unmarshalled version of the contents of a SSNTP
// DetachVolume payload.  The structure contains enough information to detach a
// volume from an existing instance.
type DetachVolume struct {
	Detach VolumeCmd `yaml:"detach_volume"`
}
