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

func performBridgeOps(shouldPass bool, assert *assert.Assertions, bridge *Bridge) {
	a := assert.Nil
	if !shouldPass {
		a = assert.NotNil
	}
	a(bridge.Enable())
	a(bridge.Disable())
	a(bridge.Destroy())
}

//Test all Bridge primitives
//
//Tests creation, attach, enable, disable and destroy
//of a bridge interface. Any failure indicates a problem
//with the netlink library or kernel API
//
//Test is expected to pass
func TestBridge_Basic(t *testing.T) {
	assert := assert.New(t)

	bridge, err := NewBridge("go_testbr")
	assert.Nil(err)

	assert.Nil(bridge.Create())

	bridge1, err := NewBridge("go_testbr")
	assert.Nil(err)

	assert.Nil(bridge1.GetDevice())
	performBridgeOps(true, assert, bridge)

	assert.NotNil(bridge.Destroy())

}

//Duplicate bridge detection
//
//Checks that duplicate bridge creation is handled
//gracefully and correctly
//
//Test is expected to pass
func TestBridge_Dup(t *testing.T) {
	assert := assert.New(t)
	bridge, err := NewBridge("go_testbr")
	assert.Nil(err)

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	bridge1, err := NewBridge("go_testbr")
	assert.Nil(err)
	assert.NotNil(bridge1.Create())
}

//Negative test cases for bridge primitives
//
//Checks various negative test scenarios are gracefully
//handled
//
//Test is expected to pass
func TestBridge_Invalid(t *testing.T) {
	assert := assert.New(t)

	bridge, err := NewBridge("go_testbr")
	assert.Nil(err)

	assert.NotNil(bridge.GetDevice())

	performBridgeOps(false, assert, bridge)
}

//Tests attaching to an existing bridge
//
//Tests that you can attach to an existing bridge
//and perform all bridge operation on such a bridge
//
//Test is expected to pass
func TestBridge_GetDevice(t *testing.T) {
	assert := assert.New(t)
	bridge, err := NewBridge("go_testbr")
	assert.Nil(err)

	assert.Nil(bridge.Create())

	bridge1, err := NewBridge("go_testbr")
	assert.Nil(err)

	assert.Nil(bridge1.GetDevice())
	performBridgeOps(true, assert, bridge1)
}
