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

// DeleteFailureReason denotes the underlying error that prevented
// an SSNTP DELETE command from deleting a running instance.
type DeleteFailureReason string

const (
	// DeleteNoInstance indicates that an instance could not be deleted
	// as it does not exist on the node to which the DELETE command was
	// sent.
	DeleteNoInstance DeleteFailureReason = "no_instance"

	// DeleteInvalidPayload indicates that the payload of the SSNTP
	// DELETE command was corrupt and could not be unmarshalled.
	DeleteInvalidPayload = "invalid_payload"

	// DeleteInvalidData is returned by ciao-launcher if the contents
	// of the DELETE payload are incorrect, e.g., the instance_uuid
	// is missing.
	DeleteInvalidData = "invalid_data"
)

// ErrorDeleteFailure represents the unmarshalled version of the contents of a
// SSNTP ERROR frame whose type is set to ssntp.DeleteFailure.
type ErrorDeleteFailure struct {
	// InstanceUUID is the UUID of the instance that could not be deleted.
	InstanceUUID string `yaml:"instance_uuid"`

	// Reason provides the reason for the delete failure, e.g.,
	// DeleteNoInstance.
	Reason DeleteFailureReason `yaml:"reason"`
}

func (r DeleteFailureReason) String() string {
	switch r {
	case DeleteNoInstance:
		return "Instance does not exist"
	case DeleteInvalidPayload:
		return "YAML payload is corrupt"
	case DeleteInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	}

	return ""
}
