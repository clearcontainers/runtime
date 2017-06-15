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
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/01org/ciao/ssntp/uuid"
)

// NoopDriver is a driver which does nothing.
type NoopDriver struct {
	deviceNum int64
}

// CreateBlockDevice pretends to create a block device.
func (d *NoopDriver) CreateBlockDevice(volumeUUID string, image string, size int) (BlockDevice, error) {
	return BlockDevice{ID: uuid.Generate().String(), Size: size}, nil
}

// CreateBlockDeviceFromSnapshot pretends to create a block device snapshot
func (d *NoopDriver) CreateBlockDeviceFromSnapshot(volumeUUID string, snapshotID string) (BlockDevice, error) {
	return BlockDevice{ID: uuid.Generate().String() + "@" + uuid.Generate().String()}, nil
}

// CreateBlockDeviceSnapshot pretends to create a block device snapshot
func (d *NoopDriver) CreateBlockDeviceSnapshot(volumeUUID string, snapshotID string) error {
	return nil
}

// CopyBlockDevice pretends to copy an existing block device
func (d *NoopDriver) CopyBlockDevice(string) (BlockDevice, error) {
	return BlockDevice{ID: uuid.Generate().String()}, nil
}

// DeleteBlockDevice pretends to delete a block device.
func (d *NoopDriver) DeleteBlockDevice(string) error {
	return nil
}

// DeleteBlockDeviceSnapshot pretends to create a block device snapshot
func (d *NoopDriver) DeleteBlockDeviceSnapshot(volumeUUID string, snapshotID string) error {
	return nil
}

// GetBlockDeviceSize pretends to return the number of bytes used by the block device
func (d *NoopDriver) GetBlockDeviceSize(volumeUUID string) (uint64, error) {
	return 0, nil
}

// MapVolumeToNode pretends to map a volume to a local device on a node.
func (d *NoopDriver) MapVolumeToNode(volumeUUID string) (string, error) {
	dNum := atomic.AddInt64(&d.deviceNum, 1)
	return fmt.Sprintf("/dev/blk%d", dNum), nil
}

// UnmapVolumeFromNode pretends to unmap a volume from a local device on a node.
func (d *NoopDriver) UnmapVolumeFromNode(volumeUUID string) error {
	return nil
}

// GetVolumeMapping returns an empty slice, indicating no devices are mapped to the
// specified volume.
func (d *NoopDriver) GetVolumeMapping() (map[string][]string, error) {
	return nil, nil
}

// IsValidSnapshotUUID checks for the Ciao standard snapshot uuid form of
// {UUID}@{UUID}
func (d *NoopDriver) IsValidSnapshotUUID(snapshotUUID string) error {
	UUIDs := strings.Split(snapshotUUID, "@")
	if len(UUIDs) != 2 {
		return fmt.Errorf("missing '@'")
	}
	_, e1 := uuid.Parse(UUIDs[0])
	_, e2 := uuid.Parse(UUIDs[1])
	if e1 != nil || e2 != nil {
		return fmt.Errorf("uuid not of form \"{UUID}@{UUID}\"")
	}

	return nil
}

// Resize the underlying rbd image. Only extending is permitted.
func (d *NoopDriver) Resize(volumeUUID string, sizeGiB int) (int, error) {
	return sizeGiB, nil
}
