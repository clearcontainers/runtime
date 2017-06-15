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

// NodeConnectedEvent contains information about a node that has either
// just connected or disconnected.
type NodeConnectedEvent struct {
	// SSNTP UUID of the agent running on that node.
	NodeUUID string `yaml:"node_uuid"`

	// The type of the node, e.g., NetworkNode or ComputeNode.
	NodeType Resource `yaml:"node_type"`
}

// NodeConnected represents the unmarshalled version of the contents of an
// SSNTP ssntp.NodeConnected event payload.   This event is sent by the
// scheduler to the controller to inform it that a node has just connected.
type NodeConnected struct {
	Connected NodeConnectedEvent `yaml:"node_connected"`
}

// NodeDisconnected represents the unmarshalled version of the contents of an
// SSNTP ssntp.NodeDisconnected event payload.   This event is sent by the
// scheduler to the controller to inform it that a node has just disconnected.
type NodeDisconnected struct {
	Disconnected NodeConnectedEvent `yaml:"node_disconnected"`
}
