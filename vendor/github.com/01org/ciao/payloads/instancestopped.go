/*
// Copyright (c) 2017 Intel Corporation
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

// InstanceStoppedEvent contains the UUID of an instance that has just been
// deleted from a node for the purposes of migration.
type InstanceStoppedEvent struct {
	InstanceUUID string `yaml:"instance_uuid"`
}

// EventInstanceStopped represents the unmarshalled version of the contents of
// an SSNTP ssntp.InstanceStopped event. This event is sent by ciao-launcher
// when it successfully deletes an instance that is being migrated.
type EventInstanceStopped struct {
	InstanceStopped InstanceStoppedEvent `yaml:"instance_stopped"`
}
