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

package client

import (
	"fmt"
	"os"
)

// Client maintains context for ciao clients of the image service
type Client struct {
	// MountPoint specifies where the images are located
	// in the filesystem.
	MountPoint string
}

// GetImagePath returns the file system location of the image
func (c Client) GetImagePath(ID string) (string, error) {
	path := fmt.Sprintf("%s/%s", c.MountPoint, ID)

	_, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	return path, nil
}
