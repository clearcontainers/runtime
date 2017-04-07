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
	"github.com/01org/ciao/database"
)

// MetaDs implements the DataStore interface for persistent data
type MetaDs struct {
	database.DbProvider
	DbDir  string
	DbFile string
}

// Write is the metadata write implementation.
func (m *MetaDs) Write(i Image) error {
	tenant := i.TenantID

	err := m.DbAdd(tenant, i.ID, &i)
	if err != nil {
		return err
	}

	return nil
}

// Delete is the metadata delete implementation.
func (m *MetaDs) Delete(tenant, id string) error {
	return m.DbDelete(tenant, id)
}

// Get is the metadata get implementation.
func (m *MetaDs) Get(tenant, ID string) (Image, error) {

	imageTable := &ImageMap{}
	img, err := m.DbGet(tenant, ID, imageTable)
	if err != nil {
		return Image{}, err
	}

	image := *img.(*Image)
	return image, err
}

// GetAll is the metadata get all images implementation.
func (m *MetaDs) GetAll(tenant string) (images []Image, err error) {
	var elements []interface{}
	imageTable := &ImageMap{}
	elements, err = m.DbProvider.DbGetAll(tenant, imageTable)

	images = make([]Image, len(elements))
	for i, img := range elements {
		image := img.(*Image)
		images[i] = *image
	}

	return images, err
}
