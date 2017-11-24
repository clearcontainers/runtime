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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBridges(t *testing.T) {
	assert := assert.New(t)
	var countBridges uint32 = 1

	bridges := NewBridges(countBridges, "")
	assert.Nil(bridges)

	bridges = NewBridges(countBridges, QemuQ35)
	assert.Len(bridges, int(countBridges))

	b := bridges[0]
	assert.NotEmpty(b.ID)
	assert.NotNil(b.Address)
}

func TestAddRemoveDevice(t *testing.T) {
	assert := assert.New(t)
	var countBridges uint32 = 1

	// create a bridge
	bridges := NewBridges(countBridges, "")
	assert.Nil(bridges)
	bridges = NewBridges(countBridges, QemuQ35)
	assert.Len(bridges, int(countBridges))

	// add device
	devID := "abc123"
	b := bridges[0]
	addr, err := b.addDevice(devID)
	assert.NoError(err)
	if addr < 1 {
		assert.Fail("address cannot be less then 1")
	}

	// remove device
	err = b.removeDevice("")
	assert.Error(err)

	err = b.removeDevice(devID)
	assert.NoError(err)
}
