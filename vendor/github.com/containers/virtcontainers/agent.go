//
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
//

package virtcontainers

import (
	"fmt"
	"syscall"

	"github.com/mitchellh/mapstructure"
)

// AgentType describes the type of guest agent a Pod should run.
type AgentType string

const (
	// NoopAgentType is the No-Op agent.
	NoopAgentType AgentType = "noop"

	// SSHdAgent is the SSH daemon agent.
	SSHdAgent AgentType = "sshd"

	// HyperstartAgent is the Hyper hyperstart agent.
	HyperstartAgent AgentType = "hyperstart"
)

// Set sets an agent type based on the input string.
func (agentType *AgentType) Set(value string) error {
	switch value {
	case "noop":
		*agentType = NoopAgentType
		return nil
	case "sshd":
		*agentType = SSHdAgent
		return nil
	case "hyperstart":
		*agentType = HyperstartAgent
		return nil
	default:
		return fmt.Errorf("Unknown agent type %s", value)
	}
}

// String converts an agent type to a string.
func (agentType *AgentType) String() string {
	switch *agentType {
	case NoopAgentType:
		return string(NoopAgentType)
	case SSHdAgent:
		return string(SSHdAgent)
	case HyperstartAgent:
		return string(HyperstartAgent)
	default:
		return ""
	}
}

// newAgent returns an agent from an agent type.
func newAgent(agentType AgentType) agent {
	switch agentType {
	case NoopAgentType:
		return &noopAgent{}
	case SSHdAgent:
		return &sshd{}
	case HyperstartAgent:
		return &hyper{}
	default:
		return &noopAgent{}
	}
}

// newAgentConfig returns an agent config from a generic PodConfig interface.
func newAgentConfig(config PodConfig) interface{} {
	switch config.AgentType {
	case NoopAgentType:
		return nil
	case SSHdAgent:
		var sshdConfig SshdConfig
		err := mapstructure.Decode(config.AgentConfig, &sshdConfig)
		if err != nil {
			return err
		}
		return sshdConfig
	case HyperstartAgent:
		var hyperConfig HyperConfig
		err := mapstructure.Decode(config.AgentConfig, &hyperConfig)
		if err != nil {
			return err
		}
		return hyperConfig
	default:
		return nil
	}
}

// agent is the virtcontainers agent interface.
// Agents are running in the guest VM and handling
// communications between the host and guest.
type agent interface {
	// init is used to pass agent specific configuration to the agent implementation.
	// agent implementations also will typically start listening for agent events from
	// init().
	// After init() is called, agent implementations should be initialized and ready
	// to handle all other Agent interface methods.
	init(pod *Pod, config interface{}) error

	// capabilities should return a structure that specifies the capabilities
	// supported by the agent.
	capabilities() capabilities

	// exec will tell the agent to run a command in an already running container.
	exec(pod *Pod, c Container, process Process, cmd Cmd) error

	// startPod will tell the agent to start all containers related to the Pod.
	startPod(pod Pod) error

	// stopPod will tell the agent to stop all containers related to the Pod.
	stopPod(pod Pod) error

	// createContainer will tell the agent to create a container related to a Pod.
	createContainer(pod *Pod, c *Container) error

	// startContainer will tell the agent to start a container related to a Pod.
	startContainer(pod Pod, c Container) error

	// stopContainer will tell the agent to stop a container related to a Pod.
	stopContainer(pod Pod, c Container) error

	// killContainer will tell the agent to send a signal to a
	// container related to a Pod. If all is true, all processes in
	// the container will be sent the signal.
	killContainer(pod Pod, c Container, signal syscall.Signal, all bool) error
}
