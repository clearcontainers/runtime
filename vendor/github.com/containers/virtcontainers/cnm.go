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
	"net"

	"github.com/01org/ciao/ssntp/uuid"
	"github.com/containernetworking/cni/pkg/ns"
	types "github.com/containernetworking/cni/pkg/types/current"
)

// cnm is a network implementation for the CNM plugin.
type cnm struct {
	config NetworkConfig
}

func (n *cnm) createResult(iface net.Interface, addrs []net.Addr) (types.Result, error) {
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

	res := types.Result{
		Interfaces: ifaceList,
		IPs:        ipConfigs,
	}

	return res, nil
}

func (n *cnm) createEndpointsFromScan() ([]Endpoint, error) {
	var endpoints []Endpoint

	ifaces, err := net.Interfaces()
	if err != nil {
		return []Endpoint{}, err
	}

	uniqueID := uuid.Generate().String()

	idx := 0
	for _, iface := range ifaces {
		var endpoint Endpoint

		addrs, err := iface.Addrs()
		if err != nil {
			return []Endpoint{}, err
		}

		if iface.Name == "lo" {
			continue
		} else {
			endpoint = createNetworkEndpoint(idx, uniqueID, iface.Name)
		}

		endpoint.Properties, err = n.createResult(iface, addrs)
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
	return doNetNS(networkNSPath, func(_ ns.NetNS) error {
		return cb()
	})
}

// add adds all needed interfaces inside the network namespace for the CNM network.
func (n *cnm) add(pod Pod, config NetworkConfig) (NetworkNamespace, error) {
	endpoints, err := n.createEndpointsFromScan()
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
