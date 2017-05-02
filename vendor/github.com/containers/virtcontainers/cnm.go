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

	"github.com/01org/ciao/ssntp/uuid"
	"github.com/containernetworking/cni/pkg/ns"
	cniTypes "github.com/containernetworking/cni/pkg/types"
	types "github.com/containernetworking/cni/pkg/types/current"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// cnm is a network implementation for the CNM plugin.
type cnm struct {
	config NetworkConfig
}

func (n *cnm) getNetIfaceRoutesWithinNetNs(networkNSPath string, ifaceName string) ([]netlink.Route, error) {
	if networkNSPath == "" {
		return []netlink.Route{}, fmt.Errorf("Network namespace path cannot be empty")
	}

	netnsHandle, err := netns.GetFromPath(networkNSPath)
	if err != nil {
		return []netlink.Route{}, err
	}
	defer netnsHandle.Close()

	netHandle, err := netlink.NewHandleAt(netnsHandle)
	if err != nil {
		return []netlink.Route{}, err
	}
	defer netHandle.Delete()

	link, err := netHandle.LinkByName(ifaceName)
	if err != nil {
		return []netlink.Route{}, err
	}

	routes, err := netHandle.RouteList(link, netlink.FAMILY_ALL)
	if err != nil {
		return []netlink.Route{}, err
	}

	return routes, nil
}

func (n *cnm) createResult(iface net.Interface, addrs []net.Addr, routes []netlink.Route) (types.Result, error) {
	var ipConfigs []*types.IPConfig
	for _, addr := range addrs {
		ip, ipNet, err := net.ParseCIDR(addr.String())
		if err != nil {
			return types.Result{}, err
		}

		version := "6"
		if ip.To4() != nil {
			version = "4"
		}
		ipNet.IP = ip

		ipConfig := &types.IPConfig{
			Version:   version,
			Interface: iface.Index,
			Address:   *ipNet,
		}

		ipConfigs = append(ipConfigs, ipConfig)
	}

	ifaceList := []*types.Interface{
		{
			Name: iface.Name,
			Mac:  iface.HardwareAddr.String(),
		},
	}

	var resultRoutes []*cniTypes.Route
	for _, route := range routes {
		if route.Dst == nil {
			continue
		}

		r := &cniTypes.Route{
			Dst: *(route.Dst),
			GW:  route.Gw,
		}

		resultRoutes = append(resultRoutes, r)
	}

	res := types.Result{
		Interfaces: ifaceList,
		IPs:        ipConfigs,
		Routes:     resultRoutes,
	}

	return res, nil
}

func (n *cnm) createEndpointsFromScan(networkNSPath string) ([]Endpoint, error) {
	var endpoints []Endpoint

	netIfaces, err := getIfacesFromNetNs(networkNSPath)
	if err != nil {
		return []Endpoint{}, err
	}

	uniqueID := uuid.Generate().String()

	idx := 0
	for _, netIface := range netIfaces {
		var endpoint Endpoint

		if netIface.iface.Name == "lo" {
			continue
		} else {
			endpoint, err = createNetworkEndpoint(idx, uniqueID, netIface.iface.Name)
			if err != nil {
				return []Endpoint{}, err
			}
		}

		routes, err := n.getNetIfaceRoutesWithinNetNs(networkNSPath, netIface.iface.Name)
		if err != nil {
			return []Endpoint{}, err
		}

		endpoint.Properties, err = n.createResult(netIface.iface, netIface.addrs, routes)
		if err != nil {
			return []Endpoint{}, err
		}

		endpoints = append(endpoints, endpoint)

		idx++
	}

	return endpoints, nil
}

// init initializes the network, setting a new network namespace for the CNM network.
func (n *cnm) init(config *NetworkConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be empty")
	}

	if config.NetNSPath == "" {
		path, err := createNetNS()
		if err != nil {
			return err
		}

		config.NetNSPath = path
	}

	return nil
}

// run runs a callback in the specified network namespace.
func (n *cnm) run(networkNSPath string, cb func() error) error {
	if networkNSPath == "" {
		return fmt.Errorf("networkNSPath cannot be empty")
	}

	return doNetNS(networkNSPath, func(_ ns.NetNS) error {
		return cb()
	})
}

// add adds all needed interfaces inside the network namespace for the CNM network.
func (n *cnm) add(pod Pod, config NetworkConfig) (NetworkNamespace, error) {
	endpoints, err := n.createEndpointsFromScan(config.NetNSPath)
	if err != nil {
		return NetworkNamespace{}, err
	}

	networkNS := NetworkNamespace{
		NetNsPath: config.NetNSPath,
		Endpoints: endpoints,
	}

	err = doNetNS(networkNS.NetNsPath, func(_ ns.NetNS) error {
		for _, endpoint := range networkNS.Endpoints {
			err := bridgeNetworkPair(endpoint.NetPair)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return NetworkNamespace{}, err
	}

	err = addNetDevHypervisor(pod, networkNS.Endpoints)
	if err != nil {
		return NetworkNamespace{}, err
	}

	return networkNS, nil
}

// remove unbridges and deletes TAP interfaces. It also removes virtual network
// interfaces and deletes the network namespace for the CNM network.
func (n *cnm) remove(pod Pod, networkNS NetworkNamespace) error {
	err := doNetNS(networkNS.NetNsPath, func(_ ns.NetNS) error {
		for _, endpoint := range networkNS.Endpoints {
			err := unBridgeNetworkPair(endpoint.NetPair)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = deleteNetNS(networkNS.NetNsPath, true)
	if err != nil {
		return err
	}

	return nil
}
