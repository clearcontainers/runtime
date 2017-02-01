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
	"os"
	"path"
	"strings"
	"testing"

	"github.com/01org/ciao/database"
)

var mountPoint = "/tmp"
var metaDsTables = []string{"images"}
var dbDir = "/tmp"
var dbFile = "ciao-image.db"
var testImageID = "12345678-1234-5678-1234-567812345678"

func testCreateAndGet(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    testImageID,
		State: Created,
	}

	imageStore := ImageStore{}
	_ = imageStore.Init(d, m)

	// create the entry
	err := imageStore.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// retrieve the entry
	image, err := imageStore.GetImage(i.ID)
	if err != nil {
		t.Fatal(err)
	}

	if image.ID != i.ID {
		t.Fatal(err)
	}
}

func testGetAll(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    testImageID,
		State: Created,
	}

	imageStore := ImageStore{}
	_ = imageStore.Init(d, m)

	// create the entry
	err := imageStore.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// retrieve the entry
	images, err := imageStore.GetAllImages()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := m.(*Noop); !ok {
		if len(images) != 1 {
			t.Fatalf("len is actually %d\n", len(images))
		}

		if images[0].ID != i.ID {
			t.Fatal(err)
		}
	}
}

func testDelete(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    testImageID,
		State: Created,
	}

	imageStore := ImageStore{}
	_ = imageStore.Init(d, m)

	// create the entry
	err := imageStore.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// delete the entry
	err = imageStore.DeleteImage(i.ID)
	if err != nil {
		t.Fatal(err)
	}

	// now attempt to retrive the entry
	if _, ok := m.(*Noop); !ok {
		_, err = imageStore.GetImage(i.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testUpload(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    testImageID,
		State: Created,
	}

	imageStore := ImageStore{}
	_ = imageStore.Init(d, m)

	// create the entry
	err := imageStore.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// Upload a string
	err = imageStore.UploadImage(i.ID, strings.NewReader("Upload file"))
	if err != nil {
		t.Fatal(err)
	}
}

// cleanDatastore cleans temporal files that were created during the test
func cleanDatastore() {
	_ = os.Remove(path.Join(mountPoint, testImageID))
	_ = os.Remove(path.Join(dbDir, dbFile))
}

// Tests for Noop metaDs

func TestPosixNoopCreateAndGet(t *testing.T) {
	testCreateAndGet(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

func TestPosixNoopGetAll(t *testing.T) {
	testGetAll(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

func TestPosixNoopDelete(t *testing.T) {
	testDelete(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

func TestPosixNoopUpload(t *testing.T) {
	testUpload(t, &Posix{MountPoint: mountPoint}, &Noop{})
	cleanDatastore()
}

// Tests for MetaDs

func initMetaDs() *MetaDs {
	metaDs := &MetaDs{
		DbProvider: database.NewBoltDBProvider(),
		DbDir:      dbDir,
		DbFile:     dbFile,
	}
	metaDsTables := []string{"images"}
	_ = metaDs.DbInit(metaDs.DbDir, metaDs.DbFile)
	_ = metaDs.DbTablesInit(metaDsTables)

	return metaDs
}

func TestPosixMetaDsCreateAndGet(t *testing.T) {
	metaDs := initMetaDs()
	defer metaDs.DbClose()
	testCreateAndGet(t, &Posix{MountPoint: mountPoint}, metaDs)
}

func TestPosixMetaDsGetAll(t *testing.T) {
	metaDs := initMetaDs()
	defer metaDs.DbClose()
	testGetAll(t, &Posix{MountPoint: mountPoint}, metaDs)
}

func TestPosixMetaDsDelete(t *testing.T) {
	metaDs := initMetaDs()
	defer metaDs.DbClose()
	testDelete(t, &Posix{MountPoint: mountPoint}, metaDs)
}

func TestPosixMetaDsUpload(t *testing.T) {
	metaDs := initMetaDs()
	defer metaDs.DbClose()
	testUpload(t, &Posix{MountPoint: mountPoint}, metaDs)
	cleanDatastore()
}
