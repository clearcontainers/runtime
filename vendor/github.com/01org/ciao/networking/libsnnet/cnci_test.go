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

package libsnnet

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cnciTestInit() (*Cnci, error) {
	snTestInit()

	_, testNet, err := net.ParseCIDR(snTestNet)
	if err != nil {
		return nil, err
	}

	netConfig := &NetworkConfig{
		ManagementNet: []net.IPNet{*testNet},
		ComputeNet:    []net.IPNet{*testNet},
		Mode:          GreTunnel,
	}

	cnci := &Cnci{
		ID:            "TestCNUUID",
		NetworkConfig: netConfig,
	}

	if err := cnci.Init(); err != nil {
		return nil, err
	}
	if err := cnci.RebuildTopology(); err != nil {
		return nil, err
	}

	return cnci, nil

}

//Tests all CNCI APIs
//
//Tests all operations typically performed on a CNCI
//Test includes adding and deleting a remote subnet
//Rebuild of the topology database (to simulate agent crash)
//It also tests the reset of the node to clean status
//
//Test should pass ok
func TestCNCI_Init(t *testing.T) {
	assert := assert.New(t)
	cnci, err := cnciTestInit()
	require.Nil(t, err)

	_, tnet, _ := net.ParseCIDR("192.168.0.0/24")

	_, err = cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102"))
	assert.Nil(err)

	//Duplicate
	_, err = cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102"))
	assert.Nil(err)

	_, err = cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.103"))
	assert.Nil(err)

	_, err = cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.104"))
	assert.Nil(err)

	assert.Nil(cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102")))

	err = cnci.RebuildTopology()
	require.Nil(t, err)

	//Duplicate
	assert.Nil(cnci.RebuildTopology())

	_, err = cnci.AddRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.105"))
	assert.Nil(err)

	assert.Nil(cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.103")))

	assert.Nil(cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.105")))

	//Duplicate
	assert.Nil(cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.105")))

	assert.Nil(cnci.DelRemoteSubnet(*tnet, 1234, net.ParseIP("192.168.0.102")))
	assert.Nil(cnci.Shutdown())
	//Duplicate
	assert.Nil(cnci.Shutdown())
}

//Whitebox test case of CNCI API primitives
//
//This tests ensure that the lower level primitive
//APIs that the CNCI uses are still sane and function
//as expected. This test is expected to catch any
//issues due to change in the underlying libraries
//kernel features and applications (like dnsmasq,
//netlink) that the CNCI API relies on
//The goal of this test is to ensure we can rebase our
//dependencies and catch any dependency errors
//
//Test is expected to pass
func TestCNCI_Internal(t *testing.T) {
	assert := assert.New(t)

	// Typical inputs in YAML
	tenantUUID := "tenantUuid"
	concUUID := "concUuid"
	cnUUID := "cnUuid"
	subnetUUID := "subnetUuid"
	subnetKey := uint32(0xF)
	reserved := 10
	cnciIP := net.IPv4(127, 0, 0, 1)
	subnet := net.IPNet{
		IP:   net.IPv4(192, 168, 1, 0),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	// +The DHCP configuration, MAC to IP mapping is another inputs
	// This will be sent a-priori or based on design each time an instance is created

	// Create the CNCI aggregation bridge
	bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
	bridge, _ := NewBridge(bridgeAlias)

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	assert.Nil(bridge.Enable())

	// Attach the DNS masq against the CNCI bridge. This gives it an IP address
	d, err := newDnsmasq(bridgeAlias, tenantUUID, subnet, reserved, bridge)
	assert.Nil(err)

	assert.Nil(d.start())
	defer func() { _ = d.stop() }()

	// At this time the bridge is ready waiting for tunnels to be created
	// The next step will happen each time a tenant bridge is created for
	// this tenant on a CN
	cnIP := net.IPv4(127, 0, 0, 1)

	// Wait on SNTP messages requesting tunnel creation
	// Create a GRE tunnel that connects a tenant bridge to the CNCI bridge
	// for that subnet. The CNCI will have many bridges one for each subnet
	// the belongs to the tenant
	greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, cnUUID)
	local := cnciIP
	remote := cnIP
	key := subnetKey

	gre, _ := newGreTunEP(greAlias, local, remote, key)

	assert.Nil(gre.create())
	defer func() { _ = gre.destroy() }()

	assert.Nil(gre.attach(bridge))
	assert.Nil(gre.enable())
}
