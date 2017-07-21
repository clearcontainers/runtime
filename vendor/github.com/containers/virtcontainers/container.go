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
	"time"
)

// Process gathers data related to a container process.
type Process struct {
	Token     string
	Pid       int
	StartTime time.Time
}

// ContainerStatus describes a container status.
type ContainerStatus struct {
	ID        string
	State     State
	PID       int
	StartTime time.Time
	RootFs    string

	// Annotations allow clients to store arbitrary values,
	// for example to add additional status values required
	// to support particular specifications.
	Annotations map[string]string
}

// Mount describes a container mount.
type Mount struct {
	Source      string
	Destination string

	// Type specifies the type of filesystem to mount.
	Type string

	// Options list all the mount options of the filesystem.
	Options []string

	// HostPath used to store host side bind mount path
	HostPath string
}

// ContainerConfig describes one container runtime configuration.
type ContainerConfig struct {
	ID string

	// RootFs is the container workload image on the host.
	RootFs string

	// ReadOnlyRootfs indicates if the rootfs should be mounted readonly
	ReadonlyRootfs bool

	// Cmd specifies the command to run on a container
	Cmd Cmd

	// Annotations allow clients to store arbitrary values,
	// for example to add additional status values required
	// to support particular specifications.
	Annotations map[string]string

	Mounts []Mount
}

// valid checks that the container configuration is valid.
func (containerConfig *ContainerConfig) valid() bool {
	if containerConfig == nil {
		return false
	}

	if containerConfig.ID == "" {
		return false
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

	mounts []Mount
}

// ID returns the container identifier string.
func (c *Container) ID() string {
	return c.id
}

// Pod returns the pod handler related to this container.
func (c *Container) Pod() *Pod {
	return c.pod
}

// Process returns the container process.
func (c *Container) Process() Process {
	return c.process
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

	return c.storeProcess()
}

func (c *Container) setStateBlockIndex(index int) error {
	c.state.BlockIndex = index

	err := c.pod.storage.storeContainerResource(c.pod.id, c.id, stateFileType, c.state)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) setStateFstype(fstype string) error {
	c.state.Fstype = fstype

	err := c.pod.storage.storeContainerResource(c.pod.id, c.id, stateFileType, c.state)
	if err != nil {
		return err
	}

	return nil
}

// URL returns the URL related to the pod.
func (c *Container) URL() string {
	return c.pod.URL()
}

// GetAnnotations returns container's annotations
func (c *Container) GetAnnotations() map[string]string {
	return c.config.Annotations
}

func (c *Container) startShim() error {
	proxyInfo, url, err := c.pod.proxy.connect(*(c.pod), true)
	if err != nil {
		return err
	}

	if err := c.pod.proxy.disconnect(); err != nil {
		return err
	}

	process, err := c.createShimProcess(proxyInfo.Token, url, c.config.Cmd)
	if err != nil {
		return err
	}

	c.process = *process

	if err := c.storeProcess(); err != nil {
		return err
	}

	return nil
}

func (c *Container) storeProcess() error {
	return c.pod.storage.storeContainerProcess(c.podID, c.id, c.process)
}

func (c *Container) fetchProcess() (Process, error) {
	return c.pod.storage.fetchContainerProcess(c.podID, c.id)
}

func (c *Container) storeMounts() error {
	return c.pod.storage.storeContainerMounts(c.podID, c.id, c.mounts)
}

func (c *Container) fetchMounts() ([]Mount, error) {
	return c.pod.storage.fetchContainerMounts(c.podID, c.id)
}

// fetchContainer fetches a container config from a pod ID and returns a Container.
func fetchContainer(pod *Pod, containerID string) (*Container, error) {
	if pod == nil {
		return nil, errNeedPod
	}

	if containerID == "" {
		return nil, errNeedContainerID
	}

	fs := filesystem{}
	config, err := fs.fetchContainerConfig(pod.id, containerID)
	if err != nil {
		return nil, err
	}

	virtLog.Infof("Info structure: %+v", config)

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

// setContainerState sets both the in-memory and on-disk state of the
// container.
func (c *Container) setContainerState(state stateString) error {
	if state == "" {
		return errNeedState
	}

	// update in-memory state
	c.state.State = state

	// update on-disk state
	err := c.pod.storage.storeContainerResource(c.pod.id, c.id, stateFileType, c.state)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) createContainersDirs() error {
	err := os.MkdirAll(c.runPath, dirMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(c.configPath, dirMode)
	if err != nil {
		c.pod.storage.deleteContainerResources(c.podID, c.id, nil)
		return err
	}

	return nil
}

// newContainer creates a Container structure from a pod and a container configuration.
func newContainer(pod *Pod, contConfig ContainerConfig) (*Container, error) {
	if contConfig.valid() == false {
		return &Container{}, fmt.Errorf("Invalid container configuration")
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
		mounts:        contConfig.Mounts,
	}

	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	if err == nil {
		c.state = state
	}

	process, err := c.pod.storage.fetchContainerProcess(c.podID, c.id)
	if err == nil {
		c.process = process
	}

	mounts, err := c.fetchMounts()
	if err == nil {
		c.mounts = mounts
	}

	return c, nil
}

// newContainers uses newContainer to create a Container slice.
func newContainers(pod *Pod, contConfigs []ContainerConfig) ([]*Container, error) {
	if pod == nil {
		return nil, errNeedPod
	}

	var containers []*Container

	for _, contConfig := range contConfigs {
		c, err := newContainer(pod, contConfig)
		if err != nil {
			return containers, err
		}

		containers = append(containers, c)
	}

	return containers, nil
}

// createContainer creates and start a container inside a Pod.
func createContainer(pod *Pod, contConfig ContainerConfig) (*Container, error) {
	if pod == nil {
		return nil, errNeedPod
	}

	c, err := newContainer(pod, contConfig)
	if err != nil {
		return nil, err
	}

	err = c.createContainersDirs()
	if err != nil {
		return nil, err
	}

	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	if err == nil && state.State != "" {
		c.state.State = state.State
		return c, nil
	}

	// If we reached that point, this means that no state file has been
	// found and that we are in the first creation of this container.
	// We don't want the following code to be executed outside of this
	// specific case.
	pod.containers = append(pod.containers, c)

	if err := c.startShim(); err != nil {
		return nil, err
	}

	if err := c.pod.setContainerState(c.id, StateReady); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Container) delete() error {
	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	if err != nil {
		return err
	}

	if state.State != StateReady && state.State != StateStopped {
		return fmt.Errorf("Container not ready or stopped, impossible to delete")
	}

	if err := stopShim(c.process.Pid); err != nil {
		return err
	}

	err = c.pod.storage.deleteContainerResources(c.podID, c.id, nil)
	if err != nil {
		return err
	}

	return nil
}

// fetchState retrieves the container state.
//
// cmd specifies the operation (or verb) that the retieval is destined
// for and is only used to make the returned error as descriptive as
// possible.
func (c *Container) fetchState(cmd string) (State, error) {
	if cmd == "" {
		return State{}, fmt.Errorf("Cmd cannot be empty")
	}

	state, err := c.pod.storage.fetchPodState(c.pod.id)
	if err != nil {
		return State{}, err
	}

	if state.State != StateRunning {
		return State{}, fmt.Errorf("Pod not running, impossible to %s the container", cmd)
	}

	state, err = c.pod.storage.fetchContainerState(c.podID, c.id)
	if err != nil {
		return State{}, err
	}

	return state, nil
}

func (c *Container) start() error {
	state, err := c.fetchState("start")
	if err != nil {
		return err
	}

	if state.State != StateReady && state.State != StateStopped {
		return fmt.Errorf("Container not ready or stopped, impossible to start")
	}

	err = state.validTransition(StateReady, StateRunning)
	if err != nil {
		err = state.validTransition(StateStopped, StateRunning)
		if err != nil {
			return err
		}
	}

	if _, _, err := c.pod.proxy.connect(*(c.pod), false); err != nil {
		return err
	}
	defer c.pod.proxy.disconnect()

	err = c.pod.agent.startContainer(*(c.pod), *c)
	if err != nil {
		c.stop()
		return err
	}

	c.storeMounts()

	err = c.setContainerState(StateRunning)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) stop() error {
	state, err := c.fetchState("stop")
	if err != nil {
		return err
	}

	// In case our container is "ready", there is no point in trying to
	// stop it because nothing has been started. However, this is a valid
	// case and we handle this by updating the container state only.
	if state.State == StateReady {
		if err := state.validTransition(StateReady, StateStopped); err != nil {
			return err
		}

		return c.setContainerState(StateStopped)
	}

	if state.State != StateRunning {
		return fmt.Errorf("Container not running, impossible to stop")
	}

	err = state.validTransition(StateRunning, StateStopped)
	if err != nil {
		return err
	}

	if _, _, err := c.pod.proxy.connect(*(c.pod), false); err != nil {
		return err
	}
	defer c.pod.proxy.disconnect()

	err = c.pod.agent.killContainer(*(c.pod), *c, syscall.SIGKILL, true)
	if err != nil {
		return err
	}

	// Wait for the end of container
	if err := waitForShim(c.process.Pid); err != nil {
		return err
	}

	err = c.pod.agent.stopContainer(*(c.pod), *c)
	if err != nil {
		return err
	}

	err = c.setContainerState(StateStopped)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) enter(cmd Cmd) (*Process, error) {
	state, err := c.fetchState("enter")
	if err != nil {
		return nil, err
	}

	if state.State != StateRunning {
		return nil, fmt.Errorf("Container not running, impossible to enter")
	}

	proxyInfo, url, err := c.pod.proxy.connect(*(c.pod), true)
	if err != nil {
		return nil, err
	}
	defer c.pod.proxy.disconnect()

	process, err := c.createShimProcess(proxyInfo.Token, url, cmd)
	if err != nil {
		return nil, err
	}

	if err := c.pod.agent.exec(c.pod, *c, *process, cmd); err != nil {
		return nil, err
	}

	return process, nil
}

func (c *Container) kill(signal syscall.Signal, all bool) error {
	state, err := c.fetchState("signal")
	if err != nil {
		return err
	}

	if state.State != StateRunning {
		return fmt.Errorf("Container not running, impossible to signal the container")
	}

	if _, _, err := c.pod.proxy.connect(*(c.pod), false); err != nil {
		return err
	}
	defer c.pod.proxy.disconnect()

	err = c.pod.agent.killContainer(*(c.pod), *c, signal, all)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) createShimProcess(token, url string, cmd Cmd) (*Process, error) {
	if c.pod.state.URL != url {
		return &Process{}, fmt.Errorf("Pod URL %s and URL from proxy %s MUST be identical", c.pod.state.URL, url)
	}

	shimParams := ShimParams{
		Token:   token,
		URL:     url,
		Console: cmd.Console,
		Detach:  cmd.Detach,
	}

	pid, err := c.pod.shim.start(*(c.pod), shimParams)
	if err != nil {
		return &Process{}, err
	}

	process := newProcess(token, pid)

	return &process, nil
}

func newProcess(token string, pid int) Process {
	return Process{
		Token:     token,
		Pid:       pid,
		StartTime: time.Now().UTC(),
	}
}
