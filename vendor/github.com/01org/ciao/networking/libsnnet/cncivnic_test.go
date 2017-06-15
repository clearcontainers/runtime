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
	"github.com/vishvananda/netlink"
)

//Just pick the first physical interface with an IP
func getFirstPhyDevice() (int, error) {

	links, err := netlink.LinkList()
	if err != nil {
		return 0, err
	}

	for _, link := range links {

		if !validPhysicalLink(link) {
			continue
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil || len(addrs) == 0 {
			continue
		}

		return link.Attrs().Index, nil
	}

	return 0, fmt.Errorf("Unable to obtain physical device")

}

//Test CNCI VNIC primitives
//
//Tests all the primitives used to create a CNCI instance
//compatible vnic including create, enable, disable, destroy
//
//Test is expected to pass
func TestCnciVnic_Basic(t *testing.T) {
	assert := assert.New(t)

	pIndex, err := getFirstPhyDevice()
	assert.Nil(err)

	cnciVnic, _ := newCnciVnic("testcnciVnic")
	cnciVnic.Link.ParentIndex = pIndex
	cnciVnic.Link.HardwareAddr, _ = net.ParseMAC("DE:AD:BE:EF:01:02")

	assert.Nil(cnciVnic.create())

	assert.Nil(cnciVnic.enable())
	assert.Nil(cnciVnic.disable())
	assert.Nil(cnciVnic.destroy())

	assert.NotNil(cnciVnic.destroy())
}

//Test duplicate creation
//
//Tests the creation of a duplicate interface is handled
//gracefully
//
//Test is expected to pass
func TestCnciVnic_Dup(t *testing.T) {
	assert := assert.New(t)

	pIndex, err := getFirstPhyDevice()
	assert.Nil(err)

	cnciVnic, _ := newCnciVnic("testcnciVnic")
	cnciVnic.Link.ParentIndex = pIndex

	assert.Nil(cnciVnic.create())
	assert.NotNil(cnciVnic.create())
	defer func() { _ = cnciVnic.destroy() }()

	cnciVnic1, _ := newCnciVnic("testcnciVnic")
	cnciVnic1.Link.ParentIndex = pIndex

	//Duplicate
	assert.NotNil(cnciVnic1.create())
}

//Negative test cases
//
//Tests for graceful handling of various Negative
//primitive invocation scenarios
//
//Test is expected to pass
func TestCnciVnic_Invalid(t *testing.T) {
	assert := assert.New(t)
	cnciVnic, err := newCnciVnic("testcnciVnic")

	assert.Nil(err)
	assert.NotNil(cnciVnic.getDevice())
	assert.NotNil(cnciVnic.enable())
	assert.NotNil(cnciVnic.disable())
	assert.NotNil(cnciVnic.destroy())

}

//Test ability to attach
//
//Tests that you can attach to an existing CNCI VNIC and
//perform all CNCI VNIC operations on the attached VNIC
//
//Test is expected to pass
func TestCnciVnic_GetDevice(t *testing.T) {
	assert := assert.New(t)
	cnciVnic1, _ := newCnciVnic("testcnciVnic")

	pIndex, err := getFirstPhyDevice()
	assert.Nil(err)
	cnciVnic1.Link.ParentIndex = pIndex

	assert.Nil(cnciVnic1.create())

	cnciVnic, _ := newCnciVnic("testcnciVnic")

	assert.Nil(cnciVnic.getDevice())

	assert.Nil(cnciVnic.enable())
	assert.Nil(cnciVnic.disable())
	assert.Nil(cnciVnic.destroy())
}
