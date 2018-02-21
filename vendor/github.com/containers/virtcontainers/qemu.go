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
	"time"

	"github.com/containers/virtcontainers/pkg/uuid"
	govmmQemu "github.com/intel/govmm/qemu"
	"github.com/sirupsen/logrus"
)

type qmpChannel struct {
	ctx  context.Context
	path string
	qmp  *govmmQemu.QMP
}

// QemuState keeps Qemu's state
type QemuState struct {
	Bridges []Bridge
	UUID    string
}

// qemu is an Hypervisor interface implementation for the Linux qemu hypervisor.
type qemu struct {
	config HypervisorConfig

	qmpMonitorCh qmpChannel
	qmpControlCh qmpChannel

	qemuConfig govmmQemu.Config

	pod *Pod

	state QemuState

	arch qemuArch
}

const qmpCapErrMsg = "Failed to negoatiate QMP capabilities"

const qmpSockPathSizeLimit = 107

const defaultConsole = "console.sock"

// agnostic list of kernel parameters
var defaultKernelParameters = []Param{
	{"panic", "1"},
	{"initcall_debug", ""},
}

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

// Logger returns a logrus logger appropriate for logging qemu messages
func (q *qemu) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "qemu")
}

func (q *qemu) kernelParameters() string {
	// get a list of arch kernel parameters
	params := q.arch.kernelParameters(q.config.Debug)

	// use default parameters
	params = append(params, defaultKernelParameters...)

	// add the params specified by the provided config. As the kernel
	// honours the last parameter value set and since the config-provided
	// params are added here, they will take priority over the defaults.
	params = append(params, q.config.KernelParams...)

	paramsStr := SerializeParams(params, "=")

	return strings.Join(paramsStr, " ")
}

// Adds all capabilities supported by qemu implementation of hypervisor interface
func (q *qemu) capabilities() capabilities {
	return q.arch.capabilities()
}

// get the QEMU binary path
func (q *qemu) qemuPath() (string, error) {
	p, err := q.config.HypervisorAssetPath()
	if err != nil {
		return "", err
	}

	if p == "" {
		p, err = q.arch.qemuPath()
		if err != nil {
			return "", err
		}
	}

	if _, err = os.Stat(p); os.IsNotExist(err) {
		return "", fmt.Errorf("QEMU path (%s) does not exist", p)
	}

	return p, nil
}

// init intializes the Qemu structure.
func (q *qemu) init(pod *Pod) error {
	valid, err := pod.config.HypervisorConfig.valid()
	if valid == false || err != nil {
		return err
	}

	q.config = pod.config.HypervisorConfig
	q.pod = pod
	q.arch = newQemuArch(q.config.HypervisorMachineType)

	if err = pod.storage.fetchHypervisorState(pod.id, &q.state); err != nil {
		q.Logger().Debug("Creating bridges")
		q.state.Bridges = q.arch.bridges(q.config.DefaultBridges)

		q.Logger().Debug("Creating UUID")
		q.state.UUID = uuid.Generate().String()

		if err = pod.storage.storeHypervisorState(pod.id, q.state); err != nil {
			return err
		}
	}

	nested, err := RunningOnVMM(procCPUInfo)
	if err != nil {
		return err
	}

	if !q.config.DisableNestingChecks && nested {
		q.arch.enableNestingChecks()
	} else {
		q.Logger().WithField("inside-vm", fmt.Sprintf("%t", nested)).Debug("Disable nesting environment checks")
		q.arch.disableNestingChecks()
	}

	return nil
}

func (q *qemu) cpuTopology(podConfig PodConfig) govmmQemu.SMP {
	vcpus := q.config.DefaultVCPUs
	if podConfig.VMConfig.VCPUs > 0 {
		vcpus = uint32(podConfig.VMConfig.VCPUs)
	}

	return q.arch.cpuTopology(vcpus)
}

func (q *qemu) memoryTopology(podConfig PodConfig) (govmmQemu.Memory, error) {
	hostMemKb, err := getHostMemorySizeKb(procMemInfo)
	if err != nil {
		return govmmQemu.Memory{}, fmt.Errorf("Unable to read memory info: %s", err)
	}
	if hostMemKb == 0 {
		return govmmQemu.Memory{}, fmt.Errorf("Error host memory size 0")
	}

	hostMemMb := uint64(float64(hostMemKb / 1024))

	memMb := uint64(q.config.DefaultMemSz)
	if podConfig.VMConfig.Memory > 0 {
		memMb = uint64(podConfig.VMConfig.Memory)
	}

	return q.arch.memoryTopology(memMb, hostMemMb), nil
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

	machine, err := q.arch.machine()
	if err != nil {
		return err
	}

	accelerators := podConfig.HypervisorConfig.MachineAccelerators
	if accelerators != "" {
		if !strings.HasPrefix(accelerators, ",") {
			accelerators = fmt.Sprintf(",%s", accelerators)
		}
		machine.Options += accelerators
	}

	smp := q.cpuTopology(podConfig)

	memory, err := q.memoryTopology(podConfig)
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
		Params: q.kernelParameters(),
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

	devices = q.arch.append9PVolumes(devices, podConfig.Volumes)
	devices = q.arch.appendConsole(devices, q.getPodConsole(podConfig.ID))
	devices = q.arch.appendBridges(devices, q.state.Bridges)

	imagePath, err := q.config.ImageAssetPath()
	if err != nil {
		return err
	}

	devices, err = q.arch.appendImage(devices, imagePath)
	if err != nil {
		return err
	}

	cpuModel := q.arch.cpuModel()

	firmwarePath, err := podConfig.HypervisorConfig.FirmwareAssetPath()
	if err != nil {
		return err
	}

	qemuPath, err := q.qemuPath()
	if err != nil {
		return err
	}

	qemuConfig := govmmQemu.Config{
		Name:        fmt.Sprintf("pod-%s", podConfig.ID),
		UUID:        q.state.UUID,
		Path:        qemuPath,
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
	if q.config.Debug {
		params := q.arch.kernelParameters(q.config.Debug)
		strParams := SerializeParams(params, "=")
		formatted := strings.Join(strParams, " ")

		// The name of this field matches a similar one generated by
		// the runtime and allows users to identify which parameters
		// are set here, which come from the runtime and which are set
		// by the user.
		q.Logger().WithField("default-kernel-parameters", formatted).Debug()
	}

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
		q.qemuConfig.Devices = q.arch.append9PVolume(q.qemuConfig.Devices, v)
	case Socket:
		q.qemuConfig.Devices = q.arch.appendSocket(q.qemuConfig.Devices, v)
	case Endpoint:
		q.qemuConfig.Devices = q.arch.appendNetwork(q.qemuConfig.Devices, v)
	case Drive:
		q.qemuConfig.Devices = q.arch.appendBlockDevice(q.qemuConfig.Devices, v)

	//vhostUserDevice is an interface, hence the pointer for Net, SCSI and Blk:
	case VhostUserNetDevice:
		q.qemuConfig.Devices = q.arch.appendVhostUserDevice(q.qemuConfig.Devices, &v)
	case VhostUserSCSIDevice:
		q.qemuConfig.Devices = q.arch.appendVhostUserDevice(q.qemuConfig.Devices, &v)
	case VhostUserBlkDevice:
		q.qemuConfig.Devices = q.arch.appendVhostUserDevice(q.qemuConfig.Devices, &v)
	case VFIODevice:
		q.qemuConfig.Devices = q.arch.appendVFIODevice(q.qemuConfig.Devices, v)
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
