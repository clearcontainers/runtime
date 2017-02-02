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

// TenantAddedEvent is populated by ciao-launcher whenever it creates
// or removes a local tunnel for a tenant on a CN.  This information is
// sent to a CNCI instance, via the scheduler.  The cnci-agent then does
// its magic.
type TenantAddedEvent struct {
	// The UUID of the ciao-launcher that generated the event.
	AgentUUID string `yaml:"agent_uuid"`

	// The IP address of the CN on which the originating agent runs.
	AgentIP string `yaml:"agent_ip"`

	// The UUID of the tenant.
	TenantUUID string `yaml:"tenant_uuid"`

	// The subnet of the Tenant.
	TenantSubnet string `yaml:"tenant_subnet"`

	// The UUID of the concentrator.
	ConcentratorUUID string `yaml:"concentrator_uuid"`

	// The IP address of the concentrator.
	ConcentratorIP string `yaml:"concentrator_ip"`

	// The UUID of the subnet.
	SubnetKey int `yaml:"subnet_key"`
}

// EventTenantAdded represents the unmarshalled version of the contents of an
// SSNTP ssntp.TenantAdded event payload. The structure contains all the
// information needed by an CNCI instance to add a remote tunnel for a
// CN
type EventTenantAdded struct {
	TenantAdded TenantAddedEvent `yaml:"tenant_added"`
}

// EventTenantRemoved represents the unmarshalled version of the contents of an
// SSNTP ssntp.TenantRemoved event payload. The structure contains all the
// information needed by an CNCI instance to remove a remote tunnel for a
// CN
type EventTenantRemoved struct {
	TenantRemoved TenantAddedEvent `yaml:"tenant_removed"`
}
