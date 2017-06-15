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

// Internal tests (whitebox) for libsnnet
package libsnnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

//Tests the implementation of the db rebuild from aliases
//
//This test uses a mix of primitives and APIs to check
//the reliability of the dbRebuild API
//
//The test is expected to pass
func TestCN_dbRebuild(t *testing.T) {
	assert := assert.New(t)
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	cn := &ComputeNode{}
	alias := genCnVnicAliases(vnicCfg)

	bridgeAlias := alias.bridge
	vnicAlias := alias.vnic
	greAlias := alias.gre

	bridge, _ := NewBridge(bridgeAlias)

	if assert.NotNil(bridge.GetDevice()) {
		// First instance to land, create the bridge and tunnel
		assert.Nil(bridge.Create())
		defer func(b *Bridge) { _ = b.Destroy() }(bridge)

		// Create the tunnel to connect to the CNCI
		local := vnicCfg.VnicIP //Fake it for now
		remote := vnicCfg.ConcIP
		subnetKey := vnicCfg.SubnetKey

		gre, _ := newGreTunEP(greAlias, local, remote, uint32(subnetKey))

		assert.Nil(gre.create())
		defer func() { _ = gre.destroy() }()

		assert.Nil(gre.attach(bridge))
	}

	// Create the VNIC for the instance
	vnic, _ := NewVnic(vnicAlias)

	assert.Nil(vnic.Create())
	defer func() { _ = vnic.Destroy() }()

	assert.Nil(vnic.Attach(bridge))

	//Add a second vnic
	vnicCfg.VnicIP = net.IPv4(192, 168, 1, 101)
	alias1 := genCnVnicAliases(vnicCfg)
	vnic1, _ := NewVnic(alias1.vnic)

	assert.Nil(vnic1.Create())
	defer func() { _ = vnic1.Destroy() }()

	assert.Nil(vnic1.Attach(bridge))

	/* Test negative test cases */
	assert.NotNil(cn.DbRebuild(nil))
	cn.NetworkConfig = &NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          GreTunnel,
	}

	assert.NotNil(cn.DbRebuild(nil))

	/* Test positive */
	cn.cnTopology = &cnTopology{
		bridgeMap: make(map[string]map[string]bool),
		linkMap:   make(map[string]*linkInfo),
		nameMap:   make(map[string]bool),
	}

	assert.Nil(cn.DbRebuild(nil))

	cnt, err := cn.dbUpdate(alias.bridge, alias1.vnic, dbDelVnic)
	if assert.Nil(err) {
		assert.Equal(cnt, 1)
	}

	cnt, err = cn.dbUpdate(alias.bridge, alias.vnic, dbDelVnic)
	if assert.Nil(err) {
		assert.Equal(cnt, 0)
	}

	cnt, err = cn.dbUpdate(alias.bridge, "", dbDelBr)
	if assert.Nil(err) {
		assert.Equal(cnt, 0)
	}

	cnt, err = cn.dbUpdate(alias.bridge, "", dbInsBr)
	if assert.Nil(err) {
		assert.Equal(cnt, 1)
	}

	cnt, err = cn.dbUpdate(alias.bridge, alias.vnic, dbInsVnic)
	if assert.Nil(err) {
		assert.Equal(cnt, 1)
	}

	cnt, err = cn.dbUpdate(alias.bridge, alias1.vnic, dbInsVnic)
	if assert.Nil(err) {
		assert.Equal(cnt, 2)
	}

	//Negative tests
	_, err = cn.dbUpdate(alias.bridge, alias1.vnic, dbInsVnic)
	assert.NotNil(err)

	_, err = cn.dbUpdate(alias.bridge, "", dbInsBr)
	assert.NotNil(err)
}
