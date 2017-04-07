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

	"github.com/01org/ciao/ssntp/uuid"
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

	NetworkModel  NetworkModel
	NetworkConfig NetworkConfig

	// Volumes is a list of shared volumes between the host and the Pod.
	Volumes []Volume

	// Containers describe the list of containers within a Pod.
	// This list can be empty and populated by adding containers
	// to the Pod a posteriori.
	Containers []ContainerConfig

	// Annotations keys must be unique strings an must be name-spaced
	// with e.g. reverse domain notation (org.clearlinux.key).
	Annotations map[string]string
}

// valid checks that the pod configuration is valid.
func (podConfig *PodConfig) valid() bool {
	if _, err := newHypervisor(podConfig.HypervisorType); err != nil {
		podConfig.HypervisorType = QemuHypervisor
	}

	if podConfig.ID == "" {
		podConfig.ID = uuid.Generate().String()
	}

	return true
}

// lock locks any pod to prevent it from being accessed by other processes.
func lockPod(podID string) (*os.File, error) {
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
	storage    resourceStorage
	network    network

	config *PodConfig

	volumes []Volume

	containers []*Container

	runPath    string
	configPath string

	url string

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
	return p.url
}

// GetContainers returns a container config list.
func (p *Pod) GetContainers() []*Container {
	return p.containers
}

func (p *Pod) createSetStates() error {
	err := p.setPodState(StateReady)
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

	err = hypervisor.init(podConfig.HypervisorConfig)
	if err != nil {
		return nil, err
	}

	network := newNetwork(podConfig.NetworkModel)

	p := &Pod{
		id:         podConfig.ID,
		hypervisor: hypervisor,
		agent:      agent,
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

	err = p.storage.createAllResources(*p)
	if err != nil {
		return nil, err
	}

	err = p.hypervisor.createPod(podConfig)
	if err != nil {
		p.storage.deletePodResources(p.id, nil)
		return nil, err
	}

	var agentConfig interface{}

	if podConfig.AgentConfig != nil {
		switch podConfig.AgentConfig.(type) {
		case (map[string]interface{}):
			agentConfig = newAgentConfig(podConfig)
		default:
			agentConfig = podConfig.AgentConfig.(interface{})
		}
	} else {
		agentConfig = nil
	}

	err = p.agent.init(p, agentConfig)
	if err != nil {
		p.storage.deletePodResources(p.id, nil)
		return nil, err
	}

	state, err := p.storage.fetchPodState(p.id)
	if err == nil && state.State != "" {
		p.state = state
		return p, nil
	}

	err = p.createSetStates()
	if err != nil {
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
	err := p.setPodState(StateRunning)
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

	err := p.agent.start(p)
	if err != nil {
		p.stop()
		return err
	}

	virtLog.Infof("VM started")

	return nil
}

// start starts a pod. The containers that are making the pod
// will be started.
func (p *Pod) start() error {
	err := p.startCheckStates()
	if err != nil {
		return err
	}

	err = p.agent.startPod(*p)
	if err != nil {
		p.stop()
		return err
	}

	err = p.startSetStates()
	if err != nil {
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

	err = p.setPodState(StateStopped)
	if err != nil {
		return err
	}

	return nil
}

// stopVM stops the agent inside the VM and shut down the VM itself.
func (p *Pod) stopVM() error {
	err := p.agent.stop(*p)
	if err != nil {
		return err
	}

	err = p.hypervisor.stopPod()
	if err != nil {
		return err
	}

	return nil
}

// stop stops a pod. The containers that are making the pod
// will be destroyed.
func (p *Pod) stop() error {
	err := p.stopCheckStates()
	if err != nil {
		return err
	}

	err = p.agent.stopPod(*p)
	if err != nil {
		return err
	}

	err = p.stopSetStates()
	if err != nil {
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

func (p *Pod) setPodState(state stateString) error {
	p.state = State{
		State: state,
	}

	err := p.storage.storePodResource(p.id, stateFileType, p.state)
	if err != nil {
		return err
	}

	return nil
}

// endSession makes sure to end the session properly.
func (p *Pod) endSession() error {
	return nil
}

func (p *Pod) setContainerState(contID string, state stateString) error {
	contState := State{
		State: state,
	}

	err := p.storage.storeContainerResource(p.id, contID, stateFileType, contState)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pod) setContainersState(state stateString) error {
	for _, container := range p.config.Containers {
		err := p.setContainerState(container.ID, state)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pod) deleteContainerState(contID string) error {
	err := p.storage.deleteContainerResources(p.id, contID, []podResource{stateFileType})
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

func (p *Pod) checkContainerState(contID string, expectedState stateString) error {
	state, err := p.storage.fetchContainerState(p.id, contID)
	if err != nil {
		return err
	}

	if state.State != expectedState {
		return fmt.Errorf("Container %s not %s", contID, expectedState)
	}

	return nil
}

func (p *Pod) checkContainersState(state stateString) error {
	for _, container := range p.config.Containers {
		err := p.checkContainerState(container.ID, state)
		if err != nil {
			return err
		}
	}

	return nil
}
