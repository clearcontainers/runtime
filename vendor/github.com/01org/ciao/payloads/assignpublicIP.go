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

// PublicIPCommand contains information about a IP and its associated data.
type PublicIPCommand struct {
	ConcentratorUUID string `yaml:"concentrator_uuid"`
	TenantUUID       string `yaml:"tenant_uuid"`
	InstanceUUID     string `yaml:"instance_uuid"`
	PublicIP         string `yaml:"public_ip"`
	PrivateIP        string `yaml:"private_ip"`
	VnicMAC          string `yaml:"vnic_mac"`
}

// CommandAssignPublicIP is a wrapper around PublicIPCommand. It is the
// AssignPublicIP command payload.
type CommandAssignPublicIP struct {
	AssignIP PublicIPCommand `yaml:"assign_public_ip"`
}

// CommandReleasePublicIP is a wrapper around PublicIPCommand. It is the
// ReleasePublicIP command payload.
type CommandReleasePublicIP struct {
	ReleaseIP PublicIPCommand `yaml:"release_public_ip"`
}

// PublicIPFailureReason represents the potential
// AssignPublicIP/ReleasePublicIP commands failure reasons.
type PublicIPFailureReason string

const (
	// PublicIPNoInstance indicates that a public IP assignment/release failed
	// because the corresponding instance does not exist
	PublicIPNoInstance PublicIPFailureReason = "no_instance"

	// PublicIPInvalidPayload constant is used to denote when YAML payload
	//  is corrupt.
	PublicIPInvalidPayload = "invalid_payload"

	// PublicIPInvalidData constant is used to denote when command section
	// of YAML payload is corrupt or missing required information.
	PublicIPInvalidData = "invalid_data"

	// PublicIPAssignFailure constant is used to denote when Public IP assignment
	// operation failed.
	PublicIPAssignFailure = "assign_failure"

	// PublicIPReleaseFailure constant is used to denote when Public IP release
	// operation failed.
	PublicIPReleaseFailure = "release_failure"
)

// ErrorPublicIPFailure represents the PublicIPFailure SSNTP error payload.
// It includes information about the IP itself and the actual reason for failure.
type ErrorPublicIPFailure struct {
	ConcentratorUUID string                `yaml:"concentrator_uuid"`
	TenantUUID       string                `yaml:"tenant_uuid"`
	InstanceUUID     string                `yaml:"instance_uuid"`
	PublicIP         string                `yaml:"public_ip"`
	PrivateIP        string                `yaml:"private_ip"`
	VnicMAC          string                `yaml:"vnic_mac"`
	Reason           PublicIPFailureReason `yaml:"reason"`
}

func (r PublicIPFailureReason) String() string {
	switch r {
	case PublicIPNoInstance:
		return "Instance does not exist"
	case PublicIPInvalidPayload:
		return "YAML payload is corrupt"
	case PublicIPInvalidData:
		return "Command section of YAML payload is corrupt or missing required information"
	case PublicIPAssignFailure:
		return "Public IP assignment operation_failed"
	case PublicIPReleaseFailure:
		return "Public IP release operation_failed"
	}
	return ""
}
