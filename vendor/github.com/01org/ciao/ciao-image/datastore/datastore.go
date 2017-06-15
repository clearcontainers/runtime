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
	"io"
	"time"

	"github.com/01org/ciao/openstack/image"
)

// State represents the state of the image.
type State string

const (
	// Created means that an empty image has been created
	Created State = "created"

	// Saving means the image is being saved
	Saving State = "saving"

	// Active means that the image is created, uploaded and ready to use.
	Active State = "active"

	// Killed means that an image data upload error occurred.
	Killed State = "killed"
)

// Status translate an image state to an openstack image status.
func (state State) Status() image.Status {
	switch state {
	case Created:
		return image.Queued
	case Saving:
		return image.Saving
	case Active:
		return image.Active
	case Killed:
		return image.Killed
	}

	return image.Active
}

// Type represents the valid image types.
type Type string

const (
	// Raw is the raw image format.
	Raw Type = "raw"

	// QCow is the qcow2 format.
	QCow Type = "qcow2"

	// ISO is the iso format.
	ISO Type = "iso"
)

// Image contains the information that ciao will store about the image
type Image struct {
	ID         string
	State      State
	TenantID   string
	Name       string
	CreateTime time.Time
	Type       Type
	Size       uint64
	Visibility image.Visibility
	Tags       string
}

// DataStore is the image data storage interface.
type DataStore interface {
	Init(RawDataStore, MetaDataStore) error
	CreateImage(image Image) error
	GetAllImages(tenant string) ([]Image, error)
	GetImage(tenant, id string) (Image, error)
	UpdateImage(Image) error
	DeleteImage(tenant, id string) error
	UploadImage(tenant, id string, imageFile io.Reader) error
}

// MetaDataStore is the metadata storing interface that's used by
// image cache implementation.
type MetaDataStore interface {
	Write(image Image) error
	Delete(tenant, ID string) error
	Get(tenant, ID string) (Image, error)
	GetAll(tenant string) ([]Image, error)
}

// RawDataStore is the raw data storage interface that's used by the
// image cache implementation.
type RawDataStore interface {
	Write(ID string, body io.Reader) error
	Delete(ID string) error
	GetImageSize(ID string) (uint64, error)
}
