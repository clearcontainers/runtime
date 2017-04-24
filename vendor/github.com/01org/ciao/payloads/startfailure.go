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

// StartFailureReason denotes the underlying error that prevented
// an SSNTP START command from launching a new instance on a CN
// or a NN.  Most, but not all, of these errors are returned by
// ciao-launcher
type StartFailureReason string

const (
	// FullCloud is returned by the scheduler when all nodes in the cluster
	// are FULL and it is unable to satisfy a START request.
	FullCloud StartFailureReason = "full_cloud"

	// FullComputeNode indicates that the node to which the START command
	// was sent had insufficient resources to start the requested instance.
	FullComputeNode = "full_cn"

	// NoComputeNodes is returned by the scheduler if no compute nodes are
	// running in the cluster upon which the instance can be started.
	NoComputeNodes = "no_cn"

	// NoNetworkNodes is returned by the scheduler if no network nodes are
	// running in the cluster upon which the instance can be started.
	NoNetworkNodes = "no_net_cn"

	// InvalidPayload indicates that the contents of the START payload are
	// corrupt
	InvalidPayload = "invalid_payload"

	// InvalidData indicates that the start section of the payload is
	// corrupt or missing information such as image-id
	InvalidData = "invalid_data"

	// AlreadyRunning is returned when an attempt is made to start an
	// instance on a node upon which that very same instance is already
	// running.
	AlreadyRunning = "already_running"

	// InstanceExists is returned when an attempt is made to start an
	// instance on a node upon which that very same instance already
	// exists but is not currently running.
	InstanceExists = "instance_exists"

	// ImageFailure indicates that ciao-launcher is unable to prepare
	// the rootfs for the instance, e.g., the image_uuid refers to an
	// non-existent backing image
	ImageFailure = "image_failure"

	// LaunchFailure indicates that the instance has been successfully
	// created but could not be launched.  Actually, this is sort of an
	// odd situation as the START command partially succeeded.
	// ciao-launcher returns an error code, but the instance has been
	// created and could be booted a later stage via the RESTART command.
	LaunchFailure = "launch_failure"

	// NetworkFailure indicates that it was not possible to initialise
	// networking for the instance.
	NetworkFailure = "network_failure"
)

// ErrorStartFailure represents the unmarshalled version of the contents of a
// SSNTP ERROR frame whose type is set to ssntp.StartFailure.
type ErrorStartFailure struct {
	// InstanceUUID is the UUID of the instance that could not be started.
	InstanceUUID string `yaml:"instance_uuid"`

	// Reason provides the reason for the start failure, e.g.,
	// LaunchFailure.
	Reason StartFailureReason `yaml:"reason"`
}

func (r StartFailureReason) String() string {
	switch r {
	case FullCloud:
		return "Cloud is full"
	case FullComputeNode:
		return "Compute node is full"
	case NoComputeNodes:
		return "No compute node available"
	case NoNetworkNodes:
		return "No network node available"
	case InvalidPayload:
		return "YAML payload is corrupt"
	case InvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case AlreadyRunning:
		return "Instance is already running"
	case InstanceExists:
		return "Instance already exists"
	case ImageFailure:
		return "Failed to create instance image"
	case LaunchFailure:
		return "Failed to launch instance"
	case NetworkFailure:
		return "Failed to create VNIC for instance"
	}

	return ""
}
