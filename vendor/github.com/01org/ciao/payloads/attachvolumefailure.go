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

// AttachVolumeFailureReason denotes the underlying error that prevented
// an SSNTP AttachVolume command from attaching a volume to an instance.
type AttachVolumeFailureReason string

const (
	// AttachVolumeNoInstance indicates that a volume could not be attached
	// to an instance as the instance does not exist on the node to
	// which the AttachVolume command was sent.
	AttachVolumeNoInstance AttachVolumeFailureReason = "no_instance"

	// AttachVolumeInvalidPayload indicates that the payload of the SSNTP
	// AttachVolume command was corrupt and could not be unmarshalled.
	AttachVolumeInvalidPayload = "invalid_payload"

	// AttachVolumeInvalidData is returned by ciao-launcher if the contents
	// of the AttachVolume payload are incorrect, e.g., the instance_uuid
	// is missing.
	AttachVolumeInvalidData = "invalid_data"

	// AttachVolumeAttachFailure indicates that the attempt to attach a
	// volume to an instance failed.
	AttachVolumeAttachFailure = "attach_failure"

	// AttachVolumeAlreadyAttached indicates that the volume is already
	// attached to the instance.
	AttachVolumeAlreadyAttached = "already_attached"

	// AttachVolumeStateFailure indicates that launcher was unable to
	// update its internal state to register the new volume.
	AttachVolumeStateFailure = "state_failure"

	// AttachVolumeInstanceFailure indicates that the volume could not
	// be attached as the instance has failed to start and is being
	// deleted
	AttachVolumeInstanceFailure = "instance_failure"

	// AttachVolumeNotSupported indicates that the attach volume command
	// is not supported for the given workload type, e.g., a container.
	AttachVolumeNotSupported = "not_supported"
)

// ErrorAttachVolumeFailure represents the unmarshalled version of the contents of a
// SSNTP ERROR frame whose type is set to ssntp.AttachVolumeFailure.
type ErrorAttachVolumeFailure struct {
	// InstanceUUID is the UUID of the instance to which a volume could not be
	// attached.
	InstanceUUID string `yaml:"instance_uuid"`

	// VolumeUUID is the UUID of the volume that could not be attached.
	VolumeUUID string `yaml:"volume_uuid"`

	// Reason provides the reason for the attach failure, e.g.,
	// AttachVolumehNoInstance.
	Reason AttachVolumeFailureReason `yaml:"reason"`
}

func (r AttachVolumeFailureReason) String() string {
	switch r {
	case AttachVolumeNoInstance:
		return "Instance does not exist"
	case AttachVolumeInvalidPayload:
		return "YAML payload is corrupt"
	case AttachVolumeInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case AttachVolumeAttachFailure:
		return "Failed to attach volume to instance"
	case AttachVolumeAlreadyAttached:
		return "Volume already attached"
	case AttachVolumeStateFailure:
		return "State failure"
	case AttachVolumeInstanceFailure:
		return "Instance failure"
	case AttachVolumeNotSupported:
		return "Not Supported"
	}

	return ""
}
