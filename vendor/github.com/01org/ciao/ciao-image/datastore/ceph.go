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

package datastore

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/01org/ciao/ciao-storage"
)

// Ceph implements the DataStore interface for Ceph RBD based storage
type Ceph struct {
	ImageTempDir string
	BlockDriver  storage.CephDriver
}

// Write copies an image onto the fileystem into a tempory location and uploads
// it into ceph, snapshots.
func (c *Ceph) Write(ID string, body io.Reader) error {
	image, err := ioutil.TempFile("", "ciao-image")
	if err != nil {
		return fmt.Errorf("Error creating temporary image file: %v", err)
	}
	defer os.Remove(image.Name())

	// TODO(rbradford): Is there a better way
	buf := make([]byte, 1<<16)
	_, err = io.CopyBuffer(image, body, buf)
	if err != nil {
		image.Close()
		return fmt.Errorf("Error writing to temporary image file: %v", err)
	}

	err = image.Close()
	if err != nil {
		return fmt.Errorf("Error closing temporary image file: %v", err)
	}

	_, err = c.BlockDriver.CreateBlockDevice(ID, image.Name(), 0)
	if err != nil {
		return fmt.Errorf("Error creating block device: %v", err)
	}

	err = c.BlockDriver.CreateBlockDeviceSnapshot(ID, "ciao-image")
	if err != nil {
		c.BlockDriver.DeleteBlockDevice(ID)
		return fmt.Errorf("Unable to create snapshot: %v", err)
	}

	return nil
}

// Delete removes an image from ceph after deleting the snapshot.
func (c *Ceph) Delete(ID string) error {
	err := c.BlockDriver.DeleteBlockDeviceSnapshot(ID, "ciao-image")
	if err != nil {
		return fmt.Errorf("Unable to delete snapshot: %v", err)
	}

	err = c.BlockDriver.DeleteBlockDevice(ID)
	if err != nil {
		return fmt.Errorf("Error deleting block device: %v", err)
	}
	return nil
}

// GetImageSize returns the size, in bytes, of the block device
func (c *Ceph) GetImageSize(ID string) (uint64, error) {
	imageSize, err := c.BlockDriver.GetBlockDeviceSize(ID)
	if err != nil {
		return 0, fmt.Errorf("Error getting image size: %v", err)
	}
	return imageSize, nil
}
