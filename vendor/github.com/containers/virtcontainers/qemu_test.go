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

package virtcontainers

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	ciaoQemu "github.com/01org/ciao/qemu"
)

const (
	testQemuKernelPath = testDir + testKernel
	testQemuImagePath  = testDir + testImage
	testQemuPath       = testDir + testHypervisor
)

func newQemuConfig() HypervisorConfig {
	return HypervisorConfig{
		KernelPath:     testQemuKernelPath,
		ImagePath:      testQemuImagePath,
		HypervisorPath: testQemuPath,
	}
}

func testQemuBuildKernelParams(t *testing.T, kernelParams []Param, expected string) {
	qemuConfig := newQemuConfig()
	qemuConfig.KernelParams = kernelParams

	q := &qemu{}

	err := q.buildKernelParams(qemuConfig)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Join(q.kernelParams, " ") != expected {
		t.Fatal()
	}
}

var testQemuKernelParams = "root=/dev/pmem0p1 rootflags=dax,data=ordered,errors=remount-ro rw rootfstype=ext4 tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k panic=1 console=hvc0 console=hvc1 initcall_debug init=/usr/lib/systemd/systemd systemd.unit=container.target iommu=off quiet systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket systemd.show_status=false cryptomgr.notests"

func TestQemuBuildKernelParamsFoo(t *testing.T) {
	expectedOut := testQemuKernelParams + " foo=foo bar=bar"

	params := []Param{
		{
			parameter: "foo",
			value:     "foo",
		},
		{
			parameter: "bar",
			value:     "bar",
		},
	}

	testQemuBuildKernelParams(t, params, expectedOut)
}

func testQemuAppend(t *testing.T, structure interface{}, expected []ciaoQemu.Device, devType deviceType) {
	var devices []ciaoQemu.Device
	q := &qemu{}

	switch s := structure.(type) {
	case Volume:
		devices = q.appendVolume(devices, s)
	case Socket:
		devices = q.appendSocket(devices, s)
	case PodConfig:
		switch devType {
		case fsDev:
			devices = q.appendFSDevices(devices, s)
		case consoleDev:
			devices = q.appendConsoles(devices, s)
		}
	}

	if reflect.DeepEqual(devices, expected) == false {
		t.Fatal()
	}
}

func TestQemuAppendVolume(t *testing.T) {
	mountTag := "testMountTag"
	hostPath := "testHostPath"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-%s-9p", mountTag),
			Path:          hostPath,
			MountTag:      mountTag,
			SecurityModel: ciaoQemu.None,
		},
	}

	volume := Volume{
		MountTag: mountTag,
		HostPath: hostPath,
	}

	testQemuAppend(t, volume, expectedOut, -1)
}

func TestQemuAppendSocket(t *testing.T) {
	deviceID := "channelTest"
	id := "charchTest"
	hostPath := "/tmp/hyper_test.sock"
	name := "sh.hyper.channel.test"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.CharDevice{
			Driver:   ciaoQemu.VirtioSerialPort,
			Backend:  ciaoQemu.Socket,
			DeviceID: deviceID,
			ID:       id,
			Path:     hostPath,
			Name:     name,
		},
	}

	socket := Socket{
		DeviceID: deviceID,
		ID:       id,
		HostPath: hostPath,
		Name:     name,
	}

	testQemuAppend(t, socket, expectedOut, -1)
}

func TestQemuAppendFSDevices(t *testing.T) {
	podID := "testPodID"
	contID := "testContID"
	contRootFs := "testContRootFs"
	volMountTag := "testVolMountTag"
	volHostPath := "testVolHostPath"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("ctr-%s-9p", fmt.Sprintf("%s.1", contID)),
			Path:          fmt.Sprintf("%s.1", contRootFs),
			MountTag:      fmt.Sprintf("ctr-rootfs-%s.1", contID),
			SecurityModel: ciaoQemu.None,
		},
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("ctr-%s-9p", fmt.Sprintf("%s.2", contID)),
			Path:          fmt.Sprintf("%s.2", contRootFs),
			MountTag:      fmt.Sprintf("ctr-rootfs-%s.2", contID),
			SecurityModel: ciaoQemu.None,
		},
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-%s-9p", fmt.Sprintf("%s.1", volMountTag)),
			Path:          fmt.Sprintf("%s.1", volHostPath),
			MountTag:      fmt.Sprintf("%s.1", volMountTag),
			SecurityModel: ciaoQemu.None,
		},
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-%s-9p", fmt.Sprintf("%s.2", volMountTag)),
			Path:          fmt.Sprintf("%s.2", volHostPath),
			MountTag:      fmt.Sprintf("%s.2", volMountTag),
			SecurityModel: ciaoQemu.None,
		},
	}

	volumes := []Volume{
		{
			MountTag: fmt.Sprintf("%s.1", volMountTag),
			HostPath: fmt.Sprintf("%s.1", volHostPath),
		},
		{
			MountTag: fmt.Sprintf("%s.2", volMountTag),
			HostPath: fmt.Sprintf("%s.2", volHostPath),
		},
	}

	containers := []ContainerConfig{
		{
			ID:     fmt.Sprintf("%s.1", contID),
			RootFs: fmt.Sprintf("%s.1", contRootFs),
		},
		{
			ID:     fmt.Sprintf("%s.2", contID),
			RootFs: fmt.Sprintf("%s.2", contRootFs),
		},
	}

	podConfig := PodConfig{
		ID:         podID,
		Volumes:    volumes,
		Containers: containers,
	}

	testQemuAppend(t, podConfig, expectedOut, fsDev)
}

func TestQemuAppendConsoles(t *testing.T) {
	podID := "testPodID"
	contConsolePath := "testContConsolePath"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.SerialDevice{
			Driver: ciaoQemu.VirtioSerial,
			ID:     "serial0",
		},
		ciaoQemu.CharDevice{
			Driver:   ciaoQemu.Console,
			Backend:  ciaoQemu.Serial,
			DeviceID: "console0",
			ID:       "charconsole0",
			Path:     contConsolePath,
		},
		ciaoQemu.CharDevice{
			Driver:   ciaoQemu.Console,
			Backend:  ciaoQemu.Socket,
			DeviceID: "console1",
			ID:       "charconsole1",
			Path:     fmt.Sprintf("%s/%s/console.sock", runStoragePath, podID),
		},
	}

	containers := []ContainerConfig{
		{
			Interactive: true,
			Console:     contConsolePath,
		},
		{
			Interactive: false,
			Console:     "",
		},
	}

	podConfig := PodConfig{
		ID:         podID,
		Containers: containers,
	}

	testQemuAppend(t, podConfig, expectedOut, consoleDev)
}

func TestQemuAppendImage(t *testing.T) {
	var devices []ciaoQemu.Device

	qemuConfig := newQemuConfig()
	q := &qemu{
		config: qemuConfig,
	}

	imageFile, err := os.Open(q.config.ImagePath)
	if err != nil {
		t.Fatal(err)
	}
	defer imageFile.Close()

	imageStat, err := imageFile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.Object{
			Driver:   ciaoQemu.NVDIMM,
			Type:     ciaoQemu.MemoryBackendFile,
			DeviceID: "nv0",
			ID:       "mem0",
			MemPath:  q.config.ImagePath,
			Size:     (uint64)(imageStat.Size()),
		},
	}

	podConfig := PodConfig{}

	devices, err = q.appendImage(devices, podConfig)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(devices, expectedOut) == false {
		t.Fatal()
	}
}

func TestQemuInit(t *testing.T) {
	qemuConfig := newQemuConfig()
	q := &qemu{}

	err := q.init(qemuConfig)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(qemuConfig, q.config) == false {
		t.Fatal()
	}

	if reflect.DeepEqual(qemuConfig.HypervisorPath, q.path) == false {
		t.Fatal()
	}

	if strings.Join(q.kernelParams, " ") != testQemuKernelParams {
		t.Fatal()
	}
}

func TestQemuSetCPUResources(t *testing.T) {
	vcpus := 1

	q := &qemu{}

	expectedOut := ciaoQemu.SMP{
		CPUs:    uint32(vcpus),
		Cores:   uint32(vcpus),
		Sockets: uint32(1),
		Threads: uint32(1),
	}

	vmConfig := Resources{
		VCPUs: uint(vcpus),
	}

	podConfig := PodConfig{
		VMConfig: vmConfig,
	}

	smp := q.setCPUResources(podConfig)

	if reflect.DeepEqual(smp, expectedOut) == false {
		t.Fatal()
	}
}

func TestQemuSetMemoryResources(t *testing.T) {
	mem := 1000

	q := &qemu{}

	expectedOut := ciaoQemu.Memory{
		Size:   "1000M",
		Slots:  uint8(2),
		MaxMem: "1500M",
	}

	vmConfig := Resources{
		Memory: uint(mem),
	}

	podConfig := PodConfig{
		VMConfig: vmConfig,
	}

	memory := q.setMemoryResources(podConfig)

	if reflect.DeepEqual(memory, expectedOut) == false {
		t.Fatal()
	}
}

func testQemuAddDevice(t *testing.T, devInfo interface{}, devType deviceType, expected []ciaoQemu.Device) {
	q := &qemu{}

	err := q.addDevice(devInfo, devType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(q.qemuConfig.Devices, expected) == false {
		t.Fatal()
	}
}

func TestQemuAddDeviceFsDev(t *testing.T) {
	mountTag := "testMountTag"
	hostPath := "testHostPath"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-%s-9p", mountTag),
			Path:          hostPath,
			MountTag:      mountTag,
			SecurityModel: ciaoQemu.None,
		},
	}

	volume := Volume{
		MountTag: mountTag,
		HostPath: hostPath,
	}

	testQemuAddDevice(t, volume, fsDev, expectedOut)
}

func TestQemuAddDeviceSerialPordDev(t *testing.T) {
	deviceID := "channelTest"
	id := "charchTest"
	hostPath := "/tmp/hyper_test.sock"
	name := "sh.hyper.channel.test"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.CharDevice{
			Driver:   ciaoQemu.VirtioSerialPort,
			Backend:  ciaoQemu.Socket,
			DeviceID: deviceID,
			ID:       id,
			Path:     hostPath,
			Name:     name,
		},
	}

	socket := Socket{
		DeviceID: deviceID,
		ID:       id,
		HostPath: hostPath,
		Name:     name,
	}

	testQemuAddDevice(t, socket, serialPortDev, expectedOut)
}
