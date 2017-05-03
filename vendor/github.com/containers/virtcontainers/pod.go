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
	"strings"
	"syscall"
	"time"
)

// controlSocket is the pod control socket.
// It is an hypervisor resource, and for example qemu's control
// socket is the QMP one.
const controlSocket = "ctrl.sock"

// monitorSocket is the pod monitoring socket.
// It is an hypervisor resource, and is a qmp socket in the qemu case.
// This is a socket that any monitoring entity will listen to in order
// to understand if the VM is still alive or not.
const monitorSocket = "monitor.sock"

// stateString is a string representing a pod state.
type stateString string

const (
	// StateReady represents a pod/container that's ready to be run
	StateReady stateString = "ready"

	// StateRunning represents a pod/container that's currently running.
	StateRunning stateString = "running"

	// StateStopped represents a pod/container that has been stopped.
	StateStopped stateString = "stopped"
)

// State is a pod state structure.
type State struct {
	State stateString `json:"state"`
	URL   string      `json:"url,omitempty"`
}

// valid checks that the pod state is valid.
func (state *State) valid() bool {
	for _, validState := range []stateString{StateReady, StateRunning, StateStopped} {
		if state.State == validState {
			return true
		}
	}

	return false
}

// validTransition returns an error if we want to move to
// an unreachable state.
func (state *State) validTransition(oldState stateString, newState stateString) error {
	if state.State != oldState {
		return fmt.Errorf("Invalid state %s (Expecting %s)", state.State, oldState)
	}

	switch state.State {
	case StateReady:
		if newState == StateRunning {
			return nil
		}

	case StateRunning:
		if newState == StateStopped {
			return nil
		}

	case StateStopped:
		if newState == StateRunning {
			return nil
		}
	}

	return fmt.Errorf("Can not move from %s to %s",
		state.State, newState)
}

// Volume is a shared volume between the host and the VM,
// defined by its mount tag and its host path.
type Volume struct {
	// MountTag is a label used as a hint to the guest.
	MountTag string

	// HostPath is the host filesystem path for this volume.
	HostPath string
}

// Volumes is a Volume list.
type Volumes []Volume

// Set assigns volume values from string to a Volume.
func (v *Volumes) Set(volStr string) error {
	if volStr == "" {
		return fmt.Errorf("volStr cannot be empty")
	}

	volSlice := strings.Split(volStr, " ")
	const expectedVolLen = 2
	const volDelimiter = ":"

	for _, vol := range volSlice {
		volArgs := strings.Split(vol, volDelimiter)

		if len(volArgs) != expectedVolLen {
			return fmt.Errorf("Wrong string format: %s, expecting only %v parameters separated with %q",
				vol, expectedVolLen, volDelimiter)
		}

		if volArgs[0] == "" || volArgs[1] == "" {
			return fmt.Errorf("Volume parameters cannot be empty")
		}

		volume := Volume{
			MountTag: volArgs[0],
			HostPath: volArgs[1],
		}

		*v = append(*v, volume)
	}

	return nil
}

// String converts a Volume to a string.
func (v *Volumes) String() string {
	var volSlice []string

	for _, volume := range *v {
		volSlice = append(volSlice, fmt.Sprintf("%s:%s", volume.MountTag, volume.HostPath))
	}

	return strings.Join(volSlice, " ")
}

// Socket defines a socket to communicate between
// the host and any process inside the VM.
type Socket struct {
	DeviceID string
	ID       string
	HostPath string
	Name     string
}

// Sockets is a Socket list.
type Sockets []Socket

// Set assigns socket values from string to a Socket.
func (s *Sockets) Set(sockStr string) error {
	if sockStr == "" {
		return fmt.Errorf("sockStr cannot be empty")
	}

	sockSlice := strings.Split(sockStr, " ")
	const expectedSockCount = 4
	const sockDelimiter = ":"

	for _, sock := range sockSlice {
		sockArgs := strings.Split(sock, sockDelimiter)

		if len(sockArgs) != expectedSockCount {
			return fmt.Errorf("Wrong string format: %s, expecting only %v parameters separated with %q", sock, expectedSockCount, sockDelimiter)
		}

		for _, a := range sockArgs {
			if a == "" {
				return fmt.Errorf("Socket parameters cannot be empty")
			}
		}

		socket := Socket{
			DeviceID: sockArgs[0],
			ID:       sockArgs[1],
			HostPath: sockArgs[2],
			Name:     sockArgs[3],
		}

		*s = append(*s, socket)
	}

	return nil
}

// String converts a Socket to a string.
func (s *Sockets) String() string {
	var sockSlice []string

	for _, sock := range *s {
		sockSlice = append(sockSlice, fmt.Sprintf("%s:%s:%s:%s", sock.DeviceID, sock.ID, sock.HostPath, sock.Name))
	}

	return strings.Join(sockSlice, " ")
}

// EnvVar is a key/value structure representing a command
// environment variable.
type EnvVar struct {
	Var   string
	Value string
}

// Cmd represents a command to execute in a running container.
type Cmd struct {
	Args    []string
	Envs    []EnvVar
	WorkDir string

	User  string
	Group string

	Interactive bool
	Console     string
}

// Resources describes VM resources configuration.
type Resources struct {
	// VCPUs is the number of available virtual CPUs.
	VCPUs uint

	// Memory is the amount of available memory in MiB.
	Memory uint
}

// PodStatus describes a pod status.
type PodStatus struct {
	ID               string
	State            State
	Hypervisor       HypervisorType
	Agent            AgentType
	ContainersStatus []ContainerStatus
}

// PodConfig is a Pod configuration.
type PodConfig struct {
	ID string

	// Field specific to OCI specs, needed to setup all the hooks
	Hooks Hooks

	// VMConfig is the VM configuration to set for this pod.
	VMConfig Resources

	HypervisorType   HypervisorType
	HypervisorConfig HypervisorConfig

	AgentType   AgentType
	AgentConfig interface{}

	ProxyType   ProxyType
	ProxyConfig interface{}

	ShimType   ShimType
	ShimConfig interface{}

	NetworkModel  NetworkModel
	NetworkConfig NetworkConfig

	// Volumes is a list of shared volumes between the host and the Pod.
	Volumes []Volume

	// Containers describe the list of containers within a Pod.
	// This list can be empty and populated by adding containers
	// to the Pod a posteriori.
	Containers []ContainerConfig

	// Annotations keys must be unique strings and must be name-spaced
	// with e.g. reverse domain notation (org.clearlinux.key).
	Annotations map[string]string
}

// valid checks that the pod configuration is valid.
func (podConfig *PodConfig) valid() bool {
	if podConfig.ID == "" {
		return false
	}

	if _, err := newHypervisor(podConfig.HypervisorType); err != nil {
		podConfig.HypervisorType = QemuHypervisor
	}

	return true
}

// lock locks any pod to prevent it from being accessed by other processes.
func lockPod(podID string) (*os.File, error) {
	if podID == "" {
		return nil, errNeedPodID
	}

	fs := filesystem{}
	podlockFile, _, err := fs.podURI(podID, lockFileType)
	if err != nil {
		return nil, err
	}

	lockFile, err := os.Open(podlockFile)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX)
	if err != nil {
		return nil, err
	}

	return lockFile, nil
}

// unlock unlocks any pod to allow it being accessed by other processes.
func unlockPod(lockFile *os.File) error {
	if lockFile == nil {
		return fmt.Errorf("lockFile cannot be empty")
	}

	err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	if err != nil {
		return err
	}

	lockFile.Close()

	return nil
}

// Pod is composed of a set of containers and a runtime environment.
// A Pod can be created, deleted, started, stopped, listed, entered, paused and restored.
type Pod struct {
	id string

	hypervisor hypervisor
	agent      agent
	proxy      proxy
	shim       shim
	storage    resourceStorage
	network    network

	config *PodConfig

	volumes []Volume

	containers []*Container

	runPath    string
	configPath string

	state State

	lockFile *os.File
}

// ID returns the pod identifier string.
func (p *Pod) ID() string {
	return p.id
}

// Annotations returns any annotation that a user could have stored through the pod.
func (p *Pod) Annotations(key string) (string, error) {
	value, exist := p.config.Annotations[key]
	if exist == false {
		return "", fmt.Errorf("Annotations key %s does not exist", key)
	}

	return value, nil
}

// URL returns the pod URL for any runtime to connect to the proxy.
func (p *Pod) URL() string {
	return p.state.URL
}

// GetContainers returns a container config list.
func (p *Pod) GetContainers() []*Container {
	return p.containers
}

func (p *Pod) createSetStates() error {
	p.state.State = StateReady
	err := p.setPodState(p.state)
	if err != nil {
		return err
	}

	err = p.setContainersState(StateReady)
	if err != nil {
		return err
	}

	return nil
}

// createPod creates a pod from a pod description, the containers list, the hypervisor
// and the agent passed through the Config structure.
// It will create and store the pod structure, and then ask the hypervisor
// to physically create that pod i.e. starts a VM for that pod to eventually
// be started.
func createPod(podConfig PodConfig) (*Pod, error) {
	if podConfig.valid() == false {
		return nil, fmt.Errorf("Invalid pod configuration")
	}

	agent := newAgent(podConfig.AgentType)

	hypervisor, err := newHypervisor(podConfig.HypervisorType)
	if err != nil {
		return nil, err
	}

	if err := hypervisor.init(podConfig.HypervisorConfig); err != nil {
		return nil, err
	}

	proxy, err := newProxy(podConfig.ProxyType)
	if err != nil {
		return nil, err
	}

	shim, err := newShim(podConfig.ShimType)
	if err != nil {
		return nil, err
	}

	network := newNetwork(podConfig.NetworkModel)

	p := &Pod{
		id:         podConfig.ID,
		hypervisor: hypervisor,
		agent:      agent,
		proxy:      proxy,
		shim:       shim,
		storage:    &filesystem{},
		network:    network,
		config:     &podConfig,
		volumes:    podConfig.Volumes,
		runPath:    filepath.Join(runStoragePath, podConfig.ID),
		configPath: filepath.Join(configStoragePath, podConfig.ID),
		state:      State{},
	}

	containers, err := createContainers(p, podConfig.Containers)
	if err != nil {
		return nil, err
	}

	p.containers = containers

	if err := p.storage.createAllResources(*p); err != nil {
		return nil, err
	}

	if err := p.hypervisor.createPod(podConfig); err != nil {
		p.storage.deletePodResources(p.id, nil)
		return nil, err
	}

	agentConfig := newAgentConfig(podConfig)
	if err := p.agent.init(p, agentConfig); err != nil {
		p.storage.deletePodResources(p.id, nil)
		return nil, err
	}

	state, err := p.storage.fetchPodState(p.id)
	if err == nil && state.State != "" {
		p.state = state
		return p, nil
	}

	if err := p.createSetStates(); err != nil {
		p.storage.deletePodResources(p.id, nil)
		return nil, err
	}

	return p, nil
}

// storePod stores a pod config.
func (p *Pod) storePod() error {
	err := p.storage.storePodResource(p.id, configFileType, *(p.config))
	if err != nil {
		return err
	}

	for _, container := range p.containers {
		err = p.storage.storeContainerResource(p.id, container.id, configFileType, *(container.config))
		if err != nil {
			return err
		}
	}

	return nil
}

// fetchPod fetches a pod config from a pod ID and returns a pod.
func fetchPod(podID string) (*Pod, error) {
	if podID == "" {
		return nil, errNeedPodID
	}

	fs := filesystem{}
	config, err := fs.fetchPodConfig(podID)
	if err != nil {
		return nil, err
	}

	virtLog.Infof("Info structure: %+v", config)

	return createPod(config)
}

// delete deletes an already created pod.
// The VM in which the pod is running will be shut down.
func (p *Pod) delete() error {
	state, err := p.storage.fetchPodState(p.id)
	if err != nil {
		return err
	}

	if state.State != StateReady && state.State != StateStopped {
		return fmt.Errorf("Pod not ready or stopped, impossible to delete")
	}

	err = p.storage.deletePodResources(p.id, nil)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pod) startCheckStates() error {
	state, err := p.storage.fetchPodState(p.id)
	if err != nil {
		return err
	}

	err = state.validTransition(StateReady, StateRunning)
	if err != nil {
		err = state.validTransition(StateStopped, StateRunning)
		if err != nil {
			return err
		}
	}

	err = p.checkContainersState(StateReady)
	if err != nil {
		err = p.checkContainersState(StateStopped)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pod) startSetStates() error {
	p.state.State = StateRunning
	err := p.setPodState(p.state)
	if err != nil {
		return err
	}

	err = p.setContainersState(StateRunning)
	if err != nil {
		return err
	}

	return nil
}

// startVM starts the VM, ensuring it is started before it returns or issuing
// an error in case of timeout. Then it connects to the agent inside the VM.
func (p *Pod) startVM() error {
	vmStartedCh := make(chan struct{})
	vmStoppedCh := make(chan struct{})

	go func() {
		p.network.run(p.config.NetworkConfig.NetNSPath, func() error {
			err := p.hypervisor.startPod(vmStartedCh, vmStoppedCh)
			return err
		})
	}()

	// Wait for the pod started notification
	select {
	case <-vmStartedCh:
		break
	case <-time.After(time.Second):
		return fmt.Errorf("Did not receive the pod started notification")
	}

	virtLog.Infof("VM started")

	return nil
}

// startShims registers all containers to the proxy and starts one
// shim per container.
func (p *Pod) startShims() error {
	proxyInfos, url, err := p.proxy.register(*p)
	if err != nil {
		return err
	}

	if err := p.proxy.disconnect(); err != nil {
		return err
	}

	if len(proxyInfos) != len(p.containers) {
		return fmt.Errorf("Retrieved %d proxy infos, expecting %d", len(proxyInfos), len(p.containers))
	}

	p.state.URL = url
	if err := p.setPodState(p.state); err != nil {
		return err
	}

	for idx := range p.containers {
		shimParams := ShimParams{
			Token:   proxyInfos[idx].Token,
			URL:     url,
			Console: p.containers[idx].config.Cmd.Console,
		}

		pid, err := p.shim.start(*p, shimParams)
		if err != nil {
			return err
		}

		p.containers[idx].process = Process{
			Token: proxyInfos[idx].Token,
			Pid:   pid,
		}

		if err := p.containers[idx].storeProcess(); err != nil {
			return err
		}
	}

	virtLog.Infof("Shim(s) started")

	return nil
}

// start starts a pod. The containers that are making the pod
// will be started.
func (p *Pod) start() error {
	if err := p.startCheckStates(); err != nil {
		return err
	}

	if _, _, err := p.proxy.connect(*p, false); err != nil {
		return err
	}
	defer p.proxy.disconnect()

	if err := p.agent.startPod(*p); err != nil {
		return err
	}

	if err := p.startSetStates(); err != nil {
		return err
	}

	virtLog.Infof("Started Pod %s", p.ID())

	return nil
}

func (p *Pod) stopCheckStates() error {
	state, err := p.storage.fetchPodState(p.id)
	if err != nil {
		return err
	}

	err = state.validTransition(StateRunning, StateStopped)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pod) stopSetStates() error {
	err := p.setContainersState(StateStopped)
	if err != nil {
		return err
	}

	p.state.State = StateStopped
	err = p.setPodState(p.state)
	if err != nil {
		return err
	}

	return nil
}

// stopVM stops the agent inside the VM and shut down the VM itself.
func (p *Pod) stopVM() error {
	if _, _, err := p.proxy.connect(*p, false); err != nil {
		return err
	}

	if err := p.proxy.unregister(*p); err != nil {
		return err
	}

	if err := p.proxy.disconnect(); err != nil {
		return err
	}

	if err := p.hypervisor.stopPod(); err != nil {
		return err
	}

	return nil
}

// stop stops a pod. The containers that are making the pod
// will be destroyed.
func (p *Pod) stop() error {
	if err := p.stopCheckStates(); err != nil {
		return err
	}

	if _, _, err := p.proxy.connect(*p, false); err != nil {
		return err
	}
	defer p.proxy.disconnect()

	if err := p.agent.stopPod(*p); err != nil {
		return err
	}

	if err := p.stopSetStates(); err != nil {
		return err
	}

	return nil
}

// list lists all pod running on the host.
func (p *Pod) list() ([]Pod, error) {
	return nil, nil
}

// enter runs an executable within a pod.
func (p *Pod) enter(args []string) error {
	return nil
}

func (p *Pod) setPodState(state State) error {
	err := p.storage.storePodResource(p.id, stateFileType, state)
	if err != nil {
		return err
	}

	return nil
}

// endSession makes sure to end the session properly.
func (p *Pod) endSession() error {
	return nil
}

func (p *Pod) setContainerState(containerID string, state stateString) error {
	if containerID == "" {
		return errNeedContainerID
	}

	contState := State{
		State: state,
	}

	err := p.storage.storeContainerResource(p.id, containerID, stateFileType, contState)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pod) setContainersState(state stateString) error {
	if state == "" {
		return errNeedState
	}

	for _, container := range p.config.Containers {
		err := p.setContainerState(container.ID, state)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pod) deleteContainerState(containerID string) error {
	if containerID == "" {
		return errNeedContainerID
	}

	err := p.storage.deleteContainerResources(p.id, containerID, []podResource{stateFileType})
	if err != nil {
		return err
	}

	return nil
}

func (p *Pod) deleteContainersState() error {
	for _, container := range p.config.Containers {
		err := p.deleteContainerState(container.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pod) checkContainerState(containerID string, expectedState stateString) error {
	if containerID == "" {
		return errNeedContainerID
	}

	if expectedState == "" {
		return fmt.Errorf("expectedState cannot be empty")
	}

	state, err := p.storage.fetchContainerState(p.id, containerID)
	if err != nil {
		return err
	}

	if state.State != expectedState {
		return fmt.Errorf("Container %s not %s", containerID, expectedState)
	}

	return nil
}

func (p *Pod) checkContainersState(state stateString) error {
	if state == "" {
		return errNeedState
	}

	for _, container := range p.config.Containers {
		err := p.checkContainerState(container.ID, state)
		if err != nil {
			return err
		}
	}

	return nil
}
