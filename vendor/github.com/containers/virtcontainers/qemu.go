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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	ciaoQemu "github.com/01org/ciao/qemu"
	"github.com/01org/ciao/ssntp/uuid"
)

type qmpChannel struct {
	ctx          context.Context
	path         string
	disconnectCh chan struct{}
	wg           sync.WaitGroup
	qmp          *ciaoQemu.QMP
}

// qemu is an Hypervisor interface implementation for the Linux qemu hypervisor.
type qemu struct {
	path   string
	config HypervisorConfig

	hypervisorParams []string
	kernelParams     []string

	qmpMonitorCh qmpChannel
	qmpControlCh qmpChannel

	qemuConfig ciaoQemu.Config

	nestedRun bool
}

const defaultQemuPath = "/usr/bin/qemu-system-x86_64"

const defaultQemuMachineType = "pc-lite"

const defaultQemuMachineAccelerators = "kvm,kernel_irqchip,nvdimm"

const (
	// QemuPCLite is the QEMU pc-lite machine type
	QemuPCLite = defaultQemuMachineType

	// QemuPC is the QEMU pc machine type
	QemuPC = "pc"

	// QemuQ35 is the QEMU Q35 machine type
	QemuQ35 = "q35"
)

// Mapping between machine types and QEMU binary paths.
var qemuPaths = map[string]string{
	QemuPCLite: "/usr/bin/qemu-lite-system-x86_64",
	QemuPC:     defaultQemuPath,
	QemuQ35:    "/usr/bin/qemu-35-system-x86_64",
}

var supportedQemuMachines = []ciaoQemu.Machine{
	{
		Type:         QemuPCLite,
		Acceleration: defaultQemuMachineAccelerators,
	},
	{
		Type:         QemuPC,
		Acceleration: defaultQemuMachineAccelerators,
	},
	{
		Type:         QemuQ35,
		Acceleration: defaultQemuMachineAccelerators,
	},
}

const (
	defaultSockets uint32 = 1
	defaultThreads uint32 = 1
)

const (
	defaultMemSlots uint8 = 2
)

const (
	defaultConsole = "console.sock"
)

const (
	maxDevIDSize = 31
)

const (
	// NVDIMM device needs memory space 1024MB
	// See https://github.com/clearcontainers/runtime/issues/380
	maxMemoryOffset = 1024
)

type operation int

const (
	addDevice operation = iota
	removeDevice
)

type qmpLogger struct{}

func (l qmpLogger) V(level int32) bool {
	if level != 0 {
		return true
	}

	return false
}

func (l qmpLogger) Infof(format string, v ...interface{}) {
	virtLog.Infof(format, v...)
}

func (l qmpLogger) Warningf(format string, v ...interface{}) {
	virtLog.Warnf(format, v...)
}

func (l qmpLogger) Errorf(format string, v ...interface{}) {
	virtLog.Errorf(format, v...)
}

var kernelDefaultParams = []Param{
	{"root", "/dev/pmem0p1"},
	{"rootflags", "dax,data=ordered,errors=remount-ro rw"},
	{"rootfstype", "ext4"},
	{"tsc", "reliable"},
	{"no_timer_check", ""},
	{"rcupdate.rcu_expedited", "1"},
	{"i8042.direct", "1"},
	{"i8042.dumbkbd", "1"},
	{"i8042.nopnp", "1"},
	{"i8042.noaux", "1"},
	{"noreplace-smp", ""},
	{"reboot", "k"},
	{"panic", "1"},
	{"console", "hvc0"},
	{"console", "hvc1"},
	{"initcall_debug", ""},
	{"iommu", "off"},
	{"cryptomgr.notests", ""},
	{"net.ifnames", "0"},
}

// kernelDefaultParamsNonDebug is a list of the default kernel
// parameters that will be used in standard (non-debug) mode.
var kernelDefaultParamsNonDebug = []Param{
	{"quiet", ""},
	{"systemd.show_status", "false"},
}

// kernelDefaultParamsDebug is a list of the default kernel
// parameters that will be used in debug mode (as much boot output as
// possible).
var kernelDefaultParamsDebug = []Param{
	{"debug", ""},
	{"systemd.show_status", "true"},
	{"systemd.log_level", "debug"},
}

func (q *qemu) buildKernelParams(config HypervisorConfig) error {
	params := kernelDefaultParams

	if config.Debug == true {
		params = append(params, kernelDefaultParamsDebug...)
	} else {
		params = append(params, kernelDefaultParamsNonDebug...)
	}

	params = append(params, config.KernelParams...)

	q.kernelParams = SerializeParams(params, "=")

	return nil
}

// Adds all capabilities supported by qemu implementation of hypervisor interface
func (q *qemu) capabilities() capabilities {
	var caps capabilities

	// Only pc machine type supports hotplugging drives
	if q.qemuConfig.Machine.Type == QemuPC {
		caps.setBlockDeviceHotplugSupport()
	}

	return caps
}

func (q *qemu) appendVolume(devices []ciaoQemu.Device, volume Volume) []ciaoQemu.Device {
	if volume.MountTag == "" || volume.HostPath == "" {
		return devices
	}

	devID := fmt.Sprintf("extra-9p-%s", volume.MountTag)
	if len(devID) > maxDevIDSize {
		devID = string(devID[:maxDevIDSize])
	}

	devices = append(devices,
		ciaoQemu.FSDevice{
			Driver:        ciaoQemu.Virtio9P,
			FSDriver:      ciaoQemu.Local,
			ID:            devID,
			Path:          volume.HostPath,
			MountTag:      volume.MountTag,
			SecurityModel: ciaoQemu.None,
			DisableModern: q.nestedRun,
		},
	)

	return devices
}

func (q *qemu) appendBlockDevice(devices []ciaoQemu.Device, drive Drive) []ciaoQemu.Device {
	if drive.File == "" || drive.ID == "" || drive.Format == "" {
		return devices
	}

	if len(drive.ID) > maxDevIDSize {
		drive.ID = string(drive.ID[:maxDevIDSize])
	}

	devices = append(devices,
		ciaoQemu.BlockDevice{
			Driver:        ciaoQemu.VirtioBlock,
			ID:            drive.ID,
			File:          drive.File,
			AIO:           ciaoQemu.Threads,
			Format:        ciaoQemu.BlockDeviceFormat(drive.Format),
			Interface:     "none",
			DisableModern: q.nestedRun,
		},
	)

	return devices
}

func (q *qemu) appendVFIODevice(devices []ciaoQemu.Device, vfDevice VFIODevice) []ciaoQemu.Device {
	if vfDevice.BDF == "" {
		return devices
	}

	devices = append(devices,
		ciaoQemu.VFIODevice{
			BDF: vfDevice.BDF,
		},
	)

	return devices
}

func (q *qemu) appendSocket(devices []ciaoQemu.Device, socket Socket) []ciaoQemu.Device {
	devID := socket.ID
	if len(devID) > maxDevIDSize {
		devID = string(devID[:maxDevIDSize])
	}

	devices = append(devices,
		ciaoQemu.CharDevice{
			Driver:   ciaoQemu.VirtioSerialPort,
			Backend:  ciaoQemu.Socket,
			DeviceID: socket.DeviceID,
			ID:       devID,
			Path:     socket.HostPath,
			Name:     socket.Name,
		},
	)

	return devices
}

func (q *qemu) appendNetworks(devices []ciaoQemu.Device, endpoints []Endpoint) []ciaoQemu.Device {
	for idx, endpoint := range endpoints {
		devices = append(devices,
			ciaoQemu.NetDevice{
				Type:          ciaoQemu.TAP,
				Driver:        ciaoQemu.VirtioNetPCI,
				ID:            fmt.Sprintf("network-%d", idx),
				IFName:        endpoint.NetPair.TAPIface.Name,
				MACAddress:    endpoint.NetPair.TAPIface.HardAddr,
				DownScript:    "no",
				Script:        "no",
				VHost:         true,
				DisableModern: q.nestedRun,
			},
		)
	}

	return devices
}

func (q *qemu) appendFSDevices(devices []ciaoQemu.Device, podConfig PodConfig) []ciaoQemu.Device {
	// Add the containers rootfs
	for idx, c := range podConfig.Containers {
		if c.RootFs == "" || c.ID == "" {
			continue
		}

		devices = append(devices,
			ciaoQemu.FSDevice{
				Driver:        ciaoQemu.Virtio9P,
				FSDriver:      ciaoQemu.Local,
				ID:            fmt.Sprintf("ctr-9p-%d", idx),
				Path:          c.RootFs,
				MountTag:      fmt.Sprintf("ctr-rootfs-%d", idx),
				SecurityModel: ciaoQemu.None,
				DisableModern: q.nestedRun,
			},
		)
	}

	// Add the shared volumes
	for _, v := range podConfig.Volumes {
		devices = q.appendVolume(devices, v)
	}

	return devices
}

func (q *qemu) appendConsoles(devices []ciaoQemu.Device, podConfig PodConfig) []ciaoQemu.Device {
	serial := ciaoQemu.SerialDevice{
		Driver:        ciaoQemu.VirtioSerial,
		ID:            "serial0",
		DisableModern: q.nestedRun,
	}

	devices = append(devices, serial)

	var console ciaoQemu.CharDevice

	console = ciaoQemu.CharDevice{
		Driver:   ciaoQemu.Console,
		Backend:  ciaoQemu.Socket,
		DeviceID: "console0",
		ID:       "charconsole0",
		Path:     q.getPodConsole(podConfig.ID),
	}

	devices = append(devices, console)

	return devices
}

func (q *qemu) appendImage(devices []ciaoQemu.Device, podConfig PodConfig) ([]ciaoQemu.Device, error) {
	imageFile, err := os.Open(q.config.ImagePath)
	if err != nil {
		return nil, err
	}
	defer imageFile.Close()

	imageStat, err := imageFile.Stat()
	if err != nil {
		return nil, err
	}

	object := ciaoQemu.Object{
		Driver:   ciaoQemu.NVDIMM,
		Type:     ciaoQemu.MemoryBackendFile,
		DeviceID: "nv0",
		ID:       "mem0",
		MemPath:  q.config.ImagePath,
		Size:     (uint64)(imageStat.Size()),
	}

	devices = append(devices, object)

	return devices, nil
}

func (q *qemu) forceUUIDFormat(str string) string {
	re := regexp.MustCompile(`[^[0-9,a-f,A-F]]*`)
	hexStr := re.ReplaceAllLiteralString(str, ``)

	slice := []byte(hexStr)
	sliceLen := len(slice)

	var uuidSlice uuid.UUID
	uuidLen := len(uuidSlice)

	if sliceLen > uuidLen {
		copy(uuidSlice[:], slice[:uuidLen])
	} else {
		copy(uuidSlice[:], slice)
	}

	return uuidSlice.String()
}

func (q *qemu) getMachine(name string) (ciaoQemu.Machine, error) {
	for _, m := range supportedQemuMachines {
		if m.Type == name {
			return m, nil
		}
	}

	return ciaoQemu.Machine{}, fmt.Errorf("unrecognised machine type: %v", name)
}

// Build the QEMU binary path
func (q *qemu) buildPath() error {
	p := q.config.HypervisorPath
	if p != "" {
		q.path = p
		return nil
	}

	// We do not have a configured path, let's try to map one from the machine type
	machineType := q.config.HypervisorMachineType
	if machineType == "" {
		machineType = defaultQemuMachineType
	}

	p, ok := qemuPaths[machineType]
	if !ok {
		virtLog.Warnf("Unknown machine type %s", machineType)
		p = defaultQemuPath
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("QEMU path (%s) does not exist", p)
	}

	q.path = p

	return nil
}

// init intializes the Qemu structure.
func (q *qemu) init(config HypervisorConfig) error {
	valid, err := config.valid()
	if valid == false || err != nil {
		return err
	}

	q.config = config

	if err = q.buildPath(); err != nil {
		return err
	}

	if err = q.buildKernelParams(config); err != nil {
		return err
	}

	nested, err := RunningOnVMM(procCPUInfo)
	if err != nil {
		return err
	}

	virtLog.Debugf("Running inside a VM = %v", nested)

	if config.DisableNestingChecks {
		//Intentionally ignore the nesting check
		q.nestedRun = false
	} else {
		q.nestedRun = nested
	}

	return nil
}

func (q *qemu) qmpMonitor(connectedCh chan struct{}) {
	defer func(qemu *qemu) {
		if q.qmpMonitorCh.qmp != nil {
			q.qmpMonitorCh.qmp.Shutdown()
		}

		q.qmpMonitorCh.wg.Done()
	}(q)

	cfg := ciaoQemu.QMPConfig{Logger: qmpLogger{}}
	qmp, ver, err := ciaoQemu.QMPStart(q.qmpMonitorCh.ctx, q.qmpMonitorCh.path, cfg, q.qmpMonitorCh.disconnectCh)
	if err != nil {
		virtLog.Errorf("Failed to connect to QEMU instance %v", err)
		return
	}

	q.qmpMonitorCh.qmp = qmp

	virtLog.Infof("QMP version %d.%d.%d", ver.Major, ver.Minor, ver.Micro)
	virtLog.Infof("QMP capabilities %s", ver.Capabilities)

	err = q.qmpMonitorCh.qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		virtLog.Errorf("Unable to send qmp_capabilities command: %v", err)
		return
	}

	close(connectedCh)
}

func (q *qemu) setCPUResources(podConfig PodConfig) ciaoQemu.SMP {
	vcpus := q.config.DefaultVCPUs
	if podConfig.VMConfig.VCPUs > 0 {
		vcpus = uint32(podConfig.VMConfig.VCPUs)
	}

	smp := ciaoQemu.SMP{
		CPUs:    vcpus,
		Cores:   vcpus,
		Sockets: defaultSockets,
		Threads: defaultThreads,
	}

	return smp
}

func (q *qemu) setMemoryResources(podConfig PodConfig) (ciaoQemu.Memory, error) {
	hostMemKb, err := getHostMemorySizeKb(procMemInfo)
	if err != nil {
		return ciaoQemu.Memory{}, fmt.Errorf("Unable to read memory info: %s", err)
	}
	if hostMemKb == 0 {
		return ciaoQemu.Memory{}, fmt.Errorf("Error host memory size 0")
	}

	// add 1G memory space for nvdimm device (vm guest image)
	memMax := fmt.Sprintf("%dM", int(float64(hostMemKb)/1024)+maxMemoryOffset)
	mem := fmt.Sprintf("%dM", q.config.DefaultMemSz)
	if podConfig.VMConfig.Memory > 0 {
		mem = fmt.Sprintf("%dM", podConfig.VMConfig.Memory)
	}

	memory := ciaoQemu.Memory{
		Size:   mem,
		Slots:  defaultMemSlots,
		MaxMem: memMax,
	}

	return memory, nil
}

// createPod is the Hypervisor pod creation implementation for ciaoQemu.
func (q *qemu) createPod(podConfig PodConfig) error {
	var devices []ciaoQemu.Device

	machineType := q.config.HypervisorMachineType
	if machineType == "" {
		machineType = defaultQemuMachineType
	}

	machine, err := q.getMachine(machineType)
	if err != nil {
		return err
	}

	accelerators := podConfig.HypervisorConfig.MachineAccelerators
	if accelerators != "" {
		if !strings.HasPrefix(accelerators, ",") {
			accelerators = fmt.Sprintf(",%s", accelerators)
		}
		machine.Acceleration += accelerators
	}

	smp := q.setCPUResources(podConfig)

	memory, err := q.setMemoryResources(podConfig)
	if err != nil {
		return err
	}

	knobs := ciaoQemu.Knobs{
		NoUserConfig: true,
		NoDefaults:   true,
		NoGraphic:    true,
		Daemonize:    true,
		MemPrealloc:  q.config.MemPrealloc,
		HugePages:    q.config.HugePages,
		Realtime:     q.config.Realtime,
		Mlock:        q.config.Mlock,
	}

	kernel := ciaoQemu.Kernel{
		Path:   q.config.KernelPath,
		Params: strings.Join(q.kernelParams, " "),
	}

	rtc := ciaoQemu.RTC{
		Base:     "utc",
		DriftFix: "slew",
	}

	q.qmpMonitorCh = qmpChannel{
		ctx:  context.Background(),
		path: fmt.Sprintf("%s/%s/%s", runStoragePath, podConfig.ID, monitorSocket),
	}

	q.qmpControlCh = qmpChannel{
		ctx:  context.Background(),
		path: fmt.Sprintf("%s/%s/%s", runStoragePath, podConfig.ID, controlSocket),
	}

	qmpSockets := []ciaoQemu.QMPSocket{
		{
			Type:   "unix",
			Name:   q.qmpMonitorCh.path,
			Server: true,
			NoWait: true,
		},
		{
			Type:   "unix",
			Name:   q.qmpControlCh.path,
			Server: true,
			NoWait: true,
		},
	}

	devices = q.appendFSDevices(devices, podConfig)
	devices = q.appendConsoles(devices, podConfig)
	devices, err = q.appendImage(devices, podConfig)
	if err != nil {
		return err
	}

	cpuModel := "host"
	if q.nestedRun {
		cpuModel += ",pmu=off"
	}

	qemuConfig := ciaoQemu.Config{
		Name:        fmt.Sprintf("pod-%s", podConfig.ID),
		UUID:        q.forceUUIDFormat(podConfig.ID),
		Path:        q.path,
		Ctx:         q.qmpMonitorCh.ctx,
		Machine:     machine,
		SMP:         smp,
		Memory:      memory,
		Devices:     devices,
		CPUModel:    cpuModel,
		Kernel:      kernel,
		RTC:         rtc,
		QMPSockets:  qmpSockets,
		Knobs:       knobs,
		VGA:         "none",
		GlobalParam: "kvm-pit.lost_tick_policy=discard",
		Bios:        podConfig.HypervisorConfig.FirmwarePath,
	}

	q.qemuConfig = qemuConfig

	return nil
}

// startPod will start the Pod's VM.
func (q *qemu) startPod(startCh, stopCh chan struct{}) error {
	strErr, err := ciaoQemu.LaunchQemu(q.qemuConfig, qmpLogger{})
	if err != nil {
		return fmt.Errorf("%s", strErr)
	}

	// Start the QMP monitoring thread
	q.qmpMonitorCh.disconnectCh = stopCh
	q.qmpMonitorCh.wg.Add(1)
	q.qmpMonitor(startCh)

	return nil
}

// stopPod will stop the Pod's VM.
func (q *qemu) stopPod() error {
	cfg := ciaoQemu.QMPConfig{Logger: qmpLogger{}}
	q.qmpControlCh.disconnectCh = make(chan struct{})
	const timeout = time.Duration(10) * time.Second

	virtLog.Info("Stopping Pod")
	qmp, _, err := ciaoQemu.QMPStart(q.qmpControlCh.ctx, q.qmpControlCh.path, cfg, q.qmpControlCh.disconnectCh)
	if err != nil {
		virtLog.Errorf("Failed to connect to QEMU instance %v", err)
		return err
	}

	err = qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		virtLog.Errorf("Failed to negotiate capabilities with QEMU %v", err)
		return err
	}

	if err := qmp.ExecuteQuit(q.qmpMonitorCh.ctx); err != nil {
		return err
	}

	// Wait for the VM disconnection notification
	select {
	case <-q.qmpControlCh.disconnectCh:
		break
	case <-time.After(timeout):
		return fmt.Errorf("Did not receive the VM disconnection notification (timeout %ds)", timeout)
	}

	return nil
}

func (q *qemu) togglePausePod(pause bool) error {
	defer func(qemu *qemu) {
		if q.qmpMonitorCh.qmp != nil {
			q.qmpMonitorCh.qmp.Shutdown()
		}
	}(q)

	cfg := ciaoQemu.QMPConfig{Logger: qmpLogger{}}

	// Auto-closed by QMPStart().
	disconnectCh := make(chan struct{})

	qmp, _, err := ciaoQemu.QMPStart(q.qmpControlCh.ctx, q.qmpControlCh.path, cfg, disconnectCh)
	if err != nil {
		virtLog.Errorf("Failed to connect to QEMU instance %v", err)
		return err
	}

	q.qmpMonitorCh.qmp = qmp

	err = qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		virtLog.Errorf("Failed to negotiate capabilities with QEMU %v", err)
		return err
	}

	if pause {
		err = q.qmpMonitorCh.qmp.ExecuteStop(q.qmpMonitorCh.ctx)
	} else {
		err = q.qmpMonitorCh.qmp.ExecuteCont(q.qmpMonitorCh.ctx)
	}

	if err != nil {
		return err
	}

	return nil
}

func (q *qemu) qmpSetup() (*ciaoQemu.QMP, error) {
	cfg := ciaoQemu.QMPConfig{Logger: qmpLogger{}}

	// Auto-closed by QMPStart().
	disconnectCh := make(chan struct{})

	qmp, _, err := ciaoQemu.QMPStart(q.qmpControlCh.ctx, q.qmpControlCh.path, cfg, disconnectCh)
	if err != nil {
		virtLog.Errorf("Failed to connect to QEMU instance %v", err)
		return nil, err
	}

	err = qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		virtLog.Errorf("Failed to negotiate capabilities with QEMU %v", err)
		return nil, err
	}

	return qmp, nil
}

func (q *qemu) hotplugBlockDevice(drive Drive, op operation) error {
	defer func(qemu *qemu) {
		if q.qmpMonitorCh.qmp != nil {
			q.qmpMonitorCh.qmp.Shutdown()
		}
	}(q)

	qmp, err := q.qmpSetup()
	if err != nil {
		return err
	}

	q.qmpMonitorCh.qmp = qmp

	devID := "virtio-" + drive.ID

	if op == addDevice {
		if err := q.qmpMonitorCh.qmp.ExecuteBlockdevAdd(q.qmpMonitorCh.ctx, drive.File, drive.ID); err != nil {
			return err
		}

		driver := "virtio-blk-pci"
		if err := q.qmpMonitorCh.qmp.ExecuteDeviceAdd(q.qmpMonitorCh.ctx, drive.ID, devID, driver, ""); err != nil {
			return err
		}
	} else {
		if err := q.qmpMonitorCh.qmp.ExecuteDeviceDel(q.qmpMonitorCh.ctx, devID); err != nil {
			return err
		}

		if err := q.qmpMonitorCh.qmp.ExecuteBlockdevDel(q.qmpMonitorCh.ctx, drive.ID); err != nil {
			return err
		}
	}

	return nil
}

func (q *qemu) hotplugDevice(devInfo interface{}, devType deviceType, op operation) error {
	switch devType {
	case blockDev:
		drive := devInfo.(Drive)
		return q.hotplugBlockDevice(drive, op)
	default:
		return fmt.Errorf("Only hotplug for block devices supported for now, provided device type : %v", devType)
	}
}

func (q *qemu) hotplugAddDevice(devInfo interface{}, devType deviceType) error {
	return q.hotplugDevice(devInfo, devType, addDevice)
}

func (q *qemu) hotplugRemoveDevice(devInfo interface{}, devType deviceType) error {
	return q.hotplugDevice(devInfo, devType, removeDevice)
}

func (q *qemu) pausePod() error {
	return q.togglePausePod(true)
}

func (q *qemu) resumePod() error {
	return q.togglePausePod(false)
}

// addDevice will add extra devices to Qemu command line.
func (q *qemu) addDevice(devInfo interface{}, devType deviceType) error {
	switch devType {
	case fsDev:
		volume := devInfo.(Volume)
		q.qemuConfig.Devices = q.appendVolume(q.qemuConfig.Devices, volume)
	case serialPortDev:
		socket := devInfo.(Socket)
		q.qemuConfig.Devices = q.appendSocket(q.qemuConfig.Devices, socket)
	case netDev:
		endpoints := devInfo.([]Endpoint)
		q.qemuConfig.Devices = q.appendNetworks(q.qemuConfig.Devices, endpoints)
	case blockDev:
		drive := devInfo.(Drive)
		q.qemuConfig.Devices = q.appendBlockDevice(q.qemuConfig.Devices, drive)
	case vfioDev:
		vfDevice := devInfo.(VFIODevice)
		q.qemuConfig.Devices = q.appendVFIODevice(q.qemuConfig.Devices, vfDevice)
	default:
		break
	}

	return nil
}

// getPodConsole builds the path of the console where we can read
// logs coming from the pod.
func (q *qemu) getPodConsole(podID string) string {
	return filepath.Join(runStoragePath, podID, defaultConsole)
}
