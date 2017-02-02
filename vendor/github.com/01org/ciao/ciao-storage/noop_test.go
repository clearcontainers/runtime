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

package storage

import (
	"os"
	"testing"

	"github.com/01org/ciao/bat"
)

var noopDriver = NoopDriver{}

// Check creating a ceph backed block device works
//
// TestCreateBlockDevice creates a block device containing some random data,
// checks for errors and then deletes it.
func TestNoopCreateBlockDevice(t *testing.T) {
	path, err := bat.CreateRandomFile(20)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	device, err := noopDriver.CreateBlockDevice("", path, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = noopDriver.DeleteBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}
}

// Check copying a ceph backed block device works
//
// TestCopyBlockDevice creates a block device containing some random data,
// checks for errors and then copies it. The created volumes are then
// deleted.
func TestNoopCopyBlockDevice(t *testing.T) {
	path, err := bat.CreateRandomFile(20)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	device, err := noopDriver.CreateBlockDevice("", path, 0)
	if err != nil {
		t.Fatal(err)
	}

	copy, err := noopDriver.CopyBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = noopDriver.DeleteBlockDevice(copy.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = noopDriver.DeleteBlockDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoopMappings(t *testing.T) {
	s, err := noopDriver.MapVolumeToNode("")
	if err != nil || s != "/dev/blk1" {
		t.Fatal(err)
	}

	s, err = noopDriver.MapVolumeToNode("")
	if err != nil || s != "/dev/blk2" {
		t.Fatal(err)
	}

	m, err := noopDriver.GetVolumeMapping()
	if err != nil || m != nil {
		t.Fatal(err)
	}

	err = noopDriver.UnmapVolumeFromNode("")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoopSnapshots(t *testing.T) {
	err := noopDriver.CreateBlockDeviceSnapshot("", "")
	if err != nil {
		t.Fatal(err)
	}

	bd, err := noopDriver.CreateBlockDeviceFromSnapshot("", "")
	if err != nil || bd.ID == "" {
		t.Fatal(err)
	}

	err = noopDriver.IsValidSnapshotUUID(bd.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = noopDriver.DeleteBlockDeviceSnapshot("", "")
	if err != nil {
		t.Fatal(err)
	}
}
