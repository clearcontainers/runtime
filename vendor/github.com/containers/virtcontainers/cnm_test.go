//
// Copyright (c) 2017 Intel Corporation
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
	"reflect"
	"testing"

	cniTypes "github.com/containernetworking/cni/pkg/types"
	types "github.com/containernetworking/cni/pkg/types/current"
	"github.com/vishvananda/netlink"
)

type mockAddr struct {
	network string
	ipAddr  string
}

func (m mockAddr) Network() string {
	return m.network
}

func (m mockAddr) String() string {
	return m.ipAddr
}

func TestCNMCreateResults(t *testing.T) {
	plugin := &cnm{}
	ifaceIdx := 0

	macAddr := net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	ip, ipNet, err := net.ParseCIDR("192.168.0.1/8")
	if err != nil {
		t.Fatal()
	}
	ipNet.IP = ip

	expected := types.Result{
		Interfaces: []*types.Interface{
			{
				Name: "eth0",
				Mac:  macAddr.String(),
			},
		},
		IPs: []*types.IPConfig{
			{
				Version:   "4",
				Interface: &ifaceIdx,
				Address:   *ipNet,
			},
		},
		Routes: []*cniTypes.Route{
			{
				Dst: *ipNet,
				GW:  ipNet.IP,
			},
		},
	}

	iface := net.Interface{
		Index:        0,
		Name:         "eth0",
		HardwareAddr: macAddr,
	}

	addr := mockAddr{
		network: "test-network",
		ipAddr:  "192.168.0.1/8",
	}

	addrs := []net.Addr{net.Addr(addr)}

	routes := []netlink.Route{
		{
			Dst: ipNet,
			Gw:  ipNet.IP,
		},
	}

	result, err := plugin.createResult(iface, addrs, routes)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}
