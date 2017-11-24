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

	"github.com/sirupsen/logrus"
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

	// Device configuration for devices that must be available within the container.
	DeviceInfos []DeviceInfo
}

// valid checks that the container configuration is valid.
func (c *ContainerConfig) valid() bool {
	if c == nil {
		return false
	}

	if c.ID == "" {
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

	devices []Device
}

// ID returns the container identifier string.
func (c *Container) ID() string {
	return c.id
}

// Logger returns a logrus logger appropriate for logging Container messages
func (c *Container) Logger() *logrus.Entry {
	return virtLog.WithFields(logrus.Fields{
		"subsystem":    "container",
		"container-id": c.id,
		"pod-id":       c.podID,
	})
}

// Pod returns the pod handler related to this container.
func (c *Container) Pod() VCPod {
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

func (c *Container) setStateHotpluggedDrive(hotplugged bool) error {
	c.state.HotpluggedDrive = hotplugged

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

	return c.storeProcess()
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

func (c *Container) storeDevices() error {
	return c.pod.storage.storeContainerDevices(c.podID, c.id, c.devices)
}

func (c *Container) fetchDevices() ([]Device, error) {
	return c.pod.storage.fetchContainerDevices(c.podID, c.id)
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

	container, err := createContainer(pod, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create container with config %v in pod %v: %v", config, pod.id, err)
	}

	return container, nil
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

	// Devices will be found in storage after create stage has completed.
	// We fetch devices from storage at all other stages.
	storedDevices, err := c.fetchDevices()
	if err == nil {
		c.devices = storedDevices
	} else {
		// If devices were not found in storage, create Device implementations
		// from the configuration. This should happen at create.

		devices, err := newDevices(contConfig.DeviceInfos)
		if err != nil {
			return &Container{}, err
		}
		c.devices = devices
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

	agentCaps := c.pod.agent.capabilities()
	hypervisorCaps := c.pod.hypervisor.capabilities()

	if agentCaps.isBlockDeviceSupported() && hypervisorCaps.isBlockDeviceHotplugSupported() {
		if err := c.hotplugDrive(); err != nil {
			return err
		}
	}

	// Attach devices
	if err := c.attachDevices(); err != nil {
		return err
	}

	if err = c.pod.agent.startContainer(*(c.pod), *c); err != nil {
		c.Logger().WithError(err).Error("Failed to start container")

		if err := c.stop(); err != nil {
			c.Logger().WithError(err).Warn("Failed to stop container")
		}
		return err
	}
	c.storeMounts()
	c.storeDevices()

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

	// In case the container status has been updated implicitly because
	// the container process has terminated, it might be possible that
	// someone try to stop the container, and we don't want to issue an
	// error in that case. This should be a no-op.
	if state.State == StateStopped {
		c.Logger().Info("Container already stopped")
		return nil
	}

	if state.State != StateRunning {
		return fmt.Errorf("Container not running, impossible to stop")
	}

	err = state.validTransition(StateRunning, StateStopped)
	if err != nil {
		return err
	}

	defer func() {
		// If shim is still running something went wrong
		// Make sure we stop the shim process
		if running, _ := isShimRunning(c.process.Pid); running {
			l := c.Logger()
			l.Warn("Failed to stop container so stopping dangling shim")
			if err := stopShim(c.process.Pid); err != nil {
				l.WithError(err).Warn("failed to stop shim")
			}
		}

	}()

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

	if err = c.detachDevices(); err != nil {
		return err
	}

	if err := c.removeDrive(); err != nil {
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
	podState, err := c.pod.storage.fetchPodState(c.pod.id)
	if err != nil {
		return err
	}

	if podState.State != StateReady && podState.State != StateRunning {
		return fmt.Errorf("Pod not ready or running, impossible to signal the container")
	}

	state, err := c.pod.storage.fetchContainerState(c.podID, c.id)
	if err != nil {
		return err
	}

	// In case our container is "ready", there is no point in trying to
	// send any signal because nothing has been started. However, this is
	// a valid case that we handle by doing nothing or by killing the shim
	// and updating the container state, according to the signal.
	if state.State == StateReady {
		if signal != syscall.SIGTERM && signal != syscall.SIGKILL {
			c.Logger().WithField("signal", signal).Info("Not sending singal as container already ready")
			return nil
		}

		// Calling into stopShim() will send a SIGKILL to the shim.
		// This signal will be forwarded to the proxy and it will be
		// handled by the proxy itself. Indeed, because there is no
		// process running inside the VM, there is no point in sending
		// this signal to our agent. Instead, the proxy will take care
		// of that signal by killing the shim (sending an exit code).
		if err := stopShim(c.process.Pid); err != nil {
			return err
		}

		return c.setContainerState(StateStopped)
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

func (c *Container) processList(options ProcessListOptions) (ProcessList, error) {
	state, err := c.fetchState("ps")
	if err != nil {
		return nil, err
	}

	if state.State != StateRunning {
		return nil, fmt.Errorf("Container not running, impossible to list processes")
	}

	if _, _, err := c.pod.proxy.connect(*(c.pod), false); err != nil {
		return nil, err
	}
	defer c.pod.proxy.disconnect()

	return c.pod.agent.processListContainer(*(c.pod), *c, options)
}

func (c *Container) createShimProcess(token, url string, cmd Cmd) (*Process, error) {
	if c.pod.state.URL != url {
		return &Process{}, fmt.Errorf("Pod URL %s and URL from proxy %s MUST be identical", c.pod.state.URL, url)
	}

	shimParams := ShimParams{
		Container: c.id,
		Token:     token,
		URL:       url,
		Console:   cmd.Console,
		Detach:    cmd.Detach,
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

func (c *Container) hotplugDrive() error {
	dev, err := getDeviceForPath(c.rootFs)

	if err == errMountPointNotFound {
		return nil
	}

	if err != nil {
		return err
	}

	c.Logger().WithFields(logrus.Fields{
		"device-major": dev.major,
		"device-minor": dev.minor,
		"mount-point":  dev.mountPoint,
	}).Info("device details")

	isDM, err := checkStorageDriver(dev.major, dev.minor)
	if err != nil {
		return err
	}

	if !isDM {
		return nil
	}

	// If device mapper device, then fetch the full path of the device
	devicePath, fsType, err := getDevicePathAndFsType(dev.mountPoint)
	if err != nil {
		return err
	}

	c.Logger().WithFields(logrus.Fields{
		"device-path": devicePath,
		"fs-type":     fsType,
	}).Info("Block device detected")

	// Add drive with id as container id
	devID := makeBlockDevIDForHypervisor(c.id)
	drive := Drive{
		File:   devicePath,
		Format: "raw",
		ID:     devID,
	}

	if err := c.pod.hypervisor.hotplugAddDevice(drive, blockDev); err != nil {
		return err
	}
	c.setStateHotpluggedDrive(true)

	driveIndex, err := c.pod.getAndSetPodBlockIndex()
	if err != nil {
		return err
	}

	if err := c.setStateBlockIndex(driveIndex); err != nil {
		return err
	}

	return c.setStateFstype(fsType)
}

// isDriveUsed checks if a drive has been used for container rootfs
func (c *Container) isDriveUsed() bool {
	if c.state.Fstype == "" {
		return false
	}
	return true
}

func (c *Container) removeDrive() (err error) {
	if c.isDriveUsed() && c.state.HotpluggedDrive {
		c.Logger().Info("unplugging block device")

		devID := makeBlockDevIDForHypervisor(c.id)
		drive := Drive{
			ID: devID,
		}

		l := c.Logger().WithField("device-id", devID)
		l.Info("Unplugging block device")

		if err := c.pod.hypervisor.hotplugRemoveDevice(drive, blockDev); err != nil {
			l.WithError(err).Info("Failed to unplug block device")
			return err
		}
	}

	return nil
}

func (c *Container) attachDevices() error {
	for _, device := range c.devices {
		if err := device.attach(c.pod.hypervisor, c); err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) detachDevices() error {
	for _, device := range c.devices {
		if err := device.detach(c.pod.hypervisor); err != nil {
			return err
		}
	}

	return nil
}
