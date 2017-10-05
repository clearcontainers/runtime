//
// Copyright (c) 2017 Intel Corporation
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

package virtcontainers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-ini/ini"
)

const (
	// DeviceVFIO is the VFIO device type
	DeviceVFIO = "vfio"

	// DeviceBlock is the block device type
	DeviceBlock = "block"

	// DeviceGeneric is a generic device type
	DeviceGeneric = "generic"
)

// Defining this as a variable instead of a const, to allow
// overriding this in the tests.
var sysIOMMUPath = "/sys/kernel/iommu_groups"

var sysDevPrefix = "/sys/dev"

var blockPaths = []string{
	"/dev/sd",   //SCSI block device
	"/dev/hd",   //IDE block device
	"/dev/vd",   //Virtual Block device
	"/dev/ida/", //Compaq Intelligent Drive Array devices
}

const (
	vfioPath = "/dev/vfio/"
)

// Device is the virtcontainers device interface.
type Device interface {
	attach(hypervisor) error
	detach(hypervisor) error
}

// DeviceInfo is an embedded type that contains device data common to all types of devices.
type DeviceInfo struct {
	// Device path on host
	HostPath string

	// Device path inside the container
	ContainerPath string

	// Type of device: c, b, u or p
	// c , u - character(unbuffered)
	// p - FIFO
	// b - block(buffered) special file
	// More info in mknod(1).
	DevType string

	// Major, minor numbers for device.
	Major int64
	Minor int64

	// FileMode permission bits for the device.
	FileMode os.FileMode

	// id of the device owner.
	UID uint32

	// id of the device group.
	GID uint32
}

func newDeviceInfo(m map[string]interface{}) (DeviceInfo, error) {
	var d DeviceInfo

	s, ok := m["HostPath"]
	if ok {
		d.HostPath = s.(string)
	}

	s, ok = m["ContainerPath"]
	if ok {
		d.ContainerPath = s.(string)
	}

	s, ok = m["DevType"]
	if ok {
		d.DevType = s.(string)
	}

	s, ok = m["Major"]
	if ok {
		d.Major = int64(s.(float64))
	}

	s, ok = m["Minor"]
	if ok {
		d.Minor = int64(s.(float64))
	}

	s, ok = m["UID"]
	if ok {
		d.UID = uint32(s.(float64))
	}

	s, ok = m["GID"]
	if ok {
		d.GID = uint32(s.(float64))
	}

	s, ok = m["FileMode"]
	if ok {
		d.FileMode = os.FileMode(s.(float64))
	}

	return d, nil
}

// VFIODevice is a vfio device meant to be passed to the hypervisor
// to be used by the Virtual Machine.
type VFIODevice struct {
	DeviceType string
	DeviceInfo DeviceInfo
	BDF        string
}

func newVFIODevice(devInfo DeviceInfo) *VFIODevice {
	return &VFIODevice{
		DeviceType: DeviceVFIO,
		DeviceInfo: devInfo,
	}
}

func (device *VFIODevice) attach(h hypervisor) error {
	vfioGroup := filepath.Base(device.DeviceInfo.HostPath)
	iommuDevicesPath := filepath.Join(sysIOMMUPath, vfioGroup, "devices")

	deviceFiles, err := ioutil.ReadDir(iommuDevicesPath)
	if err != nil {
		return err
	}

	// Pass all devices in iommu group
	for _, deviceFile := range deviceFiles {

		//Get bdf of device eg 0000:00:1c.0
		deviceBDF, err := getBDF(deviceFile.Name())
		if err != nil {
			return err
		}

		device.BDF = deviceBDF

		if err := h.addDevice(device, vfioDev); err != nil {
			return err
		}
	}

	return nil
}

func (device *VFIODevice) detach(h hypervisor) error {
	return nil
}

// BlockDevice refers to a block storage device implementation.
type BlockDevice struct {
	DeviceType string
	DeviceInfo DeviceInfo
}

func newBlockDevice(devInfo DeviceInfo) *BlockDevice {
	return &BlockDevice{
		DeviceType: DeviceBlock,
		DeviceInfo: devInfo,
	}
}

func (device *BlockDevice) attach(h hypervisor) error {
	return nil
}

func (device BlockDevice) detach(h hypervisor) error {
	return nil
}

// GenericDevice refers to a device that is neither a VFIO device or block device.
type GenericDevice struct {
	DeviceType string
	DeviceInfo DeviceInfo
}

func newGenericDevice(devInfo DeviceInfo) *GenericDevice {
	return &GenericDevice{
		DeviceType: DeviceGeneric,
		DeviceInfo: devInfo,
	}
}

func (device *GenericDevice) attach(h hypervisor) error {
	return nil
}

func (device *GenericDevice) detach(h hypervisor) error {
	return nil
}

// isVFIO checks if the device provided is a vfio group.
func isVFIO(hostPath string) bool {
	if strings.HasPrefix(hostPath, vfioPath) && len(hostPath) > len(vfioPath) {
		return true
	}

	return false
}

// isBlock checks if the device is a block device.
func isBlock(hostPath string) bool {
	for _, blockPath := range blockPaths {
		if strings.HasPrefix(hostPath, blockPath) && len(hostPath) > len(blockPath) {
			return true
		}
	}

	return false
}

func createDevice(devInfo DeviceInfo) Device {
	path := devInfo.HostPath

	if isVFIO(path) {
		return newVFIODevice(devInfo)
	} else if isBlock(path) {
		return newBlockDevice(devInfo)
	} else {
		return newGenericDevice(devInfo)
	}
}

// GetHostPath is used to fetcg the host path for the device.
// The path passed in the spec refers to the path that should appear inside the container.
// We need to find the actual device path on the host based on the major-minor numbers of the device.
func GetHostPath(devInfo DeviceInfo) (string, error) {
	if devInfo.ContainerPath == "" {
		return "", fmt.Errorf("Empty path provided for device")
	}

	var pathComp string

	switch devInfo.DevType {
	case "c", "u":
		pathComp = "char"
	case "b":
		pathComp = "block"
	default:
		// Unsupported device types. Return nil error to ignore devices
		// that cannot be handled currently.
		return "", nil
	}

	format := strconv.FormatInt(devInfo.Major, 10) + ":" + strconv.FormatInt(devInfo.Minor, 10)
	sysDevPath := filepath.Join(sysDevPrefix, pathComp, format, "uevent")

	content, err := ini.Load(sysDevPath)
	if err != nil {
		return "", err
	}

	devName, err := content.Section("").GetKey("DEVNAME")
	if err != nil {
		return "", err
	}

	return filepath.Join("/dev", devName.String()), nil
}

// GetHostPathFunc is function pointer used to mock GetHostPath in tests.
var GetHostPathFunc = GetHostPath

// NewDevice returns a device interface implementation based on the host path of the device.
// The hostpath itself is inferred based on the major-minor number of the device.
func NewDevice(path, devType string, major, minor int64, fileMode *os.FileMode, uid, gid uint32) (Device, error) {
	devInfo := DeviceInfo{
		Major:         major,
		Minor:         minor,
		UID:           uid,
		GID:           gid,
		DevType:       devType,
		ContainerPath: path,
	}

	if fileMode != nil {
		devInfo.FileMode = *fileMode
	}

	hostPath, err := GetHostPathFunc(devInfo)
	if err != nil {
		return nil, err
	}

	devInfo.HostPath = hostPath

	device := createDevice(devInfo)
	return device, nil
}

// bdf returns the BDF of pci device
// Expected input strng format is [<domain>]:[<bus>][<slot>].[<func>] eg. 0000:02:10.0
func getBDF(deviceSysStr string) (string, error) {
	tokens := strings.Split(deviceSysStr, ":")

	if len(tokens) != 3 {
		return "", fmt.Errorf("Incorrect number of tokens found while parsing bdf for device : %s", deviceSysStr)
	}

	tokens = strings.SplitN(deviceSysStr, ":", 2)
	return tokens[1], nil
}
