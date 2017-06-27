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
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/01org/ciao/ssntp/uuid"
	types "github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Introduces constants related to network routes.
const (
	defaultRouteDest  = "0.0.0.0/0"
	defaultRouteLabel = "default"
)

type netIfaceAddrs struct {
	iface net.Interface
	addrs []net.Addr
}

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
	NetNsPath    string
	NetNsCreated bool
	Endpoints    []Endpoint
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

func initNetworkCommon(config NetworkConfig) (string, bool, error) {
	if config.NetNSPath == "" {
		path, err := createNetNS()
		if err != nil {
			return "", false, err
		}

		return path, true, nil
	}

	return config.NetNSPath, false, nil
}

func runNetworkCommon(networkNSPath string, cb func() error) error {
	if networkNSPath == "" {
		return fmt.Errorf("networkNSPath cannot be empty")
	}

	return doNetNS(networkNSPath, func(_ ns.NetNS) error {
		return cb()
	})
}

func addNetworkCommon(pod Pod, networkNS *NetworkNamespace) error {
	err := doNetNS(networkNS.NetNsPath, func(_ ns.NetNS) error {
		for idx := range networkNS.Endpoints {
			if err := bridgeNetworkPair(&(networkNS.Endpoints[idx].NetPair)); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return addNetDevHypervisor(pod, networkNS.Endpoints)
}

func removeNetworkCommon(networkNS NetworkNamespace) error {
	return doNetNS(networkNS.NetNsPath, func(_ ns.NetNS) error {
		for _, endpoint := range networkNS.Endpoints {
			err := unBridgeNetworkPair(endpoint.NetPair)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func createLink(netHandle *netlink.Handle, name string, expectedLink netlink.Link) (netlink.Link, error) {
	var newLink netlink.Link

	switch expectedLink.Type() {
	case (&netlink.Bridge{}).Type():
		newLink = &netlink.Bridge{
			LinkAttrs:         netlink.LinkAttrs{Name: name},
			MulticastSnooping: expectedLink.(*netlink.Bridge).MulticastSnooping,
		}
	case (&netlink.Tuntap{}).Type():
		newLink = &netlink.Tuntap{
			LinkAttrs: netlink.LinkAttrs{Name: name},
			Mode:      netlink.TUNTAP_MODE_TAP,
		}
	default:
		return nil, fmt.Errorf("Unsupported link type %s", expectedLink.Type())
	}

	if err := netHandle.LinkAdd(newLink); err != nil {
		return nil, fmt.Errorf("LinkAdd() failed for %s name %s: %s", expectedLink.Type(), name, err)
	}

	return getLinkByName(netHandle, name, expectedLink)
}

func getLinkByName(netHandle *netlink.Handle, name string, expectedLink netlink.Link) (netlink.Link, error) {
	link, err := netHandle.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("LinkByName() failed for %s name %s: %s", expectedLink.Type(), name, err)
	}

	switch expectedLink.Type() {
	case (&netlink.Bridge{}).Type():
		if l, ok := link.(*netlink.Bridge); ok {
			return l, nil
		}
	case (&netlink.Tuntap{}).Type():
		if l, ok := link.(*netlink.GenericLink); ok {
			return l, nil
		}
	case (&netlink.Veth{}).Type():
		if l, ok := link.(*netlink.Veth); ok {
			return l, nil
		}
	default:
		return nil, fmt.Errorf("Unsupported link type %s", expectedLink.Type())
	}

	return nil, fmt.Errorf("Incorrect link type %s, expecting %s", link.Type(), expectedLink.Type())
}

func bridgeNetworkPair(netPair *NetworkInterfacePair) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	tapLink, err := createLink(netHandle, netPair.TAPIface.Name, &netlink.Tuntap{})
	if err != nil {
		return fmt.Errorf("Could not create TAP interface: %s", err)
	}

	vethLink, err := getLinkByName(netHandle, netPair.VirtIface.Name, &netlink.Veth{})
	if err != nil {
		return fmt.Errorf("Could not get veth interface: %s", err)
	}

	vethLinkAttrs := vethLink.Attrs()

	// Save the veth MAC address to the TAP so that it can later be used
	// to build the hypervisor command line. This MAC address has to be
	// the one inside the VM in order to avoid any firewall issues. The
	// bridge created by the network plugin on the host actually expects
	// to see traffic from this MAC address and not another one.
	netPair.TAPIface.HardAddr = vethLinkAttrs.HardwareAddr.String()

	if err := netHandle.LinkSetMTU(tapLink, vethLinkAttrs.MTU); err != nil {
		return fmt.Errorf("Could not set TAP MTU %d: %s", vethLinkAttrs.MTU, err)
	}

	hardAddr, err := net.ParseMAC(netPair.VirtIface.HardAddr)
	if err != nil {
		return err
	}
	if err := netHandle.LinkSetHardwareAddr(vethLink, hardAddr); err != nil {
		return fmt.Errorf("Could not set MAC address %s for veth interface %s: %s",
			netPair.VirtIface.HardAddr, netPair.VirtIface.Name, err)
	}

	mcastSnoop := false
	bridgeLink, err := createLink(netHandle, netPair.Name, &netlink.Bridge{MulticastSnooping: &mcastSnoop})
	if err != nil {
		return fmt.Errorf("Could not create bridge: %s", err)
	}

	if err := netHandle.LinkSetMaster(tapLink, bridgeLink.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("Could not attach TAP %s to the bridge %s: %s",
			netPair.TAPIface.Name, netPair.Name, err)
	}

	if err := netHandle.LinkSetUp(tapLink); err != nil {
		return fmt.Errorf("Could not enable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkSetMaster(vethLink, bridgeLink.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("Could not attach veth %s to the bridge %s: %s",
			netPair.VirtIface.Name, netPair.Name, err)
	}

	if err := netHandle.LinkSetUp(vethLink); err != nil {
		return fmt.Errorf("Could not enable veth %s: %s", netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetUp(bridgeLink); err != nil {
		return fmt.Errorf("Could not enable bridge %s: %s", netPair.Name, err)
	}

	return nil
}

func unBridgeNetworkPair(netPair NetworkInterfacePair) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	tapLink, err := getLinkByName(netHandle, netPair.TAPIface.Name, &netlink.Tuntap{})
	if err != nil {
		return fmt.Errorf("Could not get TAP interface: %s", err)
	}

	vethLink, err := getLinkByName(netHandle, netPair.VirtIface.Name, &netlink.Veth{})
	if err != nil {
		return fmt.Errorf("Could not get veth interface: %s", err)
	}

	bridgeLink, err := getLinkByName(netHandle, netPair.Name, &netlink.Bridge{})
	if err != nil {
		return fmt.Errorf("Could not get bridge interface: %s", err)
	}

	if err := netHandle.LinkSetDown(bridgeLink); err != nil {
		return fmt.Errorf("Could not disable bridge %s: %s", netPair.Name, err)
	}

	if err := netHandle.LinkSetDown(vethLink); err != nil {
		return fmt.Errorf("Could not disable veth %s: %s", netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetNoMaster(vethLink); err != nil {
		return fmt.Errorf("Could not detach veth %s: %s", netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetDown(tapLink); err != nil {
		return fmt.Errorf("Could not disable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkSetNoMaster(tapLink); err != nil {
		return fmt.Errorf("Could not detach TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkDel(bridgeLink); err != nil {
		return fmt.Errorf("Could not remove bridge %s: %s", netPair.Name, err)
	}

	if err := netHandle.LinkDel(tapLink); err != nil {
		return fmt.Errorf("Could not remove TAP %s: %s", netPair.TAPIface.Name, err)
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

// doNetNS is free from any call to a go routine, and it calls
// into runtime.LockOSThread(), meaning it won't be executed in a
// different thread than the one expected by the caller.
func doNetNS(netNSPath string, cb func(ns.NetNS) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	currentNS, err := ns.GetCurrentNS()
	if err != nil {
		return err
	}
	defer currentNS.Close()

	targetNS, err := ns.GetNS(netNSPath)
	if err != nil {
		return err
	}

	if err := targetNS.Set(); err != nil {
		return err
	}
	defer currentNS.Set()

	return cb(targetNS)
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

func createNetworkEndpoint(idx int, uniqueID string, ifName string) (Endpoint, error) {
	if idx < 0 {
		return Endpoint{}, fmt.Errorf("invalid network endpoint index: %d", idx)
	}
	if uniqueID == "" {
		return Endpoint{}, errors.New("uniqueID cannot be blank")
	}

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

	return endpoint, nil
}

func createNetworkEndpoints(numOfEndpoints int) (endpoints []Endpoint, err error) {
	if numOfEndpoints < 1 {
		return endpoints, fmt.Errorf("Invalid number of network endpoints")
	}

	uniqueID := uuid.Generate().String()

	for i := 0; i < numOfEndpoints; i++ {
		endpoint, err := createNetworkEndpoint(i, uniqueID, "")
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func getIfacesFromNetNs(networkNSPath string) ([]netIfaceAddrs, error) {
	var netIfaces []netIfaceAddrs

	if networkNSPath == "" {
		return []netIfaceAddrs{}, fmt.Errorf("Network namespace path cannot be empty")
	}

	err := doNetNS(networkNSPath, func(_ ns.NetNS) error {
		ifaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				return err
			}

			netIface := netIfaceAddrs{
				iface: iface,
				addrs: addrs,
			}

			netIfaces = append(netIfaces, netIface)
		}

		return nil
	})
	if err != nil {
		return []netIfaceAddrs{}, err
	}

	return netIfaces, nil
}

func getNetIfaceByName(name string, netIfaces []netIfaceAddrs) (net.Interface, error) {
	for _, netIface := range netIfaces {
		if netIface.iface.Name == name {
			return netIface.iface, nil
		}
	}

	return net.Interface{}, fmt.Errorf("Could not find the interface %s in the list", name)
}

func addNetDevHypervisor(pod Pod, endpoints []Endpoint) error {
	return pod.hypervisor.addDevice(endpoints, netDev)
}

// network is the virtcontainers network interface.
// Container network plugins are used to setup virtual network
// between VM netns and the host network physical interface.
type network interface {
	// init initializes the network, setting a new network namespace.
	init(config NetworkConfig) (string, bool, error)

	// run runs a callback function in a specified network namespace.
	run(networkNSPath string, cb func() error) error

	// add adds all needed interfaces inside the network namespace.
	add(pod Pod, config NetworkConfig, netNsPath string, netNsCreated bool) (NetworkNamespace, error)

	// remove unbridges and deletes TAP interfaces. It also removes virtual network
	// interfaces and deletes the network namespace.
	remove(pod Pod, networkNS NetworkNamespace) error
}
