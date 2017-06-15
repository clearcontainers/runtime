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
	"os"
	"path"
)

// Posix implements the DataStore interface for posix filesystems
type Posix struct {
	MountPoint string
}

// Write copies an image into the posix filesystem.
func (p *Posix) Write(ID string, body io.Reader) (err error) {
	imageName := path.Join(p.MountPoint, ID)
	if _, err := os.Stat(imageName); !os.IsNotExist(err) {
		return fmt.Errorf("image already uploaded with that ID")
	}

	image, err := os.Create(imageName)
	if err != nil {
		return err
	}

	buf := make([]byte, 1<<16)

	_, err = io.CopyBuffer(image, body, buf)
	defer func() {
		err1 := image.Close()
		if err == nil {
			err = err1
		}
	}()

	return err
}

// Delete removes an image from the posix filesystem
func (p *Posix) Delete(ID string) error {
	imageName := path.Join(p.MountPoint, ID)

	_, err := os.Stat(imageName)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}

	err = os.Remove(imageName)

	return err
}

// GetImageSize obtains the image size from the underlying filesystem
func (p *Posix) GetImageSize(ID string) (uint64, error) {
	imageName := path.Join(p.MountPoint, ID)

	fi, err := os.Stat(imageName)
	imageSize := uint64(fi.Size())
	if err != nil {
		return 0, fmt.Errorf("Error getting image size: %v", err)
	}
	return imageSize, nil
}
