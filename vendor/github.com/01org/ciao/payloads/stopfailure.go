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

// StopFailureReason denotes the underlying error that prevented
// an SSNTP STOP command from stopping a running instance.
type StopFailureReason string

const (
	// StopNoInstance indicates that an instance could not be stopped
	// as it does not exist on the node to which the STOP command was
	// sent.
	StopNoInstance StopFailureReason = "no_instance"

	// StopInvalidPayload indicates that the payload of the SSNTP
	// STOP command was corrupt and could not be unmarshalled.
	StopInvalidPayload = "invalid_payload"

	// StopInvalidData is returned by ciao-launcher if the contents
	// of the STOP payload are incorrect, e.g., the instance_uuid
	// is missing.
	StopInvalidData = "invalid_data"

	// StopAlreadyStopped indicates that the instance does exist on the
	// node to which the STOP command was sent, but that it
	// is not currently running, e.g., it's status is either exited or
	// pending.
	StopAlreadyStopped = "already_stopped"
)

// ErrorStopFailure represents the unmarshalled version of the contents of a
// SSNTP ERROR frame whose type is set to ssntp.StopFailure.
type ErrorStopFailure struct {
	// InstanceUUID is the UUID of the instance that could not be stopped.
	InstanceUUID string `yaml:"instance_uuid"`

	// Reason provides the reason for the stop failure, e.g.,
	// StopAlreadyStopped.
	Reason StopFailureReason `yaml:"reason"`
}

func (r StopFailureReason) String() string {
	switch r {
	case StopNoInstance:
		return "Instance does not exist"
	case StopInvalidPayload:
		return "YAML payload is corrupt"
	case StopInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case StopAlreadyStopped:
		return "Instance has already shut down"
	}

	return ""
}
