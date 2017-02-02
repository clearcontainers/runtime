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

// DetachVolumeFailureReason denotes the underlying error that prevented
// an SSNTP DetachVolume command from detaching a volume from an instance.
type DetachVolumeFailureReason string

const (
	// DetachVolumeNoInstance indicates that a volume could not be detached
	// from an instance as the instance does not exist on the node to
	// which the DetachVolume command was sent.
	DetachVolumeNoInstance DetachVolumeFailureReason = "no_instance"

	// DetachVolumeInvalidPayload indicates that the payload of the SSNTP
	// DetachVolume command was corrupt and could not be unmarshalled.
	DetachVolumeInvalidPayload = "invalid_payload"

	// DetachVolumeInvalidData is returned by ciao-launcher if the contents
	// of the DetachVolume payload are incorrect, e.g., the instance_uuid
	// is missing.
	DetachVolumeInvalidData = "invalid_data"

	// DetachVolumeDetachFailure indicates that the attempt to detach a
	// volume from an instance failed.
	DetachVolumeDetachFailure = "detach_failure"

	// DetachVolumeNotAttached indicates that the volume is not
	// attached to the instance.
	DetachVolumeNotAttached = "not_attached"

	// DetachVolumeStateFailure indicates that launcher was unable to
	// update its internal state to remove the volume.
	DetachVolumeStateFailure = "state_failure"

	// DetachVolumeInstanceFailure indicates that the volume could not
	// be detached as the instance has failed to start and is being
	// deleted
	DetachVolumeInstanceFailure = "instance_failure"

	// DetachVolumeNotSupported indicates that the detach volume command
	// is not supported for the given workload type, e.g., a container.
	DetachVolumeNotSupported = "not_supported"
)

// ErrorDetachVolumeFailure represents the unmarshalled version of the contents of a
// SSNTP ERROR frame whose type is set to ssntp.DetachVolumeFailure.
type ErrorDetachVolumeFailure struct {
	// InstanceUUID is the UUID of the instance from which a volume could not be
	// detached.
	InstanceUUID string `yaml:"instance_uuid"`

	// VolumeUUID is the UUID of the volume that could not be detached.
	VolumeUUID string `yaml:"volume_uuid"`

	// Reason provides the reason for the detach failure, e.g.,
	// DetachVolumeNoInstance.
	Reason DetachVolumeFailureReason `yaml:"reason"`
}

func (r DetachVolumeFailureReason) String() string {
	switch r {
	case DetachVolumeNoInstance:
		return "Instance does not exist"
	case DetachVolumeInvalidPayload:
		return "YAML payload is corrupt"
	case DetachVolumeInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case DetachVolumeDetachFailure:
		return "Failed to detach volume from instance"
	case DetachVolumeNotAttached:
		return "Volume not attached"
	case DetachVolumeStateFailure:
		return "State failure"
	case DetachVolumeInstanceFailure:
		return "Instance failure"
	case DetachVolumeNotSupported:
		return "Not Supported"
	}

	return ""
}
