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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func performGreOps(shouldPass bool, assert *assert.Assertions, gre *GreTunEP) {
	a := assert.Nil
	if !shouldPass {
		a = assert.NotNil
	}
	a(gre.enable())
	a(gre.disable())
	a(gre.destroy())
}

//Test all GRE tunnel primitives
//
//Tests create, enable, disable and destroy of GRE tunnels
//Failure indicates changes in netlink or kernel and in some
//case pre-existing tunnels on the test node. Ensure that
//there are no existing conflicting tunnels before running
//this test
//
//Test is expected to pass
func TestGre_Basic(t *testing.T) {
	assert := assert.New(t)
	id := "testgretap"
	local := net.ParseIP("127.0.0.1")
	remote := local
	key := uint32(0xF)

	gre, err := newGreTunEP(id, local, remote, key)
	assert.Nil(err)

	assert.Nil(gre.create())
	assert.Nil(gre.getDevice())
	performGreOps(true, assert, gre)
	assert.NotNil(gre.destroy())
}

//Test GRE tunnel bridge interactions
//
//Test all bridge, gre tunnel interactions including
//attach, detach, enable, disable, destroy
//
//Test is expected to pass
func TestGre_Bridge(t *testing.T) {
	assert := assert.New(t)
	id := "testgretap"
	local := net.ParseIP("127.0.0.1")
	remote := local
	key := uint32(0xF)

	gre, err := newGreTunEP(id, local, remote, key)
	assert.Nil(err)
	bridge, err := NewBridge("testbridge")
	assert.Nil(err)

	assert.Nil(gre.create())
	defer func() { _ = gre.destroy() }()

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	assert.Nil(gre.attach(bridge))
	//Duplicate
	assert.Nil(gre.attach(bridge))
	assert.Nil(gre.enable())
	assert.Nil(bridge.Enable())
	assert.Nil(gre.detach(bridge))
	//Duplicate
	assert.Nil(gre.detach(bridge))
}

//Tests failure paths in the GRE tunnel
//
//Tests failure paths in the GRE tunnel
//
//Test is expected to pass
func TestGre_Negative(t *testing.T) {
	assert := assert.New(t)
	id := "testgretap"
	local := net.ParseIP("127.0.0.1")
	remote := local
	key := uint32(0xF)

	gre, err := newGreTunEP(id, local, remote, key)
	assert.Nil(err)
	greDupl, err := newGreTunEP(id, local, remote, key)
	assert.Nil(err)

	assert.Nil(gre.create())
	assert.NotNil(greDupl.create())

	performGreOps(false, assert, greDupl)
	performGreOps(true, assert, gre)
}
