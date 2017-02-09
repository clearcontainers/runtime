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
	"os"
	"path/filepath"
	"syscall"

	"github.com/01org/ciao/ssntp/uuid"
	"github.com/golang/glog"
)

// Process gathers data related to a container process.
type Process struct {
	Token string
	Pid   int
}

// ContainerStatus describes a container status.
type ContainerStatus struct {
	ID    string
	State State
}

// ContainerConfig describes one container runtime configuration.
type ContainerConfig struct {
	ID string

	// RootFs is the container workload image on the host.
	RootFs string

	// Interactive specifies if the container runs in the foreground.
	Interactive bool

	// Console is a console path provided by the caller.
	Console string

	// Cmd specifies the command to run on a container
	Cmd Cmd
}

// valid checks that the container configuration is valid.
func (containerConfig *ContainerConfig) valid() bool {
	if containerConfig.ID == "" {
		containerConfig.ID = uuid.Generate().String()
	}

	return true
}

// Container is composed of a set of containers and a runtime environment.
// A Container can be created, deleted, started, stopped, listed, entered, paused and restored.
type Container struct {
	id    string
	podID string

	rootFs string

	config *ContainerConfig

	pod *Pod

	runPath       string
	configPath    string
	containerPath string

	state State

	process Process
}

// ID returns the container identifier string.
func (c *Container) ID() string {
	return c.id
}

// GetToken returns the token related to this container's process.
func (c *Container) GetToken() string {
	return c.process.Token
}

// GetPid returns the pid related to this container's process.
func (c *Container) GetPid() int {
	return c.process.Pid
}

// SetPid sets and stores the given pid as the pid of container's process.
func (c *Container) SetPid(pid int) error {
	c.process.Pid = pid

	if err := c.pod.storage.storeContainerProcess(c.podID, c.id, c.process); err != nil {
		return err
	}

	return nil
}

// fetchContainer fetches a container config from a pod ID and returns a Container.
func fetchContainer(pod *Pod, containerID string) (*Container, error) {
	fs := filesystem{}
	config, err := fs.fetchContainerConfig(pod.id, containerID)
	if err != nil {
		return nil, err
	}

	glog.Infof("Info structure:\n%+v\n", config)

	return createContainer(pod, config)
}

// storeContainer stores a container config.
func (c *Container) storeContainer() error {
	fs := filesystem{}
	err := fs.storeContainerResource(c.pod.id, c.id, configFileType, *(c.config))
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) setContainerState(state stateString) error {
	c.state = State{
		State: state,
	}

	err := c.pod.storage.storeContainerResource(c.podID, c.id, stateFileType, c.state)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) createContainersDirs() error {
	err := os.MkdirAll(c.runPath, os.ModeDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(c.configPath, os.ModeDir)
	if err != nil {
		c.pod.storage.deleteContainerResources(c.podID, c.id, nil)
		return err
	}

	return nil
}

func createContainers(pod *Pod, contConfigs []ContainerConfig) ([]*Container, error) {
	var containers []*Container

	for _, contConfig := range contConfigs {
		if contConfig.valid() == false {
			return containers, fmt.Errorf("Invalid container configuration")
		}

		c := &Container{
			id:            contConfig.ID,
			podID:         pod.id,
			rootFs:        contConfig.RootFs,
			config:        &contConfig,
			pod:           pod,
			runPath:       filepath.Join(runStoragePath, pod.id, contConfig.ID),
			configPath:    filepath.Join(configStoragePath, pod.id, contConfig.ID),
			containerPath: filepath.Join(pod.id, contConfig.ID),
			state:         State{},
			process:       Process{},
		}

		state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
		if err == nil {
			c.state.State = state.State
		}

		process, err := c.pod.storage.fetchContainerProcess(c.podID, c.id)
		if err == nil {
			c.process = process
		}

		containers = append(containers, c)
	}

	return containers, nil
}

func createContainer(pod *Pod, contConfig ContainerConfig) (*Container, error) {
	if contConfig.valid() == false {
		return nil, fmt.Errorf("Invalid container configuration")
	}

	c := &Container{
		id:            contConfig.ID,
		podID:         pod.id,
		rootFs:        contConfig.RootFs,
		config:        &contConfig,
		pod:           pod,
		runPath:       filepath.Join(runStoragePath, pod.id, contConfig.ID),
		configPath:    filepath.Join(configStoragePath, pod.id, contConfig.ID),
		containerPath: filepath.Join(pod.id, contConfig.ID),
		state:         State{},
		process:       Process{},
	}

	err := c.createContainersDirs()
	if err != nil {
		return nil, err
	}

	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	if err == nil && state.State != "" {
		c.state.State = state.State
		return c, nil
	}

	err = c.pod.setContainerState(c.id, stateReady)
	if err != nil {
		return nil, err
	}

	process, err := c.pod.storage.fetchContainerProcess(c.podID, c.id)
	if err == nil {
		c.process = process
	}

	return c, nil
}

func (c *Container) delete() error {
	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	if err != nil {
		return err
	}

	if state.State != stateReady && state.State != stateStopped {
		return fmt.Errorf("Container not ready or stopped, impossible to delete")
	}

	err = c.pod.storage.deleteContainerResources(c.podID, c.id, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) start() error {
	state, err := c.pod.storage.fetchPodState(c.pod.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Pod not running, impossible to start the container")
	}

	state, err = c.pod.storage.fetchContainerState(c.podID, c.id)
	if err != nil {
		return err
	}

	if state.State != stateReady && state.State != stateStopped {
		return fmt.Errorf("Container not ready or stopped, impossible to start")
	}

	err = state.validTransition(stateReady, stateRunning)
	if err != nil {
		err = state.validTransition(stateStopped, stateRunning)
		if err != nil {
			return err
		}
	}

	err = c.pod.agent.startAgent()
	if err != nil {
		return err
	}

	err = c.pod.agent.startContainer(*c.pod, *(c.config))
	if err != nil {
		c.stop()
		return err
	}

	err = c.setContainerState(stateRunning)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) stop() error {
	state, err := c.pod.storage.fetchPodState(c.pod.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Pod not running, impossible to stop the container")
	}

	state, err = c.pod.storage.fetchContainerState(c.pod.id, c.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Container not running, impossible to stop")
	}

	err = state.validTransition(stateRunning, stateStopped)
	if err != nil {
		return err
	}

	err = c.pod.agent.startAgent()
	if err != nil {
		return err
	}

	err = c.pod.agent.killContainer(*c.pod, *c, syscall.SIGTERM)
	if err != nil {
		return err
	}

	err = c.pod.agent.stopContainer(*c.pod, *c)
	if err != nil {
		return err
	}

	err = c.setContainerState(stateStopped)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) enter(cmd Cmd) error {
	state, err := c.pod.storage.fetchPodState(c.pod.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Pod not running, impossible to enter the container")
	}

	state, err = c.pod.storage.fetchContainerState(c.pod.id, c.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Container not running, impossible to enter")
	}

	err = c.pod.agent.startAgent()
	if err != nil {
		return err
	}

	err = c.pod.agent.exec(*c.pod, *c, cmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) kill(signal syscall.Signal) error {
	state, err := c.pod.storage.fetchPodState(c.pod.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Pod not running, impossible to signal the container")
	}

	state, err = c.pod.storage.fetchContainerState(c.pod.id, c.id)
	if err != nil {
		return err
	}

	if state.State != stateRunning {
		return fmt.Errorf("Container not running, impossible to signal the container")
	}

	err = c.pod.agent.startAgent()
	if err != nil {
		return err
	}

	err = c.pod.agent.killContainer(*c.pod, *c, signal)
	if err != nil {
		return err
	}

	return nil
}
