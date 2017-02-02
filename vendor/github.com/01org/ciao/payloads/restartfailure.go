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

// RestartFailureReason denotes the underlying error that prevented
// an SSNTP RESTART command from booting up an existing instance on a CN
// or a NN.
type RestartFailureReason string

const (
	// RestartNoInstance indicates that an instance could not be restarted
	// as it does not exist on the node to which the RESTART command was
	// sent.
	RestartNoInstance RestartFailureReason = "no_instance"

	// RestartInvalidPayload indicates that the payload of the SSNTP
	// RESTART command was corrupt and could not be unmarshalled.
	RestartInvalidPayload = "invalid_payload"

	// RestartInvalidData is returned by ciao-launcher if the contents
	// of the RESTART payload are incorrect, e.g., the instance_uuid
	// is missing.
	RestartInvalidData = "invalid_data"

	// RestartAlreadyRunning indicates that an attempt was made to restart
	// a running instance.
	RestartAlreadyRunning = "already_running"

	// RestartInstanceCorrupt indicates that it was impossible to restart
	// an instance as the state of that instance stored on the node has
	// become corrupted.
	RestartInstanceCorrupt = "instance_corrupt"

	// RestartLaunchFailure indicates that it was not possible to restart
	// the instance, for example, the call to docker start fails.
	RestartLaunchFailure = "launch_failure"

	// RestartNetworkFailure indicates that it was not possible to
	// initialise networking for the instance before restarting it.
	RestartNetworkFailure = "network_failure"
)

// ErrorRestartFailure represents the unmarshalled version of the contents of a
// SSNTP ERROR frame whose type is set to ssntp.RestartFailure.
type ErrorRestartFailure struct {
	// InstanceUUID is the UUID of the instance that could not be started.
	InstanceUUID string `yaml:"instance_uuid"`

	// Reason provides the reason for the restart failure, e.g.,
	// RestartLaunchFailure.
	Reason RestartFailureReason `yaml:"reason"`
}

func (r RestartFailureReason) String() string {
	switch r {
	case RestartNoInstance:
		return "Instance does not exist"
	case RestartInvalidPayload:
		return "YAML payload is corrupt"
	case RestartInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case RestartAlreadyRunning:
		return "Instance is already running"
	case RestartInstanceCorrupt:
		return "Instance is corrupt"
	case RestartLaunchFailure:
		return "Failed to launch instance"
	case RestartNetworkFailure:
		return "Failed to locate VNIC for instance"
	}

	return ""
}
