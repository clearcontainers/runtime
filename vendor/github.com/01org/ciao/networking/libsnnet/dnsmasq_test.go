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
	"io/ioutil"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

//Test normal operation DCHP/DNS server setup for a CNCI
//
//This test created a bridge, assigns an IP to it, attaches
//a bridge local dnsmasq process to serve DHCP and DNS on this
//bridge. It also tests for reload of the dnsmasq, stop and
//restart
//
//Test is expected to pass
func TestDnsmasq_Basic(t *testing.T) {
	assert := assert.New(t)

	id := "concuuid"
	tenant := "tenantuuid"
	reserved := 0
	subnet := net.IPNet{
		IP:   net.IPv4(192, 168, 1, 0),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}

	bridge, _ := NewBridge("dns_testbr")

	err := bridge.Create()
	assert.Nil(err)

	defer func() { _ = bridge.Destroy() }()

	d, err := newDnsmasq(id, tenant, subnet, reserved, bridge)
	assert.Nil(err)

	if len(d.IPMap) != (256 - reserved - 3) {
		t.Errorf("Incorrect subnet calculation")
	}

	err = d.start()
	assert.Nil(err)

	err = d.reload()
	assert.Nil(err)

	err = d.restart()
	assert.Nil(err)

	err = d.stop()
	assert.Nil(err)

	err = d.restart()
	assert.Nil(err)

	err = d.reload()
	assert.Nil(err)

	err = d.stop()
	assert.Nil(err)

}

//Dnsmasq negative test cases
//
//Tests that error conditions are handled gracefully
//Checks that duplicate subnet creation is handled properly
//Note: This test needs improvement
//
//Test is expected to pass
func TestDnsmasq_Negative(t *testing.T) {
	assert := assert.New(t)

	id := "concuuid"
	tenant := "tenantuuid"
	reserved := 10
	subnet := net.IPNet{
		IP:   net.IPv4(192, 168, 1, 0),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}

	bridge, _ := NewBridge("dns_testbr")

	err := bridge.Create()
	assert.Nil(err)
	defer func() { _ = bridge.Destroy() }()

	// Note: Re instantiate d each time as that
	// is how it will be used
	d, err := newDnsmasq(id, tenant, subnet, reserved, bridge)
	if assert.Nil(err) {
		assert.Nil(d.start())

	}
	//Attach should work
	d, err = newDnsmasq(id, tenant, subnet, reserved, bridge)
	if assert.Nil(err) {
		pid, err := d.attach()
		if assert.Nil(err) {
			pidStr := strconv.Itoa(pid)
			fileName := "/proc/" + pidStr + "/cmdline"
			_, err := ioutil.ReadFile(fileName)
			assert.Nil(err)
		}
	}

	d, err = newDnsmasq(id, tenant, subnet, reserved, bridge)
	if assert.Nil(err) {
		//Restart should work
		assert.Nil(d.restart())
		//Reload should work
		assert.Nil(d.reload())
	}

	// Duplicate creation - should fail
	d, err = newDnsmasq(id, tenant, subnet, reserved, bridge)
	if assert.Nil(err) {
		assert.NotNil(d.start())
		assert.Nil(d.stop())
		_, err = d.attach()
		assert.NotNil(err)
		assert.NotNil(d.stop())
		assert.NotNil(d.reload())
	}

	//Restart should not fail
	d, err = newDnsmasq(id, tenant, subnet, reserved, bridge)
	if assert.Nil(err) {
		assert.Nil(d.restart())
		assert.Nil(d.stop())
	}
}
