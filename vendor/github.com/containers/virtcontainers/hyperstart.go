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
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	cniTypes "github.com/containernetworking/cni/pkg/types"
	"github.com/containers/virtcontainers/pkg/hyperstart"
)

var defaultSockPathTemplates = []string{"%s/%s/hyper.sock", "%s/%s/tty.sock"}
var defaultChannelTemplate = "sh.hyper.channel.%d"
var defaultDeviceIDTemplate = "channel%d"
var defaultIDTemplate = "charch%d"
var defaultSharedDir = "/tmp/hyper/shared/pods/"
var defaultPauseBinDir = "/usr/bin/"
var mountTag = "hyperShared"
var rootfsDir = "rootfs"
var pauseBinName = "pause"
var pauseContainerName = "pause-container"
var maxHostnameLen = 64

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
		virtLog.Infof("No sockets from configuration")

		podSocketPaths := []string{
			fmt.Sprintf(defaultSockPathTemplates[0], runStoragePath, pod.id),
			fmt.Sprintf(defaultSockPathTemplates[1], runStoragePath, pod.id),
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

	virtLog.Debugf("Hyperstart config %v", c)

	return true
}

// hyper is the Agent interface implementation for hyperstart.
type hyper struct {
	config HyperConfig
	proxy  proxy
}

type hyperstartProxyCmd struct {
	cmd     string
	message interface{}
	token   string
}

func (h *hyper) buildHyperContainerProcess(cmd Cmd) (*hyperstart.Process, error) {
	var envVars []hyperstart.EnvironmentVar

	for _, e := range cmd.Envs {
		envVar := hyperstart.EnvironmentVar{
			Env:   e.Var,
			Value: e.Value,
		}

		envVars = append(envVars, envVar)
	}

	process := &hyperstart.Process{
		Terminal:         cmd.Interactive,
		Args:             cmd.Args,
		Envs:             envVars,
		Workdir:          cmd.WorkDir,
		User:             cmd.User,
		Group:            cmd.PrimaryGroup,
		AdditionalGroups: cmd.SupplementaryGroups,
	}

	return process, nil
}

func (h *hyper) processHyperRoute(route *cniTypes.Route, deviceName string) *hyperstart.Route {
	gateway := route.GW.String()
	if gateway == "<nil>" {
		gateway = ""
	}

	destination := route.Dst.String()
	if destination == defaultRouteDest {
		destination = defaultRouteLabel
	}

	// Skip IPv6 because not supported by hyperstart
	if destination != defaultRouteDest && route.Dst.IP.To4() == nil {
		virtLog.Warnf("IPv6 route destination %q not supported", destination)
		return nil
	}

	return &hyperstart.Route{
		Dest:    destination,
		Gateway: gateway,
		Device:  deviceName,
	}
}

func (h *hyper) buildNetworkInterfacesAndRoutes(pod Pod) ([]hyperstart.NetworkIface, []hyperstart.Route, error) {
	networkNS, err := pod.storage.fetchPodNetwork(pod.id)
	if err != nil {
		return []hyperstart.NetworkIface{}, []hyperstart.Route{}, err
	}

	if networkNS.NetNsPath == "" {
		return []hyperstart.NetworkIface{}, []hyperstart.Route{}, nil
	}

	netIfaces, err := getIfacesFromNetNs(networkNS.NetNsPath)
	if err != nil {
		return []hyperstart.NetworkIface{}, []hyperstart.Route{}, err
	}

	var ifaces []hyperstart.NetworkIface
	var routes []hyperstart.Route
	for _, endpoint := range networkNS.Endpoints {
		netIface, err := getNetIfaceByName(endpoint.NetPair.VirtIface.Name, netIfaces)
		if err != nil {
			return []hyperstart.NetworkIface{}, []hyperstart.Route{}, err
		}

		var ipAddrs []hyperstart.IPAddress
		for _, ipConfig := range endpoint.Properties.IPs {
			// Skip IPv6 because not supported by hyperstart
			if ipConfig.Version == "6" || ipConfig.Address.IP.To4() == nil {
				continue
			}

			netMask, _ := ipConfig.Address.Mask.Size()

			ipAddr := hyperstart.IPAddress{
				IPAddress: ipConfig.Address.IP.String(),
				NetMask:   fmt.Sprintf("%d", netMask),
			}

			ipAddrs = append(ipAddrs, ipAddr)
		}

		iface := hyperstart.NetworkIface{
			NewDevice:   endpoint.NetPair.VirtIface.Name,
			IPAddresses: ipAddrs,
			MTU:         netIface.MTU,
			MACAddr:     endpoint.NetPair.TAPIface.HardAddr,
		}

		ifaces = append(ifaces, iface)

		for _, r := range endpoint.Properties.Routes {
			route := h.processHyperRoute(r, endpoint.NetPair.VirtIface.Name)
			if route == nil {
				continue
			}

			routes = append(routes, *route)
		}
	}

	return ifaces, routes, nil
}

func (h *hyper) copyPauseBinary(podID string) error {
	pauseDir := filepath.Join(defaultSharedDir, podID, pauseContainerName, rootfsDir)

	if err := os.MkdirAll(pauseDir, dirMode); err != nil {
		return err
	}

	pausePath := filepath.Join(pauseDir, pauseBinName)

	return fileCopy(h.config.PauseBinPath, pausePath)
}

func (h *hyper) removePauseBinary(podID string) error {
	pauseDir := filepath.Join(defaultSharedDir, podID, pauseContainerName)

	return os.RemoveAll(pauseDir)
}

func (h *hyper) bindMountContainerRootfs(podID, cID, cRootFs string, readonly bool) error {
	rootfsDest := filepath.Join(defaultSharedDir, podID, cID, rootfsDir)

	return bindMount(cRootFs, rootfsDest, readonly)
}

// bindMountContainerMounts handles bind-mounts by bindmounting to the host shared directory
// which is mounted through 9pfs in the VM.
// Hyperstart uses "fsmap" struct to bind mount these mounts in the hypertstart shared directory
// to the correct mountpoint within the container rootfs.
func (h *hyper) bindMountContainerMounts(podID string, cID string, mounts []Mount) ([]*hyperstart.FsmapDescriptor, error) {
	if mounts == nil {
		return nil, nil
	}

	var fsmap []*hyperstart.FsmapDescriptor

	// TODO: We need to handle system mounts by having the agent create them inside the VM.
	// Handle just bind mounts for now

	for ind := range mounts {
		m := &(mounts[ind])

		if isSystemMount(m.Destination) {
			continue
		}

		if m.Type != "bind" {
			continue
		}

		randBytes, err := generateRandomBytes(8)
		if err != nil {
			return nil, err
		}

		// These mounts are created in the hyperstart shared dir
		filename := fmt.Sprintf("%s-%s-%s", cID, hex.EncodeToString(randBytes), filepath.Base(m.Destination))
		mountDest := filepath.Join(defaultSharedDir, podID, filename)

		err = bindMount(m.Source, mountDest, false)
		if err != nil {
			return nil, err
		}

		m.HostPath = mountDest

		// Check if mount is readonly, let hyperstart handle the readonly mount within the VM
		readonly := false
		for _, flag := range m.Options {
			if flag == "ro" {
				readonly = true
			}
		}

		fsmapDesc := &hyperstart.FsmapDescriptor{
			Source:       filename,
			Path:         m.Destination,
			ReadOnly:     readonly,
			DockerVolume: false,
		}

		fsmap = append(fsmap, fsmapDesc)
	}

	return fsmap, nil
}

func (h *hyper) bindUnmountContainerMounts(mounts []Mount) error {
	if mounts == nil {
		return nil
	}

	for _, m := range mounts {
		if !isSystemMount(m.Destination) && m.Type == "bind" {
			err := syscall.Unmount(m.HostPath, 0)

			if err != nil {
				virtLog.Warnf("Could not umount :%s", m.HostPath)
				return err
			}
		}
	}
	return nil
}

func (h *hyper) bindUnmountContainerRootfs(podID, cID string) error {
	rootfsDest := filepath.Join(defaultSharedDir, podID, cID, rootfsDir)
	syscall.Unmount(rootfsDest, 0)

	return nil
}

func (h *hyper) bindUnmountAllRootfs(pod Pod) {
	for _, c := range pod.containers {
		h.bindUnmountContainerMounts(c.mounts)
		if c.state.Fstype == "" {
			// Need to check for error returned by this call.
			// See: https://github.com/containers/virtcontainers/issues/295
			h.bindUnmountContainerRootfs(pod.id, c.id)
		}
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

	h.proxy = pod.proxy

	return nil
}

func (h *hyper) createPod(pod *Pod) (err error) {
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

	return nil
}

func (h *hyper) capabilities() capabilities {
	var caps capabilities

	// add all capabilities supported by agent
	caps.setBlockDeviceSupport()

	return caps
}

// exec is the agent command execution implementation for hyperstart.
func (h *hyper) exec(pod *Pod, c Container, process Process, cmd Cmd) error {
	hyperProcess, err := h.buildHyperContainerProcess(cmd)
	if err != nil {
		return err
	}

	execCommand := hyperstart.ExecCommand{
		Container: c.id,
		Process:   *hyperProcess,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.ExecCmd,
		message: execCommand,
		token:   process.Token,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	return nil
}

// startPod is the agent Pod starting implementation for hyperstart.
func (h *hyper) startPod(pod Pod) error {
	ifaces, routes, err := h.buildNetworkInterfacesAndRoutes(pod)
	if err != nil {
		return err
	}

	hostname := pod.config.Hostname
	if len(hostname) > maxHostnameLen {
		hostname = hostname[:maxHostnameLen]
	}

	hyperPod := hyperstart.Pod{
		Hostname:   hostname,
		Containers: []hyperstart.Container{},
		Interfaces: ifaces,
		Routes:     routes,
		ShareDir:   mountTag,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.StartPod,
		message: hyperPod,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	if err := h.startPauseContainer(pod.id); err != nil {
		return err
	}

	return nil
}

// stopPod is the agent Pod stopping implementation for hyperstart.
func (h *hyper) stopPod(pod Pod) error {
	if err := h.stopPauseContainer(pod.id); err != nil {
		return err
	}

	return nil
}

// startPauseContainer starts a specific container running the pause binary provided.
func (h *hyper) startPauseContainer(podID string) error {
	cmd := Cmd{
		Args:        []string{fmt.Sprintf("./%s", pauseBinName)},
		Envs:        []EnvVar{},
		WorkDir:     "/",
		Interactive: false,
	}

	process, err := h.buildHyperContainerProcess(cmd)
	if err != nil {
		return err
	}

	container := hyperstart.Container{
		ID:      pauseContainerName,
		Image:   pauseContainerName,
		Rootfs:  rootfsDir,
		Process: process,
	}

	if err := h.copyPauseBinary(podID); err != nil {
		return err
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.NewContainer,
		message: container,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	return nil
}

func (h *hyper) startOneContainer(pod Pod, c Container) error {
	process, err := h.buildHyperContainerProcess(c.config.Cmd)
	if err != nil {
		return err
	}

	container := hyperstart.Container{
		ID:      c.id,
		Image:   c.id,
		Rootfs:  rootfsDir,
		Process: process,
	}

	if c.state.Fstype != "" {
		driveName, err := getVirtDriveName(c.state.BlockIndex)
		if err != nil {
			return err
		}

		container.Fstype = c.state.Fstype
		container.Image = driveName
	} else {

		if err := h.bindMountContainerRootfs(pod.id, c.id, c.rootFs, c.config.ReadonlyRootfs); err != nil {
			h.bindUnmountAllRootfs(pod)
			return err
		}
	}

	//TODO : Enter mount namespace

	// Handle container mounts
	fsmap, err := h.bindMountContainerMounts(pod.id, c.id, c.mounts)
	if err != nil {
		h.bindUnmountAllRootfs(pod)
		return err
	}

	// Assign fsmap for hyperstart to mount these at the correct location within the container
	container.Fsmap = fsmap

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
func (h *hyper) createContainer(pod *Pod, c *Container) error {
	return nil
}

// startContainer is the agent Container starting implementation for hyperstart.
func (h *hyper) startContainer(pod Pod, c Container) error {
	return h.startOneContainer(pod, c)
}

func (h *hyper) stopPauseContainer(podID string) error {
	if err := h.killOneContainer(pauseContainerName, syscall.SIGKILL, true); err != nil {
		return err
	}

	if err := h.removePauseBinary(podID); err != nil {
		return err
	}

	return nil
}

// stopContainer is the agent Container stopping implementation for hyperstart.
func (h *hyper) stopContainer(pod Pod, c Container) error {
	return h.stopOneContainer(pod.id, c)
}

func (h *hyper) stopOneContainer(podID string, c Container) error {
	removeCommand := hyperstart.RemoveCommand{
		Container: c.id,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.RemoveContainer,
		message: removeCommand,
	}

	if _, err := h.proxy.sendCmd(proxyCmd); err != nil {
		return err
	}

	if err := h.bindUnmountContainerMounts(c.mounts); err != nil {
		return err
	}

	if c.state.Fstype == "" {
		if err := h.bindUnmountContainerRootfs(podID, c.id); err != nil {
			return err
		}
	}

	return nil
}

// killContainer is the agent process signal implementation for hyperstart.
func (h *hyper) killContainer(pod Pod, c Container, signal syscall.Signal, all bool) error {
	return h.killOneContainer(c.id, signal, all)
}

func (h *hyper) killOneContainer(cID string, signal syscall.Signal, all bool) error {
	killCmd := hyperstart.KillCommand{
		Container:    cID,
		Signal:       signal,
		AllProcesses: all,
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
