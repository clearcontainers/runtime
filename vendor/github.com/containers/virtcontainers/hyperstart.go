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
	"math/rand"
	"os"
	"path/filepath"
	"syscall"

	"github.com/golang/glog"

	"github.com/containers/virtcontainers/pkg/hyperstart"
	hyperJson "github.com/hyperhq/runv/hyperstart/api/json"
)

var defaultSockPathTemplates = []string{"/tmp/hyper-pod-%s.sock", "/tmp/tty-pod%s.sock"}
var defaultChannelTemplate = "sh.hyper.channel.%d"
var defaultDeviceIDTemplate = "channel%d"
var defaultIDTemplate = "charch%d"
var defaultSharedDir = "/tmp/hyper/shared/pods/"
var defaultPauseBinDir = "/usr/bin/"
var mountTag = "hyperShared"
var rootfsDir = "rootfs"
var pauseBinName = "pause"
var pauseContainerName = "pause-container"

const (
	unixSocket = "unix"
)

// HyperConfig is a structure storing information needed for
// hyperstart agent initialization.
type HyperConfig struct {
	SockCtlName  string
	SockTtyName  string
	Volumes      []Volume
	Sockets      []Socket
	PauseBinPath string
}

func (c *HyperConfig) validate(pod Pod) bool {
	if len(c.Sockets) == 0 {
		glog.Infof("No sockets from configuration\n")

		podSocketPaths := []string{
			fmt.Sprintf(defaultSockPathTemplates[0], pod.id),
			fmt.Sprintf(defaultSockPathTemplates[1], pod.id),
		}

		c.SockCtlName = podSocketPaths[0]
		c.SockTtyName = podSocketPaths[1]

		for i := 0; i < len(podSocketPaths); i++ {
			s := Socket{
				DeviceID: fmt.Sprintf(defaultDeviceIDTemplate, i),
				ID:       fmt.Sprintf(defaultIDTemplate, i),
				HostPath: podSocketPaths[i],
				Name:     fmt.Sprintf(defaultChannelTemplate, i),
			}
			c.Sockets = append(c.Sockets, s)
		}
	}

	if len(c.Sockets) != 2 {
		return false
	}

	if c.PauseBinPath == "" {
		c.PauseBinPath = filepath.Join(defaultPauseBinDir, pauseBinName)
	}

	glog.Infof("Hyperstart config %v\n", c)

	return true
}

// hyper is the Agent interface implementation for hyperstart.
type hyper struct {
	config HyperConfig
	proxy  proxy
}

// ExecInfo is the structure corresponding to the format
// expected by hyperstart to execute a command on the guest.
type ExecInfo struct {
	Container string            `json:"container"`
	Process   hyperJson.Process `json:"process"`
}

// KillCommand is the structure corresponding to the format
// expected by hyperstart to kill a container on the guest.
type KillCommand struct {
	Container string         `json:"container"`
	Signal    syscall.Signal `json:"signal"`
}

// RemoveContainer is the structure corresponding to the format
// expected by hyperstart to remove a container on the guest.
type RemoveContainer struct {
	Container string `json:"container"`
}

type hyperstartProxyCmd struct {
	cmd     string
	message interface{}
	token   string
}

func (h *hyper) buildHyperContainerProcess(cmd Cmd, terminal bool) (*hyperJson.Process, error) {
	var envVars []hyperJson.EnvironmentVar

	for _, e := range cmd.Envs {
		envVar := hyperJson.EnvironmentVar{
			Env:   e.Var,
			Value: e.Value,
		}

		envVars = append(envVars, envVar)
	}

	process := &hyperJson.Process{
		User:     cmd.User,
		Group:    cmd.Group,
		Terminal: terminal,
		Args:     cmd.Args,
		Envs:     envVars,
		Workdir:  cmd.WorkDir,

		// TODO: Remove when switching to new proxy.
		// Temporary to get it still working with the current proxy.
		Stdio:  uint64(rand.Int63()),
		Stderr: uint64(rand.Int63()),
	}

	return process, nil
}

func (h *hyper) linkPauseBinary(podID string) error {
	pauseDir := filepath.Join(defaultSharedDir, podID, pauseContainerName, rootfsDir)

	if err := os.MkdirAll(pauseDir, dirMode); err != nil {
		return err
	}

	pausePath := filepath.Join(pauseDir, pauseBinName)

	return os.Link(h.config.PauseBinPath, pausePath)
}

func (h *hyper) unlinkPauseBinary(podID string) error {
	pauseDir := filepath.Join(defaultSharedDir, podID, pauseContainerName)

	return os.RemoveAll(pauseDir)
}

func (h *hyper) bindMountContainerRootfs(podID, cID, cRootFs string) error {
	rootfsDest := filepath.Join(defaultSharedDir, podID, cID)

	return bindMount(cRootFs, rootfsDest)
}

func (h *hyper) bindUnmountContainerRootfs(podID, cID string) error {
	rootfsDest := filepath.Join(defaultSharedDir, podID, cID)
	syscall.Unmount(rootfsDest, 0)

	return nil
}

func (h *hyper) bindUnmountAllRootfs(pod Pod) {
	for _, c := range pod.containers {
		h.bindUnmountContainerRootfs(pod.id, c.id)
	}
}

// init is the agent initialization implementation for hyperstart.
func (h *hyper) init(pod *Pod, config interface{}) (err error) {
	switch c := config.(type) {
	case HyperConfig:
		if c.validate(*pod) == false {
			return fmt.Errorf("Invalid configuration")
		}
		h.config = c
	default:
		return fmt.Errorf("Invalid config type")
	}

	// Override pod agent configuration
	pod.config.AgentConfig = h.config

	for _, volume := range h.config.Volumes {
		err := pod.hypervisor.addDevice(volume, fsDev)
		if err != nil {
			return err
		}
	}

	for _, socket := range h.config.Sockets {
		err := pod.hypervisor.addDevice(socket, serialPortDev)
		if err != nil {
			return err
		}
	}

	// Adding the hyper shared volume.
	// This volume contains all bind mounted container bundles.
	sharedVolume := Volume{
		MountTag: mountTag,
		HostPath: filepath.Join(defaultSharedDir, pod.id),
	}

	if err := os.MkdirAll(sharedVolume.HostPath, dirMode); err != nil {
		return err
	}

	if err := pod.hypervisor.addDevice(sharedVolume, fsDev); err != nil {
		return err
	}

	h.proxy, err = newProxy(pod.config.ProxyType)
	if err != nil {
		return err
	}

	return nil
}

// start is the agent starting implementation for hyperstart.
func (h *hyper) start(pod *Pod) error {
	proxyInfos, err := h.proxy.register(*pod)
	if err != nil {
		return err
	}

	if len(proxyInfos) != len(pod.containers) {
		return fmt.Errorf("Retrieved %d proxy infos, expecting %d", len(proxyInfos), len(pod.containers))
	}

	for idx := range pod.containers {
		pod.containers[idx].process = Process{
			Token: proxyInfos[idx].Token,
		}

		if err := pod.containers[idx].storeProcess(); err != nil {
			return err
		}
	}

	return h.proxy.disconnect()
}

// stop is the agent stopping implementation for hyperstart.
func (h *hyper) stop(pod Pod) error {
	if _, err := h.proxy.connect(pod, false); err != nil {
		return err
	}

	if err := h.proxy.unregister(pod); err != nil {
		return err
	}

	return h.proxy.disconnect()
}

// exec is the agent command execution implementation for hyperstart.
func (h *hyper) exec(pod Pod, c Container, cmd Cmd) (*Process, error) {
	proxyInfo, err := h.proxy.connect(pod, true)
	if err != nil {
		return nil, err
	}

	process, err := h.buildHyperContainerProcess(cmd, c.config.Interactive)
	if err != nil {
		return nil, err
	}

	execInfo := ExecInfo{
		Container: c.id,
		Process:   *process,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.ExecCmd,
		message: execInfo,
		token:   proxyInfo.Token,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return nil, err
	}

	if err := h.proxy.disconnect(); err != nil {
		return nil, err
	}

	processInfo := &Process{
		Token: proxyInfo.Token,
	}

	return processInfo, nil
}

// startPod is the agent Pod starting implementation for hyperstart.
func (h *hyper) startPod(pod Pod) error {
	proxyInfo, err := h.proxy.connect(pod, true)
	if err != nil {
		return err
	}

	hyperPod := hyperJson.Pod{
		Hostname:             pod.id,
		DeprecatedContainers: []hyperJson.Container{},
		ShareDir:             mountTag,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.StartPod,
		message: hyperPod,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	if err := h.startPauseContainer(pod.id, proxyInfo.Token); err != nil {
		return err
	}

	for _, c := range pod.containers {
		if err := h.startOneContainer(pod, *c); err != nil {
			return err
		}
	}

	return h.proxy.disconnect()
}

// stopPod is the agent Pod stopping implementation for hyperstart.
func (h *hyper) stopPod(pod Pod) error {
	if _, err := h.proxy.connect(pod, false); err != nil {
		return err
	}

	for _, c := range pod.containers {
		state, err := pod.storage.fetchContainerState(pod.id, c.id)
		if err != nil {
			return err
		}

		if state.State != StateRunning {
			continue
		}

		if err := h.killOneContainer(c.id, syscall.SIGTERM); err != nil {
			return err
		}

		if err := h.stopOneContainer(pod.id, c.id); err != nil {
			return err
		}
	}

	if err := h.stopPauseContainer(pod.id); err != nil {
		return err
	}

	if err := h.proxy.disconnect(); err != nil {
		return err
	}

	return nil
}

// startPauseContainer starts a specific container running the pause binary provided.
func (h *hyper) startPauseContainer(podID, token string) error {
	cmd := Cmd{
		Args:    []string{fmt.Sprintf("./%s", pauseBinName)},
		Envs:    []EnvVar{},
		WorkDir: "/",
	}

	process, err := h.buildHyperContainerProcess(cmd, false)
	if err != nil {
		return err
	}

	container := hyperJson.Container{
		Id:      pauseContainerName,
		Image:   pauseContainerName,
		Rootfs:  rootfsDir,
		Process: process,
	}

	if err := h.linkPauseBinary(podID); err != nil {
		return err
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.NewContainer,
		message: container,
		token:   token,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	return nil
}

func (h *hyper) startOneContainer(pod Pod, c Container) error {
	process, err := h.buildHyperContainerProcess(c.config.Cmd, c.config.Interactive)
	if err != nil {
		return err
	}

	container := hyperJson.Container{
		Id:      c.id,
		Image:   c.id,
		Rootfs:  rootfsDir,
		Process: process,
	}

	if err := h.bindMountContainerRootfs(pod.id, c.id, c.rootFs); err != nil {
		h.bindUnmountAllRootfs(pod)
		return err
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.NewContainer,
		message: container,
		token:   c.process.Token,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	return nil
}

// createContainer is the agent Container creation implementation for hyperstart.
func (h *hyper) createContainer(pod Pod, c *Container) error {
	proxyInfo, err := h.proxy.connect(pod, true)
	if err != nil {
		return err
	}

	c.process = Process{
		Token: proxyInfo.Token,
	}

	if err := c.storeProcess(); err != nil {
		return err
	}

	return h.proxy.disconnect()
}

// startContainer is the agent Container starting implementation for hyperstart.
func (h *hyper) startContainer(pod Pod, c Container) error {
	if _, err := h.proxy.connect(pod, false); err != nil {
		return err
	}

	if err := h.startOneContainer(pod, c); err != nil {
		return err
	}

	return h.proxy.disconnect()
}

func (h *hyper) stopPauseContainer(podID string) error {
	if err := h.killOneContainer(pauseContainerName, syscall.SIGKILL); err != nil {
		return err
	}

	if err := h.unlinkPauseBinary(podID); err != nil {
		return err
	}

	return nil
}

// stopContainer is the agent Container stopping implementation for hyperstart.
func (h *hyper) stopContainer(pod Pod, c Container) error {
	if _, err := h.proxy.connect(pod, false); err != nil {
		return err
	}

	if err := h.stopOneContainer(pod.id, c.id); err != nil {
		return err
	}

	if err := h.proxy.disconnect(); err != nil {
		return err
	}

	return nil
}

func (h *hyper) stopOneContainer(podID, cID string) error {
	removeContainer := RemoveContainer{
		Container: cID,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.RemoveContainer,
		message: removeContainer,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	if err := h.bindUnmountContainerRootfs(podID, cID); err != nil {
		return err
	}

	return nil
}

// killContainer is the agent process signal implementation for hyperstart.
func (h *hyper) killContainer(pod Pod, c Container, signal syscall.Signal) error {
	if _, err := h.proxy.connect(pod, false); err != nil {
		return err
	}

	if err := h.killOneContainer(c.id, signal); err != nil {
		return err
	}

	if err := h.proxy.disconnect(); err != nil {
		return err
	}

	return nil
}

func (h *hyper) killOneContainer(cID string, signal syscall.Signal) error {
	killCmd := KillCommand{
		Container: cID,
		Signal:    signal,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.KillContainer,
		message: killCmd,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	return nil
}
