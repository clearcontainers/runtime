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
	"testing"

	"github.com/stretchr/testify/assert"
)

func performVnicOps(shouldPass bool, assert *assert.Assertions, vnic *Vnic) {
	a := assert.Nil
	if !shouldPass {
		a = assert.NotNil
	}
	a(vnic.Enable())
	a(vnic.Disable())
	a(vnic.Destroy())
}

//Tests all the basic VNIC primitives
//
//Tests for creation, get, enable, disable and destroy
//primitives. If any of these fail, it may be a issue
//with the underlying netlink or kernel dependencies
//
//Test is expected to pass
func TestVnic_Basic(t *testing.T) {
	assert := assert.New(t)

	vnic, err := NewVnic("testvnic")
	assert.Nil(err)
	assert.Nil(vnic.Create())

	vnic1, err := NewVnic("testvnic")
	assert.Nil(err)

	assert.Nil(vnic1.GetDevice())
	assert.NotEqual(vnic.InterfaceName(), "")
	assert.Equal(vnic.PeerName(), vnic.InterfaceName())

	performVnicOps(true, assert, vnic)
}

//Tests all the basic Container VNIC primitives
//
//Tests for creation, get, enable, disable and destroy
//primitives. If any of these fail, it may be a issue
//with the underlying netlink or kernel dependencies
//
//Test is expected to pass
func TestVnicContainer_Basic(t *testing.T) {
	assert := assert.New(t)

	vnic, _ := NewContainerVnic("testvnic")
	assert.Nil(vnic.Create())

	vnic1, _ := NewContainerVnic("testvnic")
	assert.Nil(vnic1.GetDevice())

	performVnicOps(true, assert, vnic)
}

//Duplicate VNIC creation detection
//
//Checks if the VNIC create primitive fails gracefully
//on a duplicate VNIC creation
//
//Test is expected to pass
func TestVnic_Dup(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := NewVnic("testvnic")
	vnic1, _ := NewVnic("testvnic")

	assert.Nil(vnic.Create())
	defer func() { _ = vnic.Destroy() }()
	assert.NotNil(vnic1.Create())
}

//Duplicate Container VNIC creation detection
//
//Checks if the VNIC create primitive fails gracefully
//on a duplicate VNIC creation
//
//Test is expected to pass
func TestVnicContainer_Dup(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := NewContainerVnic("testconvnic")
	vnic1, _ := NewContainerVnic("testconvnic")

	assert.Nil(vnic.Create())
	defer func() { _ = vnic.Destroy() }()
	assert.NotNil(vnic1.Create())
}

//Negative test case for VNIC primitives
//
//Simulates various error scenarios and ensures that
//they are handled gracefully
//
//Test is expected to pass
func TestVnic_Invalid(t *testing.T) {
	assert := assert.New(t)
	vnic, err := NewVnic("testvnic")
	assert.Nil(err)

	assert.NotNil(vnic.GetDevice())

	performVnicOps(false, assert, vnic)
}

//Negative test case for Container VNIC primitives
//
//Simulates various error scenarios and ensures that
//they are handled gracefully
//
//Test is expected to pass
func TestVnicContainer_Invalid(t *testing.T) {
	assert := assert.New(t)

	vnic, err := NewContainerVnic("testcvnic")
	assert.Nil(err)

	assert.NotNil(vnic.GetDevice())

	performVnicOps(false, assert, vnic)
}

//Test ability to attach to an existing VNIC
//
//Tests the ability to attach to an existing
//VNIC and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnic_GetDevice(t *testing.T) {
	assert := assert.New(t)
	vnic1, _ := NewVnic("testvnic")

	assert.Nil(vnic1.Create())
	vnic, _ := NewVnic("testvnic")

	assert.Nil(vnic.GetDevice())
	assert.NotEqual(vnic.InterfaceName(), "")
	assert.Equal(vnic.InterfaceName(), vnic1.InterfaceName())
	assert.NotEqual(vnic1.PeerName(), "")
	assert.Equal(vnic1.PeerName(), vnic.PeerName())

	performVnicOps(true, assert, vnic)
}

//Test ability to attach to an existing Container VNIC
//
//Tests the ability to attach to an existing
//VNIC and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnicContainer_GetDevice(t *testing.T) {
	assert := assert.New(t)

	vnic1, err := NewContainerVnic("testvnic")
	assert.Nil(err)

	err = vnic1.Create()
	assert.Nil(err)

	vnic, err := NewContainerVnic("testvnic")
	assert.Nil(err)

	assert.Nil(vnic.GetDevice())
	performVnicOps(true, assert, vnic)
}

//Test ability to attach to an existing VNIC
//
//Tests the ability to attach to an existing
//VNIC and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnic_GetDeviceByName(t *testing.T) {
	assert := assert.New(t)
	vnic1, _ := NewVnic("testvnic")
	vnic1.LinkName = "testiface"

	assert.Nil(vnic1.Create())
	vnic, _ := NewVnic("testvnic")

	assert.Nil(vnic.GetDeviceByName("testiface"))
	assert.NotEqual(vnic.InterfaceName(), "")
	assert.Equal(vnic.InterfaceName(), vnic1.InterfaceName())
	assert.NotEqual(vnic1.PeerName(), "")
	assert.Equal(vnic1.PeerName(), vnic.PeerName())

	performVnicOps(true, assert, vnic)
}

//Test ability to attach to an existing Container VNIC
//
//Tests the ability to attach to an existing
//VNIC and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnicContainer_GetDeviceByName(t *testing.T) {
	assert := assert.New(t)

	vnic1, err := NewContainerVnic("testvnic")
	assert.Nil(err)
	vnic1.LinkName = "testiface"

	err = vnic1.Create()
	assert.Nil(err)

	vnic, err := NewContainerVnic("testvnic")
	assert.Nil(err)

	assert.Nil(vnic.GetDeviceByName("testiface"))
	performVnicOps(true, assert, vnic)
}

//Tests VNIC attach to a bridge
//
//Tests all interactions between VNIC and Bridge
//
//Test is expected to pass
func TestVnic_Bridge(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := NewVnic("testvnic")
	bridge, _ := NewBridge("testbridge")

	assert.Nil(vnic.Create())
	defer func() { _ = vnic.Destroy() }()

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	assert.Nil(vnic.Attach(bridge))
	assert.Nil(vnic.Enable())
	assert.Nil(bridge.Enable())
	assert.Nil(vnic.Detach(bridge))

}

//Tests Container VNIC attach to a bridge
//
//Tests all interactions between VNIC and Bridge
//
//Test is expected to pass
func TestVnicContainer_Bridge(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := NewContainerVnic("testvnic")
	bridge, _ := NewBridge("testbridge")

	assert.Nil(vnic.Create())

	defer func() { _ = vnic.Destroy() }()

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	assert.Nil(vnic.Attach(bridge))
	assert.Nil(vnic.Enable())
	assert.Nil(bridge.Enable())
	assert.Nil(vnic.Detach(bridge))
}
