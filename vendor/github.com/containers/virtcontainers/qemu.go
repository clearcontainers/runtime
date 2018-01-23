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
	"strings"
	"sync"
	"time"

	"github.com/containers/virtcontainers/pkg/uuid"
	govmmQemu "github.com/intel/govmm/qemu"
	"github.com/sirupsen/logrus"
)

type qmpChannel struct {
	ctx  context.Context
	path string
	wg   sync.WaitGroup
	qmp  *govmmQemu.QMP
}

// QemuState keeps Qemu's state
type QemuState struct {
	Bridges []Bridge
	UUID    string
}

// qemu is an Hypervisor interface implementation for the Linux qemu hypervisor.
type qemu struct {
	path   string
	config HypervisorConfig

	hypervisorParams []string
	kernelParams     []string

	qmpMonitorCh qmpChannel
	qmpControlCh qmpChannel

	qemuConfig govmmQemu.Config

	nestedRun bool

	pod *Pod

	state QemuState
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

const qmpCapErrMsg = "Failed to negoatiate QMP capabilities"

const qmpSockPathSizeLimit = 107

// Mapping between machine types and QEMU binary paths.
var qemuPaths = map[string]string{
	QemuPCLite: "/usr/bin/qemu-lite-system-x86_64",
	QemuPC:     defaultQemuPath,
	QemuQ35:    "/usr/bin/qemu-35-system-x86_64",
}

var supportedQemuMachines = []govmmQemu.Machine{
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

type qmpLogger struct {
	logger *logrus.Entry
}

func newQMPLogger() qmpLogger {
	return qmpLogger{
		logger: virtLog.WithField("subsystem", "qmp"),
	}
}

func (l qmpLogger) V(level int32) bool {
	if level != 0 {
		return true
	}

	return false
}

func (l qmpLogger) Infof(format string, v ...interface{}) {
	l.logger.Infof(format, v...)
}

func (l qmpLogger) Warningf(format string, v ...interface{}) {
	l.logger.Warnf(format, v...)
}

func (l qmpLogger) Errorf(format string, v ...interface{}) {
	l.logger.Errorf(format, v...)
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
	{"pci", "lastbus=0"},
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

// Logger returns a logrus logger appropriate for logging qemu messages
func (q *qemu) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "qemu")
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

func (q *qemu) appendVolume(devices []govmmQemu.Device, volume Volume) []govmmQemu.Device {
	if volume.MountTag == "" || volume.HostPath == "" {
		return devices
	}

	devID := fmt.Sprintf("extra-9p-%s", volume.MountTag)
	if len(devID) > maxDevIDSize {
		devID = string(devID[:maxDevIDSize])
	}

	devices = append(devices,
		govmmQemu.FSDevice{
			Driver:        govmmQemu.Virtio9P,
			FSDriver:      govmmQemu.Local,
			ID:            devID,
			Path:          volume.HostPath,
			MountTag:      volume.MountTag,
			SecurityModel: govmmQemu.None,
			DisableModern: q.nestedRun,
		},
	)

	return devices
}

func (q *qemu) appendBlockDevice(devices []govmmQemu.Device, drive Drive) []govmmQemu.Device {
	if drive.File == "" || drive.ID == "" || drive.Format == "" {
		return devices
	}

	if len(drive.ID) > maxDevIDSize {
		drive.ID = string(drive.ID[:maxDevIDSize])
	}

	devices = append(devices,
		govmmQemu.BlockDevice{
			Driver:        govmmQemu.VirtioBlock,
			ID:            drive.ID,
			File:          drive.File,
			AIO:           govmmQemu.Threads,
			Format:        govmmQemu.BlockDeviceFormat(drive.Format),
			Interface:     "none",
			DisableModern: q.nestedRun,
		},
	)

	return devices
}

func (q *qemu) appendVhostUserDevice(devices []govmmQemu.Device, vhostUserDevice VhostUserDevice) []govmmQemu.Device {

	qemuVhostUserDevice := govmmQemu.VhostUserDevice{}

	switch vhostUserDevice := vhostUserDevice.(type) {
	case *VhostUserNetDevice:
		qemuVhostUserDevice.TypeDevID = makeNameID("net", vhostUserDevice.ID)
		qemuVhostUserDevice.Address = vhostUserDevice.MacAddress
	case *VhostUserSCSIDevice:
		qemuVhostUserDevice.TypeDevID = makeNameID("scsi", vhostUserDevice.ID)
	case *VhostUserBlkDevice:
	}

	qemuVhostUserDevice.VhostUserType = govmmQemu.VhostUserDeviceType(vhostUserDevice.Type())
	qemuVhostUserDevice.SocketPath = vhostUserDevice.Attrs().SocketPath
	qemuVhostUserDevice.CharDevID = makeNameID("char", vhostUserDevice.Attrs().ID)

	devices = append(devices, qemuVhostUserDevice)

	return devices
}
func (q *qemu) appendVFIODevice(devices []govmmQemu.Device, vfDevice VFIODevice) []govmmQemu.Device {
	if vfDevice.BDF == "" {
		return devices
	}

	devices = append(devices,
		govmmQemu.VFIODevice{
			BDF: vfDevice.BDF,
		},
	)

	return devices
}

func (q *qemu) appendSocket(devices []govmmQemu.Device, socket Socket) []govmmQemu.Device {
	devID := socket.ID
	if len(devID) > maxDevIDSize {
		devID = string(devID[:maxDevIDSize])
	}

	devices = append(devices,
		govmmQemu.CharDevice{
			Driver:   govmmQemu.VirtioSerialPort,
			Backend:  govmmQemu.Socket,
			DeviceID: socket.DeviceID,
			ID:       devID,
			Path:     socket.HostPath,
			Name:     socket.Name,
		},
	)

	return devices
}

func networkModelToQemuType(model NetInterworkingModel) govmmQemu.NetDeviceType {
	switch model {
	case ModelBridged:
		return govmmQemu.MACVTAP //TODO: We should rename MACVTAP to .NET_FD
	case ModelMacVtap:
		return govmmQemu.MACVTAP
	//case ModelEnlightened:
	// Here the Network plugin will create a VM native interface
	// which could be MacVtap, IpVtap, SRIOV, veth-tap, vhost-user
	// In these cases we will determine the interface type here
	// and pass in the native interface through
	default:
		//TAP should work for most other cases
		return govmmQemu.TAP
	}
}

var networkIndex = 0

func (q *qemu) appendNetwork(devices []govmmQemu.Device, endpoint Endpoint) []govmmQemu.Device {
	switch ep := endpoint.(type) {
	case *VirtualEndpoint:
		devices = append(devices,
			govmmQemu.NetDevice{
				Type:          networkModelToQemuType(ep.NetPair.NetInterworkingModel),
				Driver:        govmmQemu.VirtioNetPCI,
				ID:            fmt.Sprintf("network-%d", networkIndex),
				IFName:        ep.NetPair.TAPIface.Name,
				MACAddress:    ep.NetPair.TAPIface.HardAddr,
				DownScript:    "no",
				Script:        "no",
				VHost:         true,
				DisableModern: q.nestedRun,
				FDs:           ep.NetPair.VMFds,
				VhostFDs:      ep.NetPair.VhostFds,
			},
		)
		networkIndex++
	}

	return devices
}

func (q *qemu) appendFSDevices(devices []govmmQemu.Device, podConfig PodConfig) []govmmQemu.Device {
	// Add the shared volumes
	for _, v := range podConfig.Volumes {
		devices = q.appendVolume(devices, v)
	}

	return devices
}

func (q *qemu) appendBridges(devices []govmmQemu.Device, podConfig PodConfig) ([]govmmQemu.Device, error) {
	bus := "pci.0"
	if podConfig.HypervisorConfig.HypervisorMachineType == QemuQ35 {
		bus = "pcie.0"
	}

	for idx, b := range q.state.Bridges {
		t := govmmQemu.PCIBridge
		if b.Type == pcieBridge {
			t = govmmQemu.PCIEBridge
		}

		devices = append(devices,
			govmmQemu.BridgeDevice{
				Type: t,
				Bus:  bus,
				ID:   b.ID,
				// Each bridge is required to be assigned a unique chassis id > 0
				Chassis: (idx + 1),
				SHPC:    true,
			},
		)
	}

	return devices, nil
}

func (q *qemu) appendConsoles(devices []govmmQemu.Device, podConfig PodConfig) []govmmQemu.Device {
	serial := govmmQemu.SerialDevice{
		Driver:        govmmQemu.VirtioSerial,
		ID:            "serial0",
		DisableModern: q.nestedRun,
	}

	devices = append(devices, serial)

	var console govmmQemu.CharDevice

	console = govmmQemu.CharDevice{
		Driver:   govmmQemu.Console,
		Backend:  govmmQemu.Socket,
		DeviceID: "console0",
		ID:       "charconsole0",
		Path:     q.getPodConsole(podConfig.ID),
	}

	devices = append(devices, console)

	return devices
}

func (q *qemu) appendImage(devices []govmmQemu.Device, podConfig PodConfig) ([]govmmQemu.Device, error) {
	imagePath, err := q.config.ImageAssetPath()
	if err != nil {
		return nil, err
	}

	imageFile, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer imageFile.Close()

	imageStat, err := imageFile.Stat()
	if err != nil {
		return nil, err
	}

	object := govmmQemu.Object{
		Driver:   govmmQemu.NVDIMM,
		Type:     govmmQemu.MemoryBackendFile,
		DeviceID: "nv0",
		ID:       "mem0",
		MemPath:  imagePath,
		Size:     (uint64)(imageStat.Size()),
	}

	devices = append(devices, object)

	return devices, nil
}

func (q *qemu) getMachine(name string) (govmmQemu.Machine, error) {
	for _, m := range supportedQemuMachines {
		if m.Type == name {
			return m, nil
		}
	}

	return govmmQemu.Machine{}, fmt.Errorf("unrecognised machine type: %v", name)
}

// Build the QEMU binary path
func (q *qemu) buildPath() error {
	p, err := q.config.HypervisorAssetPath()
	if err != nil {
		return err
	}

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
		q.Logger().WithField("machine-type", machineType).Warn("Unknown machine type")
		p = defaultQemuPath
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("QEMU path (%s) does not exist", p)
	}

	q.path = p

	return nil
}

// init intializes the Qemu structure.
func (q *qemu) init(pod *Pod) error {
	valid, err := pod.config.HypervisorConfig.valid()
	if valid == false || err != nil {
		return err
	}

	q.config = pod.config.HypervisorConfig
	q.pod = pod

	if err := pod.storage.fetchHypervisorState(pod.id, &q.state); err != nil {
		q.Logger().Debug("Creating bridges")
		q.state.Bridges = NewBridges(q.config.DefaultBridges, q.config.HypervisorMachineType)

		q.Logger().Debug("Creating UUID")
		q.state.UUID = uuid.Generate().String()

		if err := pod.storage.storeHypervisorState(pod.id, q.state); err != nil {
			return err
		}
	}

	if err := q.buildPath(); err != nil {
		return err
	}

	if err := q.buildKernelParams(q.config); err != nil {
		return err
	}

	nested, err := RunningOnVMM(procCPUInfo)
	if err != nil {
		return err
	}

	if q.config.DisableNestingChecks {
		//Intentionally ignore the nesting check
		q.Logger().WithField("inside-vm", fmt.Sprintf("%t", nested)).Debug("Disable nesting environment checksx")
		q.nestedRun = false
	} else {
		q.nestedRun = nested
	}

	return nil
}

func (q *qemu) setCPUResources(podConfig PodConfig) govmmQemu.SMP {
	vcpus := q.config.DefaultVCPUs
	if podConfig.VMConfig.VCPUs > 0 {
		vcpus = uint32(podConfig.VMConfig.VCPUs)
	}

	smp := govmmQemu.SMP{
		CPUs:    vcpus,
		Cores:   vcpus,
		Sockets: defaultSockets,
		Threads: defaultThreads,
	}

	return smp
}

func (q *qemu) setMemoryResources(podConfig PodConfig) (govmmQemu.Memory, error) {
	hostMemKb, err := getHostMemorySizeKb(procMemInfo)
	if err != nil {
		return govmmQemu.Memory{}, fmt.Errorf("Unable to read memory info: %s", err)
	}
	if hostMemKb == 0 {
		return govmmQemu.Memory{}, fmt.Errorf("Error host memory size 0")
	}

	// add 1G memory space for nvdimm device (vm guest image)
	memMax := fmt.Sprintf("%dM", int(float64(hostMemKb)/1024)+maxMemoryOffset)
	mem := fmt.Sprintf("%dM", q.config.DefaultMemSz)
	if podConfig.VMConfig.Memory > 0 {
		mem = fmt.Sprintf("%dM", podConfig.VMConfig.Memory)
	}

	memory := govmmQemu.Memory{
		Size:   mem,
		Slots:  defaultMemSlots,
		MaxMem: memMax,
	}

	return memory, nil
}

func (q *qemu) qmpSocketPath(socketName string) (string, error) {
	parentDirPath := filepath.Join(runStoragePath, q.pod.id)
	if len(parentDirPath) > qmpSockPathSizeLimit {
		return "", fmt.Errorf("Parent directory path %q is too long "+
			"(%d characters), could not add any path for the QMP socket",
			parentDirPath, len(parentDirPath))
	}

	path := fmt.Sprintf("%s/%s-%s", parentDirPath, socketName, q.state.UUID)

	if len(path) > qmpSockPathSizeLimit {
		return path[:qmpSockPathSizeLimit], nil
	}

	return path, nil
}

// createPod is the Hypervisor pod creation implementation for govmmQemu.
func (q *qemu) createPod(podConfig PodConfig) error {
	var devices []govmmQemu.Device

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

	knobs := govmmQemu.Knobs{
		NoUserConfig: true,
		NoDefaults:   true,
		NoGraphic:    true,
		Daemonize:    true,
		MemPrealloc:  q.config.MemPrealloc,
		HugePages:    q.config.HugePages,
		Realtime:     q.config.Realtime,
		Mlock:        q.config.Mlock,
	}

	kernelPath, err := q.config.KernelAssetPath()
	if err != nil {
		return err
	}

	kernel := govmmQemu.Kernel{
		Path:   kernelPath,
		Params: strings.Join(q.kernelParams, " "),
	}

	rtc := govmmQemu.RTC{
		Base:     "utc",
		DriftFix: "slew",
	}

	if q.state.UUID == "" {
		return fmt.Errorf("UUID should not be empty")
	}

	monitorSockPath, err := q.qmpSocketPath(monitorSocket)
	if err != nil {
		return err
	}

	q.qmpMonitorCh = qmpChannel{
		ctx:  context.Background(),
		path: monitorSockPath,
	}

	controlSockPath, err := q.qmpSocketPath(controlSocket)
	if err != nil {
		return err
	}

	q.qmpControlCh = qmpChannel{
		ctx:  context.Background(),
		path: controlSockPath,
	}

	qmpSockets := []govmmQemu.QMPSocket{
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

	devices, err = q.appendBridges(devices, podConfig)
	if err != nil {
		return err
	}

	cpuModel := "host"
	if q.nestedRun {
		cpuModel += ",pmu=off"
	}

	firmwarePath, err := podConfig.HypervisorConfig.FirmwareAssetPath()
	if err != nil {
		return err
	}

	qemuConfig := govmmQemu.Config{
		Name:        fmt.Sprintf("pod-%s", podConfig.ID),
		UUID:        q.state.UUID,
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
		Bios:        firmwarePath,
	}

	q.qemuConfig = qemuConfig

	return nil
}

// startPod will start the Pod's VM.
func (q *qemu) startPod() error {
	strErr, err := govmmQemu.LaunchQemu(q.qemuConfig, newQMPLogger())
	if err != nil {
		return fmt.Errorf("%s", strErr)
	}

	return nil
}

// waitPod will wait for the Pod's VM to be up and running.
func (q *qemu) waitPod(timeout int) error {
	defer func(qemu *qemu) {
		if q.qmpMonitorCh.qmp != nil {
			q.qmpMonitorCh.qmp.Shutdown()
		}
	}(q)

	if timeout < 0 {
		return fmt.Errorf("Invalid timeout %ds", timeout)
	}

	cfg := govmmQemu.QMPConfig{Logger: newQMPLogger()}

	var qmp *govmmQemu.QMP
	var ver *govmmQemu.QMPVersion
	var err error

	timeStart := time.Now()
	for {
		disconnectCh := make(chan struct{})
		qmp, ver, err = govmmQemu.QMPStart(q.qmpMonitorCh.ctx, q.qmpMonitorCh.path, cfg, disconnectCh)
		if err == nil {
			break
		}

		if int(time.Now().Sub(timeStart).Seconds()) > timeout {
			return fmt.Errorf("Failed to connect to QEMU instance (timeout %ds): %v", timeout, err)
		}

		time.Sleep(time.Duration(50) * time.Millisecond)
	}

	q.qmpMonitorCh.qmp = qmp

	q.Logger().WithFields(logrus.Fields{
		"qmp-major-version": ver.Major,
		"qmp-minor-version": ver.Minor,
		"qmp-micro-version": ver.Micro,
		"qmp-capabilities":  strings.Join(ver.Capabilities, ","),
	}).Infof("QMP details")

	if err = q.qmpMonitorCh.qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx); err != nil {
		q.Logger().WithError(err).Error(qmpCapErrMsg)
		return err
	}

	return nil
}

// stopPod will stop the Pod's VM.
func (q *qemu) stopPod() error {
	cfg := govmmQemu.QMPConfig{Logger: newQMPLogger()}
	disconnectCh := make(chan struct{})

	q.Logger().Info("Stopping Pod")
	qmp, _, err := govmmQemu.QMPStart(q.qmpControlCh.ctx, q.qmpControlCh.path, cfg, disconnectCh)
	if err != nil {
		q.Logger().WithError(err).Error("Failed to connect to QEMU instance")
		return err
	}

	err = qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		q.Logger().WithError(err).Error(qmpCapErrMsg)
		return err
	}

	return qmp.ExecuteQuit(q.qmpMonitorCh.ctx)
}

func (q *qemu) togglePausePod(pause bool) error {
	defer func(qemu *qemu) {
		if q.qmpMonitorCh.qmp != nil {
			q.qmpMonitorCh.qmp.Shutdown()
		}
	}(q)

	cfg := govmmQemu.QMPConfig{Logger: newQMPLogger()}

	// Auto-closed by QMPStart().
	disconnectCh := make(chan struct{})

	qmp, _, err := govmmQemu.QMPStart(q.qmpControlCh.ctx, q.qmpControlCh.path, cfg, disconnectCh)
	if err != nil {
		q.Logger().WithError(err).Error("Failed to connect to QEMU instance")
		return err
	}

	q.qmpMonitorCh.qmp = qmp

	err = qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		q.Logger().WithError(err).Error(qmpCapErrMsg)
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

func (q *qemu) qmpSetup() (*govmmQemu.QMP, error) {
	cfg := govmmQemu.QMPConfig{Logger: newQMPLogger()}

	// Auto-closed by QMPStart().
	disconnectCh := make(chan struct{})

	qmp, _, err := govmmQemu.QMPStart(q.qmpControlCh.ctx, q.qmpControlCh.path, cfg, disconnectCh)
	if err != nil {
		q.Logger().WithError(err).Error("Failed to connect to QEMU instance")
		return nil, err
	}

	err = qmp.ExecuteQMPCapabilities(q.qmpMonitorCh.ctx)
	if err != nil {
		q.Logger().WithError(err).Error(qmpCapErrMsg)
		return nil, err
	}

	return qmp, nil
}

func (q *qemu) addDeviceToBridge(ID string) (string, string, error) {
	var err error
	var addr uint32

	// looking for an empty address in the bridges
	for _, b := range q.state.Bridges {
		addr, err = b.addDevice(ID)
		if err == nil {
			return fmt.Sprintf("0x%x", addr), b.ID, nil
		}
	}

	return "", "", err
}

func (q *qemu) removeDeviceFromBridge(ID string) error {
	var err error
	for _, b := range q.state.Bridges {
		err = b.removeDevice(ID)
		if err == nil {
			// device was removed correctly
			return nil
		}
	}

	return err
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

		addr, bus, err := q.addDeviceToBridge(drive.ID)
		if err != nil {
			return err
		}

		if err = q.qmpMonitorCh.qmp.ExecutePCIDeviceAdd(q.qmpMonitorCh.ctx, drive.ID, devID, driver, addr, bus); err != nil {
			return err
		}

	} else {
		if err := q.removeDeviceFromBridge(drive.ID); err != nil {
			return err
		}

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
	if err := q.hotplugDevice(devInfo, devType, addDevice); err != nil {
		return err
	}

	return q.pod.storage.storeHypervisorState(q.pod.id, q.state)
}

func (q *qemu) hotplugRemoveDevice(devInfo interface{}, devType deviceType) error {
	if err := q.hotplugDevice(devInfo, devType, removeDevice); err != nil {
		return err
	}

	return q.pod.storage.storeHypervisorState(q.pod.id, q.state)
}

func (q *qemu) pausePod() error {
	return q.togglePausePod(true)
}

func (q *qemu) resumePod() error {
	return q.togglePausePod(false)
}

// addDevice will add extra devices to Qemu command line.
func (q *qemu) addDevice(devInfo interface{}, devType deviceType) error {
	switch v := devInfo.(type) {
	case Volume:
		q.qemuConfig.Devices = q.appendVolume(q.qemuConfig.Devices, v)
	case Socket:
		q.qemuConfig.Devices = q.appendSocket(q.qemuConfig.Devices, v)
	case Endpoint:
		q.qemuConfig.Devices = q.appendNetwork(q.qemuConfig.Devices, v)
	case Drive:
		q.qemuConfig.Devices = q.appendBlockDevice(q.qemuConfig.Devices, v)

	//vhostUserDevice is an interface, hence the pointer for Net, SCSI and Blk:
	case VhostUserNetDevice:
		q.qemuConfig.Devices = q.appendVhostUserDevice(q.qemuConfig.Devices, &v)
	case VhostUserSCSIDevice:
		q.qemuConfig.Devices = q.appendVhostUserDevice(q.qemuConfig.Devices, &v)
	case VhostUserBlkDevice:
		q.qemuConfig.Devices = q.appendVhostUserDevice(q.qemuConfig.Devices, &v)
	case VFIODevice:
		q.qemuConfig.Devices = q.appendVFIODevice(q.qemuConfig.Devices, v)
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

func (q *qemu) getState() interface{} {
	return q.state
}
