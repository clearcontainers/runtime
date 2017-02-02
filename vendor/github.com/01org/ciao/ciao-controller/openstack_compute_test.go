//
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
//

package main

import (
	"testing"

	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/openstack/compute"
)

func compareStorageBlockDevices(t *testing.T, a storage.BlockDevice, b storage.BlockDevice) {
	if a.ID != b.ID {
		t.Errorf("Volume ID mismatch, expected %s got %s", b.ID, a.ID)
	}
	if a.Bootable != b.Bootable {
		t.Errorf("Volume Bootable flag mismatch, expected %t got %t", b.Bootable, a.Bootable)
	}
	if a.BootIndex != b.BootIndex {
		t.Errorf("Volume BootIndex mismatch, expected %d got %d", b.BootIndex, a.BootIndex)
	}
	if a.Ephemeral != b.Ephemeral {
		t.Errorf("Volume Ephemeral flag mismatch, expected %t got %t", b.Ephemeral, a.Ephemeral)
	}
	if a.Local != b.Local {
		t.Errorf("Volume Local flag mismatch, expected %t got %t", b.Local, a.Local)
	}
	if a.Swap != b.Swap {
		t.Errorf("Volume Swap flag mismatch, expected %t got %t", b.Swap, a.Swap)
	}
	if a.Tag != b.Tag {
		t.Errorf("Volume Tag mismatch, expected %s got %s", b.Tag, a.Tag)
	}
	if a.Size != b.Size {
		t.Errorf("Volume Size mismatch, expected %d got %d", b.Size, a.Size)
	}
}

// valid volume lists
var computeNoVolumes = []compute.BlockDeviceMappingV2{}
var storageNoVolumes = []storage.BlockDevice{}
var computeOneGoodVolume = []compute.BlockDeviceMappingV2{
	computeLocalSwapVolume,
}
var storageOneGoodVolume = []storage.BlockDevice{
	storageLocalSwapVolume,
}
var computeMultipleGoodVolumes = []compute.BlockDeviceMappingV2{
	computeVolume,
	computeBootVolume,
	computeVolumeSnapshot,
	computeAutoVolume,
}
var storageMultipleGoodVolumes = []storage.BlockDevice{
	storageVolume,
	storageBootVolume,
	storageVolumeSnapshot,
	storageAutoVolume,
}

// valid volumes
var computeLocalSwapVolume = compute.BlockDeviceMappingV2{
	DeviceName:          "",
	SourceType:          "blank",
	DestinationType:     "local",
	DeleteOnTermination: true,
	GuestFormat:         "swap",
	BootIndex:           "none",
	Tag:                 "A string.",
	VolumeSize:          4,
}
var storageLocalSwapVolume = storage.BlockDevice{
	Bootable:  false,
	BootIndex: 0,
	Ephemeral: true,
	Local:     true,
	Swap:      true,
	Tag:       "A string.",
	Size:      4,
}
var computeAutoVolume = compute.BlockDeviceMappingV2{
	DeviceName:      "",
	SourceType:      "blank",
	DestinationType: "volume",
	VolumeSize:      4,
}
var storageAutoVolume = storage.BlockDevice{
	Size: 4,
}
var computeVolume = compute.BlockDeviceMappingV2{
	SourceType:      "volume",
	DestinationType: "volume",
	BootIndex:       "-1",
	UUID:            "e0217fee-694e-43e6-9149-1da16f3847dc",
}
var storageVolume = storage.BlockDevice{
	BootIndex: 0,
	ID:        "e0217fee-694e-43e6-9149-1da16f3847dc",
}
var computeVolumeSnapshot = compute.BlockDeviceMappingV2{
	SourceType:      "snapshot",
	DestinationType: "volume",
	UUID:            "e0217fee-694e-43e6-9149-1da16f3847dc@725dccf6-e651-436a-ae6a-140d8d794aa3",
}
var storageVolumeSnapshot = storage.BlockDevice{
	ID: "e0217fee-694e-43e6-9149-1da16f3847dc@725dccf6-e651-436a-ae6a-140d8d794aa3",
}
var computeBootVolume = compute.BlockDeviceMappingV2{
	DeviceName:      "",
	SourceType:      "volume",
	DestinationType: "volume",
	BootIndex:       "1",
	UUID:            "08adb275-6702-43ce-8575-d268888f825a",
}
var storageBootVolume = storage.BlockDevice{
	Bootable:  true,
	BootIndex: 1,
	ID:        "08adb275-6702-43ce-8575-d268888f825a",
}

// invalid volume lists
var computeBadVolumes1 = []compute.BlockDeviceMappingV2{
	computeBadVolume1,
}
var computeBadVolumes2 = []compute.BlockDeviceMappingV2{
	computeVolume,
	computeBadVolume2,
	computeBootVolume,
}
var computeBadVolumes3 = []compute.BlockDeviceMappingV2{
	computeBadVolume3,
}
var computeBadVolumes4 = []compute.BlockDeviceMappingV2{
	computeBadVolume4,
}
var computeBadVolumes5 = []compute.BlockDeviceMappingV2{
	computeBadVolume5,
}
var computeBadVolumes6 = []compute.BlockDeviceMappingV2{
	computeBadVolume6,
}
var computeBadVolumes7 = []compute.BlockDeviceMappingV2{
	computeBadVolume7,
}
var computeBadVolumes8 = []compute.BlockDeviceMappingV2{
	computeBadVolume8,
}
var computeBadVolumes9 = []compute.BlockDeviceMappingV2{
	computeBadVolume9,
}
var computeBadVolumes10 = []compute.BlockDeviceMappingV2{
	computeBadVolume10,
}
var computeBadVolumes11 = []compute.BlockDeviceMappingV2{
	computeBadVolume11,
}
var computeBadVolumes12 = []compute.BlockDeviceMappingV2{
	computeBadVolume12,
}
var computeBadVolumes13 = []compute.BlockDeviceMappingV2{
	computeBadVolume13,
}
var computeBadVolumes14 = []compute.BlockDeviceMappingV2{
	computeBadVolume14,
}
var computeBadVolumes15 = []compute.BlockDeviceMappingV2{
	computeBadVolume15,
}
var computeBadVolumes16 = []compute.BlockDeviceMappingV2{
	computeBadVolume16,
}
var computeBadVolumes17 = []compute.BlockDeviceMappingV2{
	computeBadVolume17,
}
var computeBadVolumes18 = []compute.BlockDeviceMappingV2{
	computeBadVolume18,
}
var computeBadVolumes19 = []compute.BlockDeviceMappingV2{
	computeBadVolume19,
}

// invalid volumes
var computeBadVolume1 = compute.BlockDeviceMappingV2{
	// invalid source type
	SourceType: "foo",
}
var computeBadVolume2 = compute.BlockDeviceMappingV2{
	// invalid GuestFormat
	SourceType:      "volume",
	DestinationType: "volume",
	GuestFormat:     "yabbalabba",
}
var computeBadVolume3 = compute.BlockDeviceMappingV2{
	// snapshot with incorrect format uuid
	SourceType:      "snapshot",
	DestinationType: "volume",
	UUID:            "ba5e36b0-a386-477c-9fb5-2564aa2d47d7",
}
var computeBadVolume4 = compute.BlockDeviceMappingV2{
	// auto-created ambiguous, invalid destination type
	SourceType:      "blank",
	DestinationType: "bar",
}
var computeBadVolume5 = compute.BlockDeviceMappingV2{
	// auto-created volume, invalid SourceType
	SourceType:      "volume",
	DestinationType: "volume",
}
var computeBadVolume6 = compute.BlockDeviceMappingV2{
	// auto-created volume, invalid BootIndex
	SourceType:      "blank",
	DestinationType: "volume",
	BootIndex:       "1",
}
var computeBadVolume7 = compute.BlockDeviceMappingV2{
	// auto-created volume, can't pre-set uuid
	SourceType:      "blank",
	DestinationType: "volume",
	UUID:            "6d2bd9b6-d2e3-4b0e-a619-4bea1d1cfc38",
}
var computeBadVolume8 = compute.BlockDeviceMappingV2{
	// auto-created volume, invalid negative size
	SourceType:      "blank",
	DestinationType: "volume",
	VolumeSize:      -4,
}
var computeBadVolume9 = compute.BlockDeviceMappingV2{
	// auto-created local, invalid source type
	SourceType:          "volume",
	DestinationType:     "local",
	DeleteOnTermination: false,
}
var computeBadVolume10 = compute.BlockDeviceMappingV2{
	// auto-created local, invalid BootIndex
	SourceType:          "blank",
	DestinationType:     "local",
	DeleteOnTermination: false,
	BootIndex:           "1",
}
var computeBadVolume11 = compute.BlockDeviceMappingV2{
	// auto-created local, extraneous uuid
	SourceType:      "blank",
	DestinationType: "local",
	UUID:            "14a3c05b-f2ea-424e-850a-fb5289b32ec6",
}
var computeBadVolume12 = compute.BlockDeviceMappingV2{
	// auto-created local, invalid VolumeSize
	SourceType:          "blank",
	DestinationType:     "local",
	DeleteOnTermination: false,
	VolumeSize:          -1,
}
var computeBadVolume13 = compute.BlockDeviceMappingV2{
	// auto-created local must be ephemeral
	SourceType:          "blank",
	DestinationType:     "local",
	VolumeSize:          1,
	DeleteOnTermination: false,
}
var computeBadVolume14 = compute.BlockDeviceMappingV2{
	// pre-created volume, can't set size
	SourceType:      "volume",
	DestinationType: "volume",
	UUID:            "6d2bd9b6-d2e3-4b0e-a619-4bea1d1cfc38",
	VolumeSize:      42,
}
var computeBadVolume15 = compute.BlockDeviceMappingV2{
	// pre-created volume, missing uuid
	SourceType: "volume",
	UUID:       "",
}
var computeBadVolume16 = compute.BlockDeviceMappingV2{
	// pre-created volume, invalid BootIndex
	SourceType:      "volume",
	DestinationType: "volume",
	UUID:            "b576afd0-200e-4ae5-afb2-30f06a0d1a7c",
	BootIndex:       "first",
}
var computeBadVolume17 = compute.BlockDeviceMappingV2{
	// pre-created volume, invalid uuid
	SourceType:      "volume",
	DestinationType: "volume",
	UUID:            "foobarbaz",
}
var computeBadVolume18 = compute.BlockDeviceMappingV2{
	// pre-created volume, invalid GuestFormat
	SourceType:      "volume",
	DestinationType: "volume",
	UUID:            "14a3c05b-f2ea-424e-850a-fb5289b32ec6",
	GuestFormat:     "swap",
}
var computeBadVolume19 = compute.BlockDeviceMappingV2{
	// pre-created volume, invalid DestinationType
	SourceType:      "volume",
	DestinationType: "local",
	UUID:            "14a3c05b-f2ea-424e-850a-fb5289b32ec6",
}

//[]compute.BlockDeviceMappingV2 to []storage.BlockDevice
func TestAbstractBlockDevices(t *testing.T) {
	var blockDeviceTests = []struct {
		computeBDs []compute.BlockDeviceMappingV2
		storageBDs []storage.BlockDevice
	}{
		{
			computeNoVolumes,
			storageNoVolumes,
		},
		{
			computeOneGoodVolume,
			storageOneGoodVolume,
		},
		{
			computeMultipleGoodVolumes,
			storageMultipleGoodVolumes,
		},
	}
	for _, test := range blockDeviceTests {
		out := abstractBlockDevices(test.computeBDs)
		for i := range test.storageBDs {
			compareStorageBlockDevices(t, out[i], test.storageBDs[i])
		}
	}
}

func TestValidateBlockDeviceMappings(t *testing.T) {
	var blockDeviceTests = []struct {
		volumes []compute.BlockDeviceMappingV2
		ok      bool
	}{
		{computeNoVolumes, true},
		{computeOneGoodVolume, true},
		{computeMultipleGoodVolumes, true},
		{computeBadVolumes1, false},
		{computeBadVolumes2, false},
		{computeBadVolumes3, false},
		{computeBadVolumes4, false},
		{computeBadVolumes5, false},
		{computeBadVolumes6, false},
		{computeBadVolumes7, false},
		{computeBadVolumes8, false},
		{computeBadVolumes9, false},
		{computeBadVolumes10, false},
		{computeBadVolumes11, false},
		{computeBadVolumes12, false},
		{computeBadVolumes13, false},
		{computeBadVolumes14, false},
		{computeBadVolumes15, false},
		{computeBadVolumes16, false},
		{computeBadVolumes17, false},
		{computeBadVolumes18, false},
		{computeBadVolumes19, false},
	}
	for _, test := range blockDeviceTests {
		err := ctl.validateBlockDeviceMappings(test.volumes, 1)
		if test.ok && err != nil {
			t.Errorf("Volume list did not verify as expected:\n%v", test.volumes)
			t.Errorf("Error: %s", err.Error())
		}
		if !test.ok && err == nil {
			t.Errorf("Volume list verified when it should not:\n%v", test.volumes)
		}
	}

	err := ctl.validateBlockDeviceMappings(computeMultipleGoodVolumes, 2)
	if err == nil {
		t.Errorf("Multiple instances using same volume incorrectly allowed")
	}
}
