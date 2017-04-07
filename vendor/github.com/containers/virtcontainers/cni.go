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
	"github.com/containernetworking/cni/pkg/ns"
	cniPlugin "github.com/containers/virtcontainers/pkg/cni"
)

// cni is a network implementation for the CNI plugin.
type cni struct{}

func (n *cni) addVirtInterfaces(networkNS *NetworkNamespace) error {
	netPlugin, err := cniPlugin.NewNetworkPlugin()
	if err != nil {
		return err
	}

	for idx, endpoint := range networkNS.Endpoints {
		result, err := netPlugin.AddNetwork(endpoint.NetPair.ID, networkNS.NetNsPath, endpoint.NetPair.VirtIface.Name)
		if err != nil {
			return err
		}

		networkNS.Endpoints[idx].Properties = *result

		virtLog.Infof("AddNetwork results %v", *result)
	}

	return nil
}

func (n *cni) deleteVirtInterfaces(networkNS NetworkNamespace) error {
	netPlugin, err := cniPlugin.NewNetworkPlugin()
	if err != nil {
		return err
	}

	for _, endpoint := range networkNS.Endpoints {
		err := netPlugin.RemoveNetwork(endpoint.NetPair.ID, networkNS.NetNsPath, endpoint.NetPair.VirtIface.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// init initializes the network, setting a new network namespace for the CNI network.
func (n *cni) init(config *NetworkConfig) error {
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
// run does not switch the current process to the specified network namespace
// for the CNI network. Indeed, the switch will occur in the add() and remove()
// functions instead.
func (n *cni) run(networkNSPath string, cb func() error) error {
	return doNetNS(networkNSPath, func(_ ns.NetNS) error {
		return cb()
	})
}

// add adds all needed interfaces inside the network namespace for the CNI network.
func (n *cni) add(pod Pod, config NetworkConfig) (NetworkNamespace, error) {
	endpoints, err := createNetworkEndpoints(config.NumInterfaces)
	if err != nil {
		return NetworkNamespace{}, err
	}

	networkNS := NetworkNamespace{
		NetNsPath: config.NetNSPath,
		Endpoints: endpoints,
	}

	err = n.addVirtInterfaces(&networkNS)
	if err != nil {
		return NetworkNamespace{}, err
	}

	err = doNetNS(networkNS.NetNsPath, func(_ ns.NetNS) error {
		for _, endpoint := range networkNS.Endpoints {
			err = bridgeNetworkPair(endpoint.NetPair)
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
// interfaces and deletes the network namespace for the CNI network.
func (n *cni) remove(pod Pod, networkNS NetworkNamespace) error {
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

	err = n.deleteVirtInterfaces(networkNS)
	if err != nil {
		return err
	}

	err = deleteNetNS(networkNS.NetNsPath, true)
	if err != nil {
		return err
	}

	return nil
}
