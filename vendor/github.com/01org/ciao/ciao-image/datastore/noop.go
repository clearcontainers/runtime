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

// Noop is a Datastore implementation that does nothing.
// Use it only for development and testing purposes, data
// will not be persistent with the Noop Datastore interface.
type Noop struct {
}

// Write is the noop image metadata write implementation.
// It drops data.
func (n *Noop) Write(i Image) error {
	return nil
}

// Delete is the noop image metadata delete implementation.
// It drops data.
func (n *Noop) Delete(id string) error {
	return nil
}

// GetImageSize is the noop image implementation of image size querying
func (n *Noop) GetImageSize(ID string) (uint64, error) {
	return 0, nil
}

// Get is the noop image metadata get an image implementation.
// It drops data.
func (n *Noop) Get(id string) (Image, error) {
	return Image{ID: id}, nil
}

// GetAll is the noop image metadata get all images implementation.
// It drops data.
func (n *Noop) GetAll() ([]Image, error) {
	return []Image{}, nil
}
