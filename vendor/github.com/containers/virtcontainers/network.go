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
	"net"
	"os"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/containernetworking/cni/pkg/ns"
	types "github.com/containernetworking/cni/pkg/types/current"
	"github.com/containers/virtcontainers/logger/gloginterface"
	"golang.org/x/sys/unix"
)

// NetworkInterface defines a network interface.
type NetworkInterface struct {
	Name     string
	HardAddr string
}

// NetworkInterfacePair defines a pair between TAP and virtual network interfaces.
type NetworkInterfacePair struct {
	ID        string
	Name      string
	VirtIface NetworkInterface
	TAPIface  NetworkInterface
}

// NetworkConfig is the network configuration related to a network.
type NetworkConfig struct {
	NetNSPath     string
	NumInterfaces int
}

// Endpoint gathers a network pair and its properties.
type Endpoint struct {
	NetPair    NetworkInterfacePair
	Properties types.Result
}

// NetworkNamespace contains all data related to its network namespace.
type NetworkNamespace struct {
	NetNsPath string
	Endpoints []Endpoint
}

// NetworkModel describes the type of network specification.
type NetworkModel string

const (
	// NoopNetworkModel is the No-Op network.
	NoopNetworkModel NetworkModel = "noop"

	// CNINetworkModel is the CNI network.
	CNINetworkModel NetworkModel = "CNI"

	// CNMNetworkModel is the CNM network.
	CNMNetworkModel NetworkModel = "CNM"
)

// Set sets a network type based on the input string.
func (networkType *NetworkModel) Set(value string) error {
	switch value {
	case "noop":
		*networkType = NoopNetworkModel
		return nil
	case "CNI":
		*networkType = CNINetworkModel
		return nil
	case "CNM":
		*networkType = CNMNetworkModel
		return nil
	default:
		return fmt.Errorf("Unknown network type %s", value)
	}
}

// String converts a network type to a string.
func (networkType *NetworkModel) String() string {
	switch *networkType {
	case NoopNetworkModel:
		return string(NoopNetworkModel)
	case CNINetworkModel:
		return string(CNINetworkModel)
	case CNMNetworkModel:
		return string(CNMNetworkModel)
	default:
		return ""
	}
}

// newNetwork returns a network from a network type.
func newNetwork(networkType NetworkModel) network {
	switch networkType {
	case NoopNetworkModel:
		return &noopNetwork{}
	case CNINetworkModel:
		return &cni{}
	case CNMNetworkModel:
		return &cnm{}
	default:
		return &noopNetwork{}
	}
}

func bridgeNetworkPair(netPair NetworkInterfacePair) error {
	libsnnet.Logger = gloginterface.CiaoGlogLogger{}

	// new tap
	tapVnic, err := libsnnet.NewVnic(netPair.TAPIface.Name)
	if err != nil {
		return err
	}
	tapVnic.LinkName = netPair.TAPIface.Name

	// create tap
	err = tapVnic.Create()
	if err != nil {
		return err
	}

	// new veth
	virtVnic, err := libsnnet.NewContainerVnic(netPair.VirtIface.Name)
	if err != nil {
		return err
	}

	// create veth
	err = virtVnic.GetDeviceByName(netPair.VirtIface.Name)
	if err != nil {
		return err
	}

	// set veth MAC address
	hardAddr, err := net.ParseMAC(netPair.VirtIface.HardAddr)
	if err != nil {
		return err
	}
	err = virtVnic.SetHardwareAddr(hardAddr)
	if err != nil {
		return err
	}

	// new bridge
	bridge, err := libsnnet.NewBridge(netPair.Name)
	if err != nil {
		return err
	}
	bridge.LinkName = netPair.Name

	// create bridge
	err = bridge.Create()
	if err != nil {
		return err
	}

	// attach tap to bridge
	err = tapVnic.Attach(bridge)
	if err != nil {
		return err
	}

	// enable tap
	err = tapVnic.Enable()
	if err != nil {
		return err
	}

	// attach veth to bridge
	err = virtVnic.Attach(bridge)
	if err != nil {
		return err
	}

	// enable veth
	err = virtVnic.Enable()
	if err != nil {
		return err
	}

	// enable bridge
	err = bridge.Enable()
	if err != nil {
		return err
	}

	return nil
}

func unBridgeNetworkPair(netPair NetworkInterfacePair) error {
	libsnnet.Logger = gloginterface.CiaoGlogLogger{}

	// new tap
	tapVnic, err := libsnnet.NewVnic(netPair.TAPIface.Name)
	if err != nil {
		return err
	}

	// get tap
	err = tapVnic.GetDevice()
	if err != nil {
		return err
	}

	// new veth
	virtVnic, err := libsnnet.NewContainerVnic(netPair.VirtIface.Name)
	if err != nil {
		return err
	}

	// get veth
	err = virtVnic.GetDeviceByName(netPair.VirtIface.Name)
	if err != nil {
		return err
	}

	// new bridge
	bridge, err := libsnnet.NewBridge(netPair.Name)
	if err != nil {
		return err
	}

	// get bridge
	err = bridge.GetDevice()
	if err != nil {
		return err
	}

	// disable bridge
	err = bridge.Disable()
	if err != nil {
		return err
	}

	// disable veth
	err = virtVnic.Disable()
	if err != nil {
		return err
	}

	// detach veth from bridge
	err = virtVnic.Detach(bridge)
	if err != nil {
		return err
	}

	// disable tap
	err = tapVnic.Disable()
	if err != nil {
		return err
	}

	// detach tap from bridge
	err = tapVnic.Detach(bridge)
	if err != nil {
		return err
	}

	// destroy bridge
	err = bridge.Destroy()
	if err != nil {
		return err
	}

	// destroy tap
	err = tapVnic.Destroy()
	if err != nil {
		return err
	}

	return nil
}

func createNetNS() (string, error) {
	n, err := ns.NewNS()
	if err != nil {
		return "", err
	}

	return n.Path(), nil
}

func setNetNS(netNSPath string) error {
	n, err := ns.GetNS(netNSPath)
	if err != nil {
		return err
	}

	return n.Set()
}

func doNetNS(netNSPath string, cb func(ns.NetNS) error) error {
	n, err := ns.GetNS(netNSPath)
	if err != nil {
		return err
	}

	return n.Do(cb)
}

func deleteNetNS(netNSPath string, mounted bool) error {
	n, err := ns.GetNS(netNSPath)
	if err != nil {
		return err
	}

	err = n.Close()
	if err != nil {
		return err
	}

	// This unmount part is supposed to be done in the cni/ns package, but the "mounted"
	// flag is not updated when retrieving NetNs handler from GetNS().
	if mounted {
		if err = unix.Unmount(netNSPath, unix.MNT_DETACH); err != nil {
			return fmt.Errorf("Failed to unmount namespace %s: %v", netNSPath, err)
		}
		if err := os.RemoveAll(netNSPath); err != nil {
			return fmt.Errorf("Failed to clean up namespace %s: %v", netNSPath, err)
		}
	}

	return nil
}

func createNetworkEndpoint(idx int, uniqueID string, ifName string) Endpoint {
	hardAddr := net.HardwareAddr{0x02, 0x00, 0xCA, 0xFE, byte(idx >> 8), byte(idx)}

	endpoint := Endpoint{
		NetPair: NetworkInterfacePair{
			ID:   fmt.Sprintf("%s-%d", uniqueID, idx),
			Name: fmt.Sprintf("br%d", idx),
			VirtIface: NetworkInterface{
				Name:     fmt.Sprintf("eth%d", idx),
				HardAddr: hardAddr.String(),
			},
			TAPIface: NetworkInterface{
				Name: fmt.Sprintf("tap%d", idx),
			},
		},
	}

	if ifName != "" {
		endpoint.NetPair.VirtIface.Name = ifName
	}

	return endpoint
}

func createNetworkEndpoints(numOfEndpoints int) ([]Endpoint, error) {
	var endpoints []Endpoint

	if numOfEndpoints < 1 {
		return endpoints, fmt.Errorf("Invalid number of network endpoints")
	}

	uniqueID := uuid.Generate().String()

	for i := 0; i < numOfEndpoints; i++ {
		endpoints = append(endpoints, createNetworkEndpoint(i, uniqueID, ""))
	}

	return endpoints, nil
}

func addNetDevHypervisor(pod Pod, endpoints []Endpoint) error {
	return pod.hypervisor.addDevice(endpoints, netDev)
}

// network is the virtcontainers network interface.
// Container network plugins are used to setup virtual network
// between VM netns and the host network physical interface.
type network interface {
	// init initializes the network, setting a new network namespace.
	init(config *NetworkConfig) error

	// run runs a callback function in a specified network namespace.
	run(networkNSPath string, cb func() error) error

	// add adds all needed interfaces inside the network namespace.
	add(pod Pod, config NetworkConfig) (NetworkNamespace, error)

	// remove unbridges and deletes TAP interfaces. It also removes virtual network
	// interfaces and deletes the network namespace.
	remove(pod Pod, networkNS NetworkNamespace) error
}
