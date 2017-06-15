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

// ConcentratorInstanceAddedEvent contains information about a newly added
// CNCI instance.
type ConcentratorInstanceAddedEvent struct {
	InstanceUUID    string `yaml:"instance_uuid"`
	TenantUUID      string `yaml:"tenant_uuid"`
	ConcentratorIP  string `yaml:"concentrator_ip"`
	ConcentratorMAC string `yaml:"concentrator_mac"`
}

// EventConcentratorInstanceAdded represents the unmarshalled version of the
// contents of an SSNTP ssntp.ConcentratorInstanceAdded event.  This event is
// sent by the cnci-agent to the controller when it connects to the scheduler.
type EventConcentratorInstanceAdded struct {
	CNCIAdded ConcentratorInstanceAddedEvent `yaml:"concentrator_instance_added"`
}
