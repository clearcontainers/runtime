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

	"github.com/containers/virtcontainers/pkg/hyperstart"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

var defaultSockPathTemplates = []string{"%s/%s/hyper.sock", "%s/%s/tty.sock"}
var defaultChannelTemplate = "sh.hyper.channel.%d"
var defaultDeviceIDTemplate = "channel%d"
var defaultIDTemplate = "charch%d"
var defaultSharedDir = "/run/hyper/shared/pods/"
var mountTag = "hyperShared"
var maxHostnameLen = 64

const (
	unixSocket = "unix"
)

// HyperConfig is a structure storing information needed for
// hyperstart agent initialization.
type HyperConfig struct {
	SockCtlName string
	SockTtyName string
	Volumes     []Volume
	Sockets     []Socket
}

// Logger returns a logrus logger appropriate for logging HyperConfig messages
func (c *HyperConfig) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "hyperstart")
}

func (c *HyperConfig) validate(pod Pod) bool {
	if len(c.Sockets) == 0 {
		c.Logger().Info("No sockets from configuration")

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

// Logger returns a logrus logger appropriate for logging hyper messages
func (h *hyper) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "hyper")
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
		NoNewPrivileges:  cmd.NoNewPrivileges,
	}

	return process, nil
}

func (h *hyper) processHyperRoute(route netlink.Route, deviceName string) *hyperstart.Route {
	gateway := route.Gw.String()
	if gateway == "<nil>" {
		gateway = ""
	}

	var destination string
	if route.Dst == nil {
		destination = ""
	} else {
		destination = route.Dst.String()
		if destination == defaultRouteDest {
			destination = defaultRouteLabel
		}

		// Skip IPv6 because not supported by hyperstart
		if route.Dst.IP.To4() == nil {
			h.Logger().WithFields(logrus.Fields{
				"unsupported-route-type": "ipv6",
				"destination":            destination,
			}).Warn("unsupported route")
			return nil
		}
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

	var ifaces []hyperstart.NetworkIface
	var routes []hyperstart.Route
	for _, endpoint := range networkNS.Endpoints {
		var ipAddresses []hyperstart.IPAddress
		for _, addr := range endpoint.Properties().Addrs {
			// Skip IPv6 because not supported by hyperstart.
			// Skip localhost interface.
			if addr.IP.To4() == nil || addr.IP.IsLoopback() {
				continue
			}

			netMask, _ := addr.Mask.Size()

			ipAddress := hyperstart.IPAddress{
				IPAddress: addr.IP.String(),
				NetMask:   fmt.Sprintf("%d", netMask),
			}

			ipAddresses = append(ipAddresses, ipAddress)
		}

		iface := hyperstart.NetworkIface{
			NewDevice:   endpoint.Name(),
			IPAddresses: ipAddresses,
			MTU:         endpoint.Properties().Iface.MTU,
			MACAddr:     endpoint.HardwareAddr(),
		}

		ifaces = append(ifaces, iface)

		for _, r := range endpoint.Properties().Routes {
			route := h.processHyperRoute(r, endpoint.Name())
			if route == nil {
				continue
			}

			routes = append(routes, *route)
		}
	}

	return ifaces, routes, nil
}

func fsMapFromMounts(mounts []*Mount) []*hyperstart.FsmapDescriptor {
	var fsmap []*hyperstart.FsmapDescriptor

	for _, m := range mounts {
		fsmapDesc := &hyperstart.FsmapDescriptor{
			Source:       m.Source,
			Path:         m.Destination,
			ReadOnly:     m.ReadOnly,
			DockerVolume: false,
		}

		fsmap = append(fsmap, fsmapDesc)
	}

	return fsmap
}

// init is the agent initialization implementation for hyperstart.
func (h *hyper) init(pod *Pod, config interface{}) (err error) {
	switch c := config.(type) {
	case HyperConfig:
		if c.validate(*pod) == false {
			return fmt.Errorf("Invalid hyperstart configuration: %v", c)
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

// vmURL returns VM URL from hyperstart agent implementation.
func (h *hyper) vmURL() (string, error) {
	return "", nil
}

// setProxyURL sets proxy URL for hyperstart agent implementation.
func (h *hyper) setProxyURL(url string) error {
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

	return pod.hypervisor.addDevice(sharedVolume, fsDev)
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

	return nil
}

// stopPod is the agent Pod stopping implementation for hyperstart.
func (h *hyper) stopPod(pod Pod) error {
	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.DestroyPod,
		message: nil,
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

	container.SystemMountsInfo.BindMountDev = c.systemMountsInfo.BindMountDev

	if c.state.Fstype != "" {
		driveName, err := getVirtDriveName(c.state.BlockIndex)
		if err != nil {
			return err
		}

		container.Fstype = c.state.Fstype
		container.Image = driveName
	} else {

		if err := bindMountContainerRootfs(defaultSharedDir, pod.id, c.id, c.rootFs, false); err != nil {
			bindUnmountAllRootfs(defaultSharedDir, pod)
			return err
		}
	}

	//TODO : Enter mount namespace

	// Handle container mounts
	newMounts, err := bindMountContainerMounts(defaultSharedDir, pod.id, c.id, c.mounts)
	if err != nil {
		bindUnmountAllRootfs(defaultSharedDir, pod)
		return err
	}

	fsmap := fsMapFromMounts(newMounts)

	// Append container mounts for block devices passed with --device.
	for _, device := range c.devices {
		d, ok := device.(*BlockDevice)

		if ok {
			fsmapDesc := &hyperstart.FsmapDescriptor{
				Source:       d.VirtPath,
				Path:         d.DeviceInfo.ContainerPath,
				AbsolutePath: true,
				DockerVolume: false,
			}
			fsmap = append(fsmap, fsmapDesc)
		}
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

	if err := bindUnmountContainerMounts(c.mounts); err != nil {
		return err
	}

	if c.state.Fstype == "" {
		if err := bindUnmountContainerRootfs(defaultSharedDir, podID, c.id); err != nil {
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

func (h *hyper) processListContainer(pod Pod, c Container, options ProcessListOptions) (ProcessList, error) {
	return h.processListOneContainer(pod.id, c.id, options)
}

func (h *hyper) processListOneContainer(podID, cID string, options ProcessListOptions) (ProcessList, error) {
	psCmd := hyperstart.PsCommand{
		Container: cID,
		Format:    options.Format,
		PsArgs:    options.Args,
	}

	proxyCmd := hyperstartProxyCmd{
		cmd:     hyperstart.PsContainer,
		message: psCmd,
	}

	response, err := h.proxy.sendCmd(proxyCmd)
	if err != nil {
		return nil, err
	}

	msg, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to get response message from container %s pod %s", cID, podID)
	}

	return msg, nil
}
