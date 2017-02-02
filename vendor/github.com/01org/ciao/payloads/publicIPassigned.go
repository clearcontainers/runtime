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

// PublicIPEvent contains the basic information of Public IP event.
type PublicIPEvent struct {
	ConcentratorUUID string `yaml:"concentrator_uuid"`
	InstanceUUID     string `yaml:"instance_uuid"`
	PublicIP         string `yaml:"public_ip"`
	PrivateIP        string `yaml:"private_ip"`
}

// EventPublicIPAssigned represents the SSNTP PublicIPAssigned event payload.
type EventPublicIPAssigned struct {
	AssignedIP PublicIPEvent `yaml:"public_ip_assigned"`
}

// EventPublicIPUnassigned represents the SSNTP PublicIPUnassigned event payload.
type EventPublicIPUnassigned struct {
	UnassignedIP PublicIPEvent `yaml:"public_ip_unassigned"`
}
