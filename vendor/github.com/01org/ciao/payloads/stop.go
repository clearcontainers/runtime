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

// StopCmd contains the information needed to stop a running instance.
type StopCmd struct {
	// InstanceUUID is the UUID of the instance to stop
	InstanceUUID string `yaml:"instance_uuid"`

	// WorkloadAgentUUID identifies the node on which the instance is
	// running.  This information is needed by the scheduler to route
	// the command to the correct CN/NN.
	WorkloadAgentUUID string `yaml:"workload_agent_uuid"`
}

// Stop represents the unmarshalled version of the contents of a SSNTP STOP
// payload.  The structure contains enough information to stop a CN or NN
// instance.
type Stop struct {
	// Stop contains information about the instance to stop.
	Stop StopCmd `yaml:"stop"`
}

// Delete represents the unmarshalled version of the contents of a SSNTP DELETE
// payload.  The structure contains enough information to delete a CN or NN
// instance.
type Delete struct {
	// Delete contains information about the instance to delete.
	Delete StopCmd `yaml:"delete"`
}
