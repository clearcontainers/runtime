/*
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
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"context"

	"github.com/01org/ciao/qemu"
	"github.com/golang/glog"
)

const (
	qemuEfiFw = "/usr/share/qemu/OVMF.fd"
	seedImage = "seed.iso"
	vcTries   = 10
)

type qmpGlogLogger struct{}

func (l qmpGlogLogger) V(level int32) bool {
	return bool(glog.V(glog.Level(level)))
}

func (l qmpGlogLogger) Infof(format string, v ...interface{}) {
	glog.InfoDepth(2, fmt.Sprintf(format, v...))
}

func (l qmpGlogLogger) Warningf(format string, v ...interface{}) {
	glog.WarningDepth(2, fmt.Sprintf(format, v...))
}

func (l qmpGlogLogger) Errorf(format string, v ...interface{}) {
	glog.ErrorDepth(2, fmt.Sprintf(format, v...))
}

var virtualSizeRegexp *regexp.Regexp
var pssRegexp *regexp.Regexp

func init() {
	virtualSizeRegexp = regexp.MustCompile(`virtual size:.*\(([0-9]+) bytes\)`)
	pssRegexp = regexp.MustCompile(`^Pss:\s*([0-9]+)`)
}

type qemuV struct {
	cfg            *vmConfig
	instanceDir    string
	vcPort         int
	pid            int
	prevCPUTime    int64
	prevSampleTime time.Time
	isoPath        string
}

func (q *qemuV) init(cfg *vmConfig, instanceDir string) {
	q.cfg = cfg
	q.instanceDir = instanceDir
	q.isoPath = path.Join(instanceDir, seedImage)
}

func extractImageInfo(r io.Reader) int {
	imageSizeMiB := -1
	scanner := bufio.NewScanner(r)
	for scanner.Scan() && imageSizeMiB == -1 {
		line := scanner.Text()
		matches := virtualSizeRegexp.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		if len(matches) < 2 {
			glog.Warningf("Unable to find image size from: %s",
				line)
			break
		}

		sizeInBytes, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			glog.Warningf("Unable to parse image size from: %s",
				matches[1])
			break
		}

		size := sizeInBytes / (1024 * 1024)
		if size > int64((^uint(0))>>1) {
			glog.Warningf("Unexpectedly large disk size found: %d MiB",
				size)
			break
		}

		imageSizeMiB = int(size)
		if int64(imageSizeMiB)*1024*1024 < sizeInBytes {
			imageSizeMiB++
		}
	}

	return imageSizeMiB
}

func createCloudInitISO(instanceDir, isoPath string, cfg *vmConfig, userData, metaData []byte) error {
	if len(metaData) == 0 {
		defaultMeta := fmt.Sprintf("{\n  \"uuid\": %q,\n  \"hostname\": %[1]q\n}\n", cfg.Instance)
		metaData = []byte(defaultMeta)
	}

	if err := qemu.CreateCloudInitISO(context.TODO(), instanceDir, isoPath,
		userData, metaData); err != nil {
		glog.Errorf("Unable to create cloudinit iso image %v", err)
		return err
	}

	glog.Infof("ISO image %s created", isoPath)

	return nil
}

func (q *qemuV) ensureBackingImage() error {
	if !q.cfg.haveBootableVolume() {
		return fmt.Errorf("No bootable volumes specified in START payload")
	}

	return nil
}

func (q *qemuV) createImage(bridge string, userData, metaData []byte) error {
	err := createCloudInitISO(q.instanceDir, q.isoPath, q.cfg, userData, metaData)
	if err != nil {
		glog.Errorf("Unable to create iso image %v", err)
		return err
	}

	return nil
}

func (q *qemuV) deleteImage() error {
	return nil
}

func cleanupFds(fds []*os.File, numFds int) {

	maxFds := len(fds)

	if numFds < maxFds {
		maxFds = numFds
	}

	for i := 0; i < maxFds; i++ {
		_ = fds[i].Close()
	}
}

func computeMacvtapParam(vnicName string, mac string, queues int) ([]string, []*os.File, error) {

	fds := make([]*os.File, queues)
	params := make([]string, 0, 8)

	ifIndexPath := path.Join("/sys/class/net", vnicName, "ifindex")
	fip, err := os.Open(ifIndexPath)
	if err != nil {
		glog.Errorf("Failed to determine tap ifname: %s", err)
		return nil, nil, err
	}
	defer func() { _ = fip.Close() }()

	scan := bufio.NewScanner(fip)
	if !scan.Scan() {
		glog.Error("Unable to read tap index")
		return nil, nil, fmt.Errorf("Unable to read tap index")
	}

	i, err := strconv.Atoi(scan.Text())
	if err != nil {
		glog.Errorf("Failed to determine tap ifname: %s", err)
		return nil, nil, err
	}

	//mq support
	var fdParam bytes.Buffer
	fdSeperator := ""
	for q := 0; q < queues; q++ {

		tapDev := fmt.Sprintf("/dev/tap%d", i)

		f, err := os.OpenFile(tapDev, os.O_RDWR, 0666)
		if err != nil {
			glog.Errorf("Failed to open tap device %s: %s", tapDev, err)
			cleanupFds(fds, q)
			return nil, nil, err
		}
		fds[q] = f
		/*
		   3, what do you mean 3.  Well, it turns out that files passed to child
		   processes via cmd.ExtraFiles have different fds in the child and the
		   parent.  In the child the fds are determined by the file's position
		   in the ExtraFiles array + 3.
		*/

		// bytes.WriteString does not return an error
		_, _ = fdParam.WriteString(fmt.Sprintf("%s%d", fdSeperator, q+3))
		fdSeperator = ":"
	}

	netdev := fmt.Sprintf("type=tap,fds=%s,id=%s,vhost=on", fdParam.String(), vnicName)
	device := fmt.Sprintf("virtio-net-pci,netdev=%s,mq=on,vectors=%d,mac=%s", vnicName, 32, mac)
	params = append(params, "-netdev", netdev)
	params = append(params, "-device", device)
	return params, fds, nil
}

func computeTapParam(vnicName string, mac string) ([]string, error) {
	params := make([]string, 0, 8)
	netdev := fmt.Sprintf("type=tap,ifname=%s,script=no,downscript=no,id=%s,vhost=on", vnicName, vnicName)
	device := fmt.Sprintf("driver=virtio-net-pci,netdev=%s,mac=%s", vnicName, mac)

	params = append(params, "-netdev", netdev)
	params = append(params, "-device", device)
	return params, nil
}

func launchQemuWithNC(params []string, fds []*os.File, ipAddress string) (int, error) {
	var err error

	tries := 0
	params = append(params, "-display", "none", "-vga", "none")
	params = append(params, "-device", "isa-serial,chardev=gnc0", "-chardev", "")
	port := 0
	for ; tries < vcTries; tries++ {
		port = uiPortGrabber.grabPort()
		if port == 0 {
			break
		}
		ncString := "socket,port=%d,host=%s,server,id=gnc0,server,nowait"
		params[len(params)-1] = fmt.Sprintf(ncString, port, ipAddress)
		var errStr string

		errStr, err = qemu.LaunchCustomQemu(context.Background(), "", params, fds, qmpGlogLogger{})
		if err == nil {
			glog.Info("============================================")
			glog.Infof("Connect to vm with netcat %s %d", ipAddress, port)
			glog.Info("============================================")
			break
		}

		lowErr := strings.ToLower(errStr)
		if !strings.Contains(lowErr, "socket") {
			uiPortGrabber.releasePort(port)
			break
		}
	}

	if port == 0 || (err != nil && tries == vcTries) {
		glog.Warning("Failed to launch qemu due to chardev error.  Relaunching without virtual console")
		_, err = qemu.LaunchCustomQemu(context.Background(), "", params[:len(params)-4], fds, qmpGlogLogger{})
	}

	return port, err
}

func launchQemuWithSpice(params []string, fds []*os.File, ipAddress string) (int, error) {
	var err error

	tries := 0
	params = append(params, "-spice", "")
	port := 0
	for ; tries < vcTries; tries++ {
		port = uiPortGrabber.grabPort()
		if port == 0 {
			break
		}
		params[len(params)-1] = fmt.Sprintf("port=%d,addr=%s,disable-ticketing", port, ipAddress)
		var errStr string
		errStr, err = qemu.LaunchCustomQemu(context.Background(), "", params, fds, qmpGlogLogger{})
		if err == nil {
			glog.Info("============================================")
			glog.Infof("Connect to vm with spicec -h %s -p %d", ipAddress, port)
			glog.Info("============================================")
			break
		}

		// Not great I know, but it's the only way to figure out if spice is at fault
		lowErr := strings.ToLower(errStr)
		if !strings.Contains(lowErr, "spice") {
			uiPortGrabber.releasePort(port)
			break
		}
	}

	if port == 0 || (err != nil && tries == vcTries) {
		glog.Warning("Failed to launch qemu due to spice error.  Relaunching without virtual console")
		params = append(params[:len(params)-2], "-display", "none", "-vga", "none")
		_, err = qemu.LaunchCustomQemu(context.Background(), "", params, fds, qmpGlogLogger{})
	}

	return port, err
}

func generateQEMULaunchParams(cfg *vmConfig, isoPath, instanceDir string,
	networkParams []string, cephID string) []string {
	params := make([]string, 0, 32)

	addr := 3
	if launchWithUI.String() == "spice" {
		addr = 4
	}

	// I know this is nasty but we have to specify a bus and address otherwise qemu
	// hangs on startup.  I can't find a way to get qemu to pre-allocate the address.
	// It will do this when using the legacy method of adding volumes but we can't do
	// this if we want to be able to live detach these volumes.  The first drive qemu
	// adds, i.e., the rootfs  is assigned a slot of 3 without spice and 4 with.

	for _, v := range cfg.Volumes {
		blockdevID := fmt.Sprintf("drive_%s", v.UUID)
		volDriveStr := fmt.Sprintf("file=rbd:rbd/%s:id=%s,if=none,id=%s,format=raw",
			v.UUID, cephID, blockdevID)
		params = append(params, "-drive", volDriveStr)
		volDeviceStr :=
			fmt.Sprintf("virtio-blk-pci,scsi=off,bus=pci.0,addr=0x%x,id=device_%s,drive=%s",
				addr, v.UUID, blockdevID)
		params = append(params, "-device", volDeviceStr)
		addr++
	}

	isoParam := fmt.Sprintf("file=%s,if=virtio,media=cdrom", isoPath)
	params = append(params, "-drive", isoParam)

	params = append(params, networkParams...)

	useKvm := true

	switch qemuVirtualisation {
	case "software":
		useKvm = false
	case "auto":
		_, err := os.Stat("/dev/kvm")
		if err != nil {
			useKvm = false
		}
	}

	if useKvm {
		params = append(params, "-enable-kvm")
		params = append(params, "-cpu", "host")
	} else {
		glog.Warning("Running qemu without kvm support")
	}

	params = append(params, "-daemonize")

	qmpSocket := path.Join(instanceDir, "socket")
	qmpParam := fmt.Sprintf("unix:%s,server,nowait", qmpSocket)
	params = append(params, "-qmp", qmpParam)

	if cfg.Mem > 0 {
		memoryParam := fmt.Sprintf("%d", cfg.Mem)
		params = append(params, "-m", memoryParam)
	}
	if cfg.Cpus > 0 {
		cpusParam := fmt.Sprintf("cpus=%d", cfg.Cpus)
		params = append(params, "-smp", cpusParam)
	}

	if !cfg.Legacy {
		params = append(params, "-bios", qemuEfiFw)
	}
	return params
}

func (q *qemuV) startVM(vnicName, ipAddress, cephID string) error {

	var fds []*os.File

	glog.Info("Launching qemu")

	networkParams := make([]string, 0, 32)

	if vnicName != "" {
		if q.cfg.NetworkNode {
			var err error
			var macvtapParam []string
			//TODO: @mcastelino get from scheduler/controller
			numQueues := 4
			macvtapParam, fds, err = computeMacvtapParam(vnicName, q.cfg.VnicMAC, numQueues)
			if err != nil {
				return err
			}
			defer cleanupFds(fds, len(fds))
			networkParams = append(networkParams, macvtapParam...)
		} else {
			tapParam, err := computeTapParam(vnicName, q.cfg.VnicMAC)
			if err != nil {
				return err
			}
			networkParams = append(networkParams, tapParam...)
		}
	} else {
		networkParams = append(networkParams, "-net", "nic,model=virtio")
		networkParams = append(networkParams, "-net", "user")
	}

	params := generateQEMULaunchParams(q.cfg, q.isoPath, q.instanceDir, networkParams, cephID)

	var err error

	if !launchWithUI.Enabled() {
		params = append(params, "-display", "none", "-vga", "none")
		_, err = qemu.LaunchCustomQemu(context.Background(), "", params, fds, qmpGlogLogger{})
	} else if launchWithUI.String() == "spice" {
		var port int
		port, err = launchQemuWithSpice(params, fds, ipAddress)
		if err == nil {
			q.vcPort = port
		}
	} else {
		var port int
		port, err = launchQemuWithNC(params, fds, ipAddress)
		if err == nil {
			q.vcPort = port
		}
	}

	if err != nil {
		return err
	}

	glog.Info("Launched VM")

	return nil
}

func (q *qemuV) lostVM() {
	if launchWithUI.Enabled() {
		glog.Infof("Releasing VC Port %d", q.vcPort)
		uiPortGrabber.releasePort(q.vcPort)
		q.vcPort = 0
	}
	q.pid = 0
	q.prevCPUTime = -1
}

func qmpAttach(cmd virtualizerAttachCmd, q *qemu.QMP) {
	glog.Info("Attach command received")
	blockdevID := fmt.Sprintf("drive_%s", cmd.volumeUUID)
	err := q.ExecuteBlockdevAdd(context.Background(), cmd.device, blockdevID)
	if err != nil {
		glog.Errorf("Failed to execute blockdev-add: %v", err)
	} else {
		devID := fmt.Sprintf("device_%s", cmd.volumeUUID)
		err = q.ExecuteDeviceAdd(context.Background(), blockdevID,
			devID, "virtio-blk-pci", "")
		if err != nil {
			glog.Errorf("Failed to execute device_add: %v", err)
		}
	}
	cmd.responseCh <- err
}

func qmpDetach(cmd virtualizerDetachCmd, q *qemu.QMP) {
	glog.Info("Detach command received")
	devID := fmt.Sprintf("device_%s", cmd.volumeUUID)
	err := q.ExecuteDeviceDel(context.Background(), devID)
	if err != nil {
		glog.Errorf("Failed to execute device_del: %v", err)
	} else {
		blockdevID := fmt.Sprintf("drive_%s", cmd.volumeUUID)
		err = q.ExecuteXBlockdevDel(context.Background(), blockdevID)
		if err != nil {
			glog.Errorf("Failed to execute x-blockdev-del: %v", err)
		}
	}
	cmd.responseCh <- err
}

func qmpConnect(qmpChannel chan interface{}, instance, instanceDir string, closedCh chan struct{},
	connectedCh chan struct{}, wg *sync.WaitGroup, boot bool) {

	var q *qemu.QMP
	defer func() {
		if q != nil {
			q.Shutdown()
		}
		glog.Infof("Monitor function for %s exitting", instance)
		wg.Done()
	}()

	socket := path.Join(instanceDir, "socket")
	cfg := qemu.QMPConfig{Logger: qmpGlogLogger{}}
	q, ver, err := qemu.QMPStart(context.Background(), socket, cfg, closedCh)
	if err != nil {
		glog.Warningf("Failed to connect to QEMU instance %s: %v", instance, err)
		return
	}

	glog.Infof("Connected to %s.", instance)
	glog.Infof("QMP version %d.%d.%d", ver.Major, ver.Minor, ver.Micro)
	glog.Infof("QMP capabilities %s", ver.Capabilities)

	err = q.ExecuteQMPCapabilities(context.Background())
	if err != nil {
		glog.Errorf("Unable to send qmp_capabilities command: %v", err)
		return
	}

	close(connectedCh)

DONE:
	for {
		cmd, ok := <-qmpChannel
		if !ok {
			break DONE
		}
		switch cmd := cmd.(type) {
		case virtualizerStopCmd:
			ctx, cancelFN := context.WithTimeout(context.Background(), time.Second*10)
			err = q.ExecuteSystemPowerdown(ctx)
			cancelFN()
			if err != nil {
				glog.Warningf("Failed to power down cleanly: %v", err)
				err = q.ExecuteQuit(context.Background())
				if err != nil {
					glog.Warningf("Failed to execute quit instance: %v", err)
				}
			}
		case virtualizerAttachCmd:
			qmpAttach(cmd, q)
		case virtualizerDetachCmd:
			qmpDetach(cmd, q)
		}
	}
}

/* closedCh is closed by the monitor go routine when it loses connection to the domain socket, basically,
   indicating that the VM instance has shut down.  The instance go routine is expected to close the
   qmpChannel to force the monitor go routine to exit.

   connectedCh is closed when we successfully connect to the domain socket, inidcating that the
   VM instance is running.
*/

func (q *qemuV) monitorVM(closedCh chan struct{}, connectedCh chan struct{},
	wg *sync.WaitGroup, boot bool) chan interface{} {
	qmpChannel := make(chan interface{})
	wg.Add(1)
	go qmpConnect(qmpChannel, q.cfg.Instance, q.instanceDir, closedCh, connectedCh, wg, boot)
	return qmpChannel
}

func (q *qemuV) stats() (disk, memory, cpu int) {
	disk = 0
	memory = -1
	cpu = -1

	if q.pid == 0 {
		return
	}

	memory = computeProcessMemUsage(q.pid)
	if q.cfg == nil {
		return
	}

	cpuTime := computeProcessCPUTime(q.pid)
	now := time.Now()
	if q.prevCPUTime != -1 {
		cpu = int((100 * (cpuTime - q.prevCPUTime) /
			now.Sub(q.prevSampleTime).Nanoseconds()))
		if q.cfg.Cpus > 1 {
			cpu /= q.cfg.Cpus
		}
		// if glog.V(1) {
		//     glog.Infof("cpu %d%%\n", cpu)
		// }
	}
	q.prevCPUTime = cpuTime
	q.prevSampleTime = now

	return
}

func (q *qemuV) connected() {
	qmpSocket := path.Join(q.instanceDir, "socket")
	var buf bytes.Buffer
	cmd := exec.Command("fuser", qmpSocket)
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		glog.Errorf("Failed to run fuser: %v", err)
		return
	}

	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		pidString := strings.TrimSpace(scanner.Text())
		pid, err := strconv.Atoi(pidString)
		if err != nil {
			continue
		}

		if pid != 0 && pid != os.Getpid() {
			glog.Infof("PID of qemu for instance %s is %d", q.instanceDir, pid)
			q.pid = pid
			break
		}
	}

	if q.pid == 0 {
		glog.Errorf("Unable to determine pid for %s", q.instanceDir)
	}
	q.prevCPUTime = -1
}
