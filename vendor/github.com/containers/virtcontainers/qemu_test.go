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
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ciaoQemu "github.com/01org/ciao/qemu"
)

func newQemuConfig() HypervisorConfig {
	return HypervisorConfig{
		KernelPath:     testQemuKernelPath,
		ImagePath:      testQemuImagePath,
		HypervisorPath: testQemuPath,
		DefaultVCPUs:   defaultVCPUs,
		DefaultMemSz:   defaultMemSzMiB,
		DefaultBridges: defaultBridges,
	}
}

func testQemuBuildKernelParams(t *testing.T, kernelParams []Param, expected string, debug bool) {
	qemuConfig := newQemuConfig()
	qemuConfig.KernelParams = kernelParams

	if debug == true {
		qemuConfig.Debug = true
	}

	q := &qemu{}

	err := q.buildKernelParams(qemuConfig)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Join(q.kernelParams, " ") != expected {
		t.Fatal()
	}
}

var testQemuKernelParamsBase = "root=/dev/pmem0p1 rootflags=dax,data=ordered,errors=remount-ro rw rootfstype=ext4 tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k panic=1 console=hvc0 console=hvc1 initcall_debug iommu=off cryptomgr.notests net.ifnames=0"
var testQemuKernelParamsNonDebug = "quiet systemd.show_status=false"
var testQemuKernelParamsDebug = "debug systemd.show_status=true systemd.log_level=debug"

func TestQemuBuildKernelParamsFoo(t *testing.T) {
	// two representations of the same kernel parameters
	suffixStr := "foo=foo bar=bar"
	suffixParams := []Param{
		{
			Key:   "foo",
			Value: "foo",
		},
		{
			Key:   "bar",
			Value: "bar",
		},
	}

	type testData struct {
		debugParams string
		debugValue  bool
	}

	data := []testData{
		{testQemuKernelParamsNonDebug, false},
		{testQemuKernelParamsDebug, true},
	}

	for _, d := range data {
		// kernel params consist of a default set of params,
		// followed by a set of params that depend on whether
		// debug mode is enabled and end with any user-supplied
		// params.
		expected := []string{testQemuKernelParamsBase, d.debugParams, suffixStr}

		expectedOut := strings.Join(expected, " ")

		testQemuBuildKernelParams(t, suffixParams, expectedOut, d.debugValue)
	}
}

func testQemuAppend(t *testing.T, structure interface{}, expected []ciaoQemu.Device, devType deviceType, nestedVM bool) {
	var devices []ciaoQemu.Device
	q := &qemu{
		nestedRun: nestedVM,
	}

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
	case Drive:
		devices = q.appendBlockDevice(devices, s)
	case VFIODevice:
		devices = q.appendVFIODevice(devices, s)
	}

	if reflect.DeepEqual(devices, expected) == false {
		t.Fatalf("Got %v\nExpecting %v", devices, expected)
	}
}

func TestQemuAppendVolume(t *testing.T) {
	mountTag := "testMountTag"
	hostPath := "testHostPath"
	nestedVM := true

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-9p-%s", mountTag),
			Path:          hostPath,
			MountTag:      mountTag,
			SecurityModel: ciaoQemu.None,
			DisableModern: nestedVM,
		},
	}

	volume := Volume{
		MountTag: mountTag,
		HostPath: hostPath,
	}

	testQemuAppend(t, volume, expectedOut, -1, nestedVM)
}

func TestQemuAppendSocket(t *testing.T) {
	deviceID := "channelTest"
	id := "charchTest"
	hostPath := "/tmp/hyper_test.sock"
	name := "sh.hyper.channel.test"
	nestedVM := true

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

	testQemuAppend(t, socket, expectedOut, -1, nestedVM)
}

func TestQemuAppendBlockDevice(t *testing.T) {
	id := "blockDevTest"
	file := "/root"
	format := "raw"
	nestedVM := true

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.BlockDevice{
			Driver:        ciaoQemu.VirtioBlock,
			ID:            id,
			File:          "/root",
			AIO:           ciaoQemu.Threads,
			Format:        ciaoQemu.BlockDeviceFormat(format),
			Interface:     "none",
			DisableModern: nestedVM,
		},
	}

	drive := Drive{
		File:   file,
		Format: format,
		ID:     id,
	}

	testQemuAppend(t, drive, expectedOut, -1, nestedVM)
}

func TestQemuAppendVFIODevice(t *testing.T) {
	nestedVM := true
	bdf := "02:10.1"

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.VFIODevice{
			BDF: bdf,
		},
	}

	vfDevice := VFIODevice{
		BDF: bdf,
	}

	testQemuAppend(t, vfDevice, expectedOut, -1, nestedVM)
}

func TestQemuAppendFSDevices(t *testing.T) {
	podID := "testPodID"
	contID := "testContID"
	contRootFs := "testContRootFs"
	volMountTag := "testVolMountTag"
	volHostPath := "testVolHostPath"
	nestedVM := true

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            "ctr-9p-0",
			Path:          fmt.Sprintf("%s.1", contRootFs),
			MountTag:      "ctr-rootfs-0",
			SecurityModel: ciaoQemu.None,
			DisableModern: nestedVM,
		},
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            "ctr-9p-1",
			Path:          fmt.Sprintf("%s.2", contRootFs),
			MountTag:      "ctr-rootfs-1",
			SecurityModel: ciaoQemu.None,
			DisableModern: nestedVM,
		},
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-9p-%s", fmt.Sprintf("%s.1", volMountTag)),
			Path:          fmt.Sprintf("%s.1", volHostPath),
			MountTag:      fmt.Sprintf("%s.1", volMountTag),
			SecurityModel: ciaoQemu.None,
			DisableModern: nestedVM,
		},
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-9p-%s", fmt.Sprintf("%s.2", volMountTag)),
			Path:          fmt.Sprintf("%s.2", volHostPath),
			MountTag:      fmt.Sprintf("%s.2", volMountTag),
			SecurityModel: ciaoQemu.None,
			DisableModern: nestedVM,
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

	testQemuAppend(t, podConfig, expectedOut, fsDev, nestedVM)
}

func TestQemuAppendConsoles(t *testing.T) {
	podID := "testPodID"
	nestedVM := true

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.SerialDevice{
			Driver:        ciaoQemu.VirtioSerial,
			ID:            "serial0",
			DisableModern: nestedVM,
		},
		ciaoQemu.CharDevice{
			Driver:   ciaoQemu.Console,
			Backend:  ciaoQemu.Socket,
			DeviceID: "console0",
			ID:       "charconsole0",
			Path:     filepath.Join(runStoragePath, podID, defaultConsole),
		},
	}

	podConfig := PodConfig{
		ID:         podID,
		Containers: []ContainerConfig{},
	}

	testQemuAppend(t, podConfig, expectedOut, consoleDev, nestedVM)
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
		t.Fatalf("Got %v\nExpecting %v", devices, expectedOut)
	}
}

func TestQemuInit(t *testing.T) {
	qemuConfig := newQemuConfig()
	q := &qemu{}

	pod := &Pod{
		id:      "testPod",
		storage: &filesystem{},
		config: &PodConfig{
			HypervisorConfig: qemuConfig,
		},
	}

	// Create parent dir path for hypervisor.json
	parentDir := filepath.Join(runStoragePath, pod.id)
	if err := os.MkdirAll(parentDir, dirMode); err != nil {
		t.Fatalf("Could not create parent directory %s: %v", parentDir, err)
	}

	if err := q.init(pod); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(parentDir); err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(qemuConfig, q.config) == false {
		t.Fatalf("Got %v\nExpecting %v", q.config, qemuConfig)
	}

	if reflect.DeepEqual(qemuConfig.HypervisorPath, q.path) == false {
		t.Fatalf("Got %v\nExpecting %v", q.path, qemuConfig.HypervisorPath)
	}

	// non-debug is the default
	var testQemuKernelParamsDefault = testQemuKernelParamsBase + " " + testQemuKernelParamsNonDebug

	if strings.Join(q.kernelParams, " ") != testQemuKernelParamsDefault {
		t.Fatal()
	}
}

func TestQemuInitMissingParentDirFail(t *testing.T) {
	qemuConfig := newQemuConfig()
	q := &qemu{}

	pod := &Pod{
		id:      "testPod",
		storage: &filesystem{},
		config: &PodConfig{
			HypervisorConfig: qemuConfig,
		},
	}

	// Ensure parent dir path for hypervisor.json does not exist.
	parentDir := filepath.Join(runStoragePath, pod.id)
	if err := os.RemoveAll(parentDir); err != nil {
		t.Fatal(err)
	}

	if err := q.init(pod); err == nil {
		t.Fatal("Qemu init() expected to fail because of missing parent directory for storage")
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
		t.Fatalf("Got %v\nExpecting %v", smp, expectedOut)
	}
}

func TestQemuSetMemoryResources(t *testing.T) {
	mem := 1000

	q := &qemu{}

	hostMemKb, err := getHostMemorySizeKb(procMemInfo)
	if err != nil {
		t.Fatal(err)
	}
	memMax := fmt.Sprintf("%dM", int(float64(hostMemKb)/1024)+maxMemoryOffset)

	expectedOut := ciaoQemu.Memory{
		Size:   "1000M",
		Slots:  uint8(2),
		MaxMem: memMax,
	}

	vmConfig := Resources{
		Memory: uint(mem),
	}

	podConfig := PodConfig{
		VMConfig: vmConfig,
	}

	memory, err := q.setMemoryResources(podConfig)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(memory, expectedOut) == false {
		t.Fatalf("Got %v\nExpecting %v", memory, expectedOut)
	}
}

func testQemuAddDevice(t *testing.T, devInfo interface{}, devType deviceType, expected []ciaoQemu.Device, nestedVM bool) {
	q := &qemu{
		nestedRun: nestedVM,
	}

	err := q.addDevice(devInfo, devType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(q.qemuConfig.Devices, expected) == false {
		t.Fatalf("Got %v\nExpecting %v", q.qemuConfig.Devices, expected)
	}
}

func TestQemuAddDeviceFsDev(t *testing.T) {
	mountTag := "testMountTag"
	hostPath := "testHostPath"
	nestedVM := true

	expectedOut := []ciaoQemu.Device{
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            fmt.Sprintf("extra-9p-%s", mountTag),
			Path:          hostPath,
			MountTag:      mountTag,
			SecurityModel: ciaoQemu.None,
			DisableModern: nestedVM,
		},
	}

	volume := Volume{
		MountTag: mountTag,
		HostPath: hostPath,
	}

	testQemuAddDevice(t, volume, fsDev, expectedOut, nestedVM)
}

func TestQemuAddDeviceSerialPordDev(t *testing.T) {
	deviceID := "channelTest"
	id := "charchTest"
	hostPath := "/tmp/hyper_test.sock"
	name := "sh.hyper.channel.test"
	nestedVM := true

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

	testQemuAddDevice(t, socket, serialPortDev, expectedOut, nestedVM)
}

func TestQemuGetPodConsole(t *testing.T) {
	q := &qemu{}
	podID := "testPodID"
	expected := filepath.Join(runStoragePath, podID, defaultConsole)

	if result := q.getPodConsole(podID); result != expected {
		t.Fatalf("Got %s\nExpecting %s", result, expected)
	}
}

func TestQemuMachineTypes(t *testing.T) {
	type testData struct {
		machineType string
		expectValid bool
	}

	data := []testData{
		{"pc-lite", true},
		{"pc", true},
		{"q35", true},

		{"PC-LITE", false},
		{"PC", false},
		{"Q35", false},
		{"", false},
		{" ", false},
		{".", false},
		{"0", false},
		{"1", false},
		{"-1", false},
		{"bon", false},
	}

	q := &qemu{}

	for _, d := range data {
		m, err := q.getMachine(d.machineType)

		if d.expectValid == true {
			if err != nil {
				t.Fatalf("machine type %v unexpectedly invalid: %v", d.machineType, err)
			}

			if m.Type != d.machineType {
				t.Fatalf("expected machine type %v, got %v", d.machineType, m.Type)
			}
		} else {
			if err == nil {
				t.Fatalf("machine type %v unexpectedly valid", d.machineType)
			}
		}
	}
}

func TestQemuBlockHotplugCapabilities(t *testing.T) {
	type testData struct {
		machineType     string
		expectedSupport bool
	}

	data := []testData{
		{"pc-lite", false},
		{"q35", false},
		{"pc", true},

		{"PC-LITE", false},
		{"PC", false},
		{"Q35", false},
		{"", false},
		{" ", false},
		{".", false},
		{"0", false},
		{"1", false},
		{"-1", false},
	}

	q := &qemu{}

	for _, d := range data {
		q.qemuConfig.Machine.Type = d.machineType

		caps := q.capabilities()
		isSupported := caps.isBlockDeviceHotplugSupported()
		if isSupported != d.expectedSupport {
			t.Fatalf("expected blockdevice hotplug support : %v, got %v", d.expectedSupport, isSupported)
		}
	}
}
