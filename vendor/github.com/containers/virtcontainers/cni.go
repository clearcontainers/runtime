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

	cniTypes "github.com/containernetworking/cni/pkg/types"
	cniV2Types "github.com/containernetworking/cni/pkg/types/020"
	cniLatestTypes "github.com/containernetworking/cni/pkg/types/current"
	cniPlugin "github.com/containers/virtcontainers/pkg/cni"
	"github.com/sirupsen/logrus"
)

// cni is a network implementation for the CNI plugin.
type cni struct{}

// Logger returns a logrus logger appropriate for logging cni messages
func (n *cni) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "cni")
}

func cniDNSToDNSInfo(cniDNS cniTypes.DNS) DNSInfo {
	return DNSInfo{
		Servers:  cniDNS.Nameservers,
		Domain:   cniDNS.Domain,
		Searches: cniDNS.Search,
		Options:  cniDNS.Options,
	}
}

func convertLatestCNIResult(result *cniLatestTypes.Result) NetworkInfo {
	return NetworkInfo{
		DNS: cniDNSToDNSInfo(result.DNS),
	}
}

func convertV2CNIResult(result *cniV2Types.Result) NetworkInfo {
	return NetworkInfo{
		DNS: cniDNSToDNSInfo(result.DNS),
	}
}

func convertCNIResult(cniResult cniTypes.Result) (NetworkInfo, error) {
	switch result := cniResult.(type) {
	case *cniLatestTypes.Result:
		return convertLatestCNIResult(result), nil
	case *cniV2Types.Result:
		return convertV2CNIResult(result), nil
	default:
		return NetworkInfo{}, fmt.Errorf("Unknown CNI result type %T", result)
	}
}

func (n *cni) addVirtInterfaces(networkNS *NetworkNamespace) error {
	netPlugin, err := cniPlugin.NewNetworkPlugin()
	if err != nil {
		return err
	}

	for idx, endpoint := range networkNS.Endpoints {
		virtualEndpoint, ok := endpoint.(*VirtualEndpoint)
		if !ok {
			continue
		}

		result, err := netPlugin.AddNetwork(virtualEndpoint.NetPair.ID, networkNS.NetNsPath, virtualEndpoint.Name())
		if err != nil {
			return err
		}

		netInfo, err := convertCNIResult(result)
		if err != nil {
			return err
		}

		networkNS.Endpoints[idx].SetProperties(netInfo)

		n.Logger().Infof("AddNetwork results %s", result.String())
	}

	return nil
}

func (n *cni) deleteVirtInterfaces(networkNS NetworkNamespace) error {
	netPlugin, err := cniPlugin.NewNetworkPlugin()
	if err != nil {
		return err
	}

	for _, endpoint := range networkNS.Endpoints {
		virtualEndpoint, ok := endpoint.(*VirtualEndpoint)
		if !ok {
			continue
		}

		err := netPlugin.RemoveNetwork(virtualEndpoint.NetPair.ID, networkNS.NetNsPath, virtualEndpoint.NetPair.VirtIface.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *cni) updateEndpointsFromScan(networkNS *NetworkNamespace) error {
	endpoints, err := createEndpointsFromScan(networkNS.NetNsPath)
	if err != nil {
		return err
	}

	for _, endpoint := range endpoints {
		for _, ep := range networkNS.Endpoints {
			if ep.Name() == endpoint.Name() {
				// Update endpoint properties with info from
				// the scan. Do not update DNS since the scan
				// cannot provide it.
				prop := endpoint.Properties()
				prop.DNS = ep.Properties().DNS
				endpoint.SetProperties(prop)

				switch e := endpoint.(type) {
				case *VirtualEndpoint:
					e.NetPair = ep.(*VirtualEndpoint).NetPair
				}
				break
			}
		}
	}

	networkNS.Endpoints = endpoints

	return nil
}

// init initializes the network, setting a new network namespace for the CNI network.
func (n *cni) init(config NetworkConfig) (string, bool, error) {
	return initNetworkCommon(config)
}

// run runs a callback in the specified network namespace.
func (n *cni) run(networkNSPath string, cb func() error) error {
	return runNetworkCommon(networkNSPath, cb)
}

// add adds all needed interfaces inside the network namespace for the CNI network.
func (n *cni) add(pod Pod, config NetworkConfig, netNsPath string, netNsCreated bool) (NetworkNamespace, error) {
	endpoints, err := createNetworkEndpoints(config.NumInterfaces)
	if err != nil {
		return NetworkNamespace{}, err
	}

	networkNS := NetworkNamespace{
		NetNsPath:    netNsPath,
		NetNsCreated: netNsCreated,
		Endpoints:    endpoints,
	}

	if err := n.addVirtInterfaces(&networkNS); err != nil {
		return NetworkNamespace{}, err
	}

	if err := n.updateEndpointsFromScan(&networkNS); err != nil {
		return NetworkNamespace{}, err
	}

	if err := addNetworkCommon(pod, &networkNS); err != nil {
		return NetworkNamespace{}, err
	}

	return networkNS, nil
}

// remove unbridges and deletes TAP interfaces. It also removes virtual network
// interfaces and deletes the network namespace for the CNI network.
func (n *cni) remove(pod Pod, networkNS NetworkNamespace) error {
	if err := removeNetworkCommon(networkNS); err != nil {
		return err
	}

	if err := n.deleteVirtInterfaces(networkNS); err != nil {
		return err
	}

	return deleteNetNS(networkNS.NetNsPath, true)
}
