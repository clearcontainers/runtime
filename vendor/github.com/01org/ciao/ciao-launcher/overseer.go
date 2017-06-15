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
	"container/list"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/01org/ciao/deviceinfo"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
)

type ovsAddResult struct {
	cmdCh  chan<- interface{}
	canAdd bool
}

type ovsAddCmd struct {
	instance string
	cfg      *vmConfig
	targetCh chan<- ovsAddResult
}

type ovsGetResult struct {
	cmdCh   chan<- interface{}
	running ovsRunningState
}

type ovsGetCmd struct {
	instance string
	targetCh chan<- ovsGetResult
}

type ovsRemoveCmd struct {
	instance string
	errCh    chan<- error
}

type ovsStateChange struct {
	instance string
	state    ovsRunningState
}

type ovsStatsUpdateCmd struct {
	instance      string
	memoryUsageMB int
	diskUsageMB   int
	CPUUsage      int
	volumes       []string
}

type ovsTraceFrame struct {
	frame *ssntp.Frame
}

type ovsStatusCmd struct{}
type ovsStatsStatusCmd struct{}

type ovsRunningState int

type deviceInfo interface {
	GetLoadAvg() int
	GetFSInfo(path string) (total, available int)
	GetOnlineCPUs() int
	GetMemoryInfo() (total, available int)
}

type realDeviceInfo struct{}

func (realDeviceInfo) GetLoadAvg() int {
	return deviceinfo.GetLoadAvg()
}

func (realDeviceInfo) GetFSInfo(path string) (total, available int) {
	return deviceinfo.GetFSInfo(path)
}

func (realDeviceInfo) GetOnlineCPUs() int {
	return deviceinfo.GetOnlineCPUs()
}

func (realDeviceInfo) GetMemoryInfo() (total, available int) {
	return deviceinfo.GetMemoryInfo()
}

const (
	ovsPending ovsRunningState = iota
	ovsRunning
	ovsStopped
)

const (
	diskSpaceHWM = 80 * 1000
	memHWM       = 1 * 1000
	diskSpaceLWM = 40 * 1000
	memLWM       = 512
)

type ovsInstanceState struct {
	cmdCh          chan<- interface{}
	running        ovsRunningState
	memoryUsageMB  int
	diskUsageMB    int
	CPUUsage       int
	maxDiskUsageMB int
	maxVCPUs       int
	maxMemoryMB    int
	sshIP          string
	sshPort        int
	volumes        []string
}

type overseer struct {
	instancesDir       string
	instances          map[string]*ovsInstanceState
	ovsCh              chan interface{}
	ovsInstanceCh      chan interface{}
	childDoneCh        chan struct{}
	parentWg           *sync.WaitGroup
	childWg            *sync.WaitGroup
	ac                 *agentClient
	vcpusAllocated     int
	diskSpaceAllocated int
	memoryAllocated    int
	diskSpaceAvailable int
	memoryAvailable    int
	traceFrames        *list.List
	statsInterval      time.Duration
	di                 deviceInfo
}

type cnStats struct {
	totalMemMB      int
	availableMemMB  int
	totalDiskMB     int
	availableDiskMB int
	load            int
	cpusOnline      int
}

func (ovs *overseer) roomAvailable(cfg *vmConfig) bool {

	if len(ovs.instances) >= maxInstances {
		glog.Warningf("We're FULL.  Too many instances %d", len(ovs.instances))
		return false
	}

	diskSpaceAvailable := ovs.diskSpaceAvailable - cfg.Disk
	memoryAvailable := ovs.memoryAvailable - cfg.Mem

	glog.Infof("disk Avail %d MemAvail %d", diskSpaceAvailable, memoryAvailable)

	if diskSpaceAvailable < diskSpaceLWM {
		if diskLimit == true {
			return false
		}
	}

	if memoryAvailable < memLWM {
		if memLimit == true {
			return false
		}
	}

	return true
}

func (ovs *overseer) updateAvailableResources(cns *cnStats) {
	diskSpaceConsumed := 0
	memConsumed := 0
	for _, target := range ovs.instances {
		if target.diskUsageMB != -1 {
			diskSpaceConsumed += target.diskUsageMB
		}

		if target.memoryUsageMB != -1 {
			if target.memoryUsageMB < target.maxMemoryMB {
				memConsumed += target.memoryUsageMB
			} else {
				memConsumed += target.maxMemoryMB
			}
		}
	}

	ovs.diskSpaceAvailable = (cns.availableDiskMB + diskSpaceConsumed) -
		ovs.diskSpaceAllocated

	ovs.memoryAvailable = (cns.availableMemMB + memConsumed) -
		ovs.memoryAllocated

	if glog.V(1) {
		glog.Infof("Memory Available: %d Disk space Available %d",
			ovs.memoryAvailable, ovs.diskSpaceAvailable)
	}
}

func (ovs *overseer) computeStatus() ssntp.Status {

	if len(ovs.instances) >= maxInstances {
		return ssntp.FULL
	}

	if ovs.diskSpaceAvailable < diskSpaceHWM {
		if diskLimit == true {
			return ssntp.FULL
		}
	}

	if ovs.memoryAvailable < memHWM {
		if memLimit == true {
			return ssntp.FULL
		}
	}

	return ssntp.READY
}

func (ovs *overseer) sendReadyStatusCommand(cns *cnStats) {
	var s payloads.Ready

	s.Init()

	s.NodeUUID = ovs.ac.conn.UUID()
	s.MemTotalMB, s.MemAvailableMB = cns.totalMemMB, cns.availableMemMB
	s.Load = cns.load
	s.CpusOnline = cns.cpusOnline
	s.DiskTotalMB, s.DiskAvailableMB = cns.totalDiskMB, cns.availableDiskMB
	s.Networks = make([]payloads.NetworkStat, len(nicInfo))
	for i, nic := range nicInfo {
		s.Networks[i] = *nic
	}

	payload, err := yaml.Marshal(&s)
	if err != nil {
		glog.Errorf("Unable to Marshall status payload %v", err)
		return
	}

	_, err = ovs.ac.conn.SendStatus(ssntp.READY, payload)
	if err != nil {
		glog.Errorf("Failed to send READY status command %v", err)
	}
}

func (ovs *overseer) sendStatusCommand(cns *cnStats, status ssntp.Status) {
	switch status {
	case ssntp.READY:
		ovs.sendReadyStatusCommand(cns)
	case ssntp.FULL:
		fallthrough
	case ssntp.OFFLINE:
		_, err := ovs.ac.conn.SendStatus(status, nil)
		if err != nil {
			glog.Errorf("Failed to send %s status command %v", status, err)
		}
	default:
		glog.Errorf("Unsupported status command: %s", status)
	}
}

func (ovs *overseer) sendStats(cns *cnStats, status ssntp.Status) {
	var s payloads.Stat

	s.Init()

	s.NodeUUID = ovs.ac.conn.UUID()
	s.Status = status.String()
	s.MemTotalMB, s.MemAvailableMB = cns.totalMemMB, cns.availableMemMB
	s.Load = cns.load
	s.CpusOnline = cns.cpusOnline
	s.DiskTotalMB, s.DiskAvailableMB = cns.totalDiskMB, cns.availableDiskMB
	s.NodeHostName = hostname // global from network.go
	s.Networks = make([]payloads.NetworkStat, len(nicInfo))
	for i, nic := range nicInfo {
		s.Networks[i] = *nic
	}
	s.Instances = make([]payloads.InstanceStat, len(ovs.instances))
	i := 0
	for uuid, state := range ovs.instances {
		s.Instances[i].InstanceUUID = uuid
		if state.running == ovsRunning {
			s.Instances[i].State = payloads.Running
		} else if state.running == ovsStopped {
			s.Instances[i].State = payloads.Exited
		} else {
			s.Instances[i].State = payloads.Pending
		}
		s.Instances[i].MemoryUsageMB = state.memoryUsageMB
		s.Instances[i].DiskUsageMB = state.diskUsageMB
		s.Instances[i].CPUUsage = state.CPUUsage
		s.Instances[i].SSHIP = state.sshIP
		s.Instances[i].SSHPort = state.sshPort
		s.Instances[i].Volumes = state.volumes
		i++
	}

	payload, err := yaml.Marshal(&s)
	if err != nil {
		glog.Errorf("Unable to Marshall STATS %v", err)
		return
	}

	_, err = ovs.ac.conn.SendCommand(ssntp.STATS, payload)
	if err != nil {
		glog.Errorf("Failed to send stats command %v", err)
		return
	}
}

func (ovs *overseer) sendTraceReport() {
	var s payloads.Trace

	if ovs.traceFrames.Len() == 0 {
		return
	}

	for e := ovs.traceFrames.Front(); e != nil; e = e.Next() {
		f := e.Value.(*ssntp.Frame)
		frameTrace, err := f.DumpTrace()
		if err != nil {
			glog.Errorf("Unable to dump traced frame %v", err)
			continue
		}

		s.Frames = append(s.Frames, *frameTrace)
	}

	ovs.traceFrames = list.New()

	payload, err := yaml.Marshal(&s)
	if err != nil {
		glog.Errorf("Unable to Marshall TraceReport %v", err)
		return
	}

	_, err = ovs.ac.conn.SendEvent(ssntp.TraceReport, payload)
	if err != nil {
		glog.Errorf("Failed to send TraceReport event %v", err)
		return
	}
}

func getStats(instancesDir string) *cnStats {
	var s cnStats

	s.totalMemMB, s.availableMemMB = deviceinfo.GetMemoryInfo()
	s.load = deviceinfo.GetLoadAvg()
	s.cpusOnline = deviceinfo.GetOnlineCPUs()
	s.totalDiskMB, s.availableDiskMB = deviceinfo.GetFSInfo(instancesDir)

	return &s
}

func (ovs *overseer) processGetCommand(cmd *ovsGetCmd) {
	glog.Infof("Overseer: looking for instance %s", cmd.instance)
	var insState ovsGetResult
	target := ovs.instances[cmd.instance]
	if target != nil {
		insState.cmdCh = target.cmdCh
		insState.running = target.running
	}
	cmd.targetCh <- insState
}

func (ovs *overseer) processAddCommand(cmd *ovsAddCmd) {
	glog.Infof("Overseer: adding %s", cmd.instance)
	var targetCh chan<- interface{}
	target := ovs.instances[cmd.instance]
	canAdd := true
	cfg := cmd.cfg
	if target != nil {
		targetCh = target.cmdCh
	} else if ovs.roomAvailable(cfg) {
		ovs.vcpusAllocated += cfg.Cpus
		ovs.diskSpaceAllocated += cfg.Disk
		ovs.memoryAllocated += cfg.Mem
		targetCh = startInstance(cmd.instance, cfg, ovs.childWg, ovs.childDoneCh,
			ovs.ac, ovs.ovsInstanceCh)
		ovs.instances[cmd.instance] = &ovsInstanceState{
			cmdCh:          targetCh,
			running:        ovsPending,
			diskUsageMB:    -1,
			CPUUsage:       -1,
			memoryUsageMB:  -1,
			maxDiskUsageMB: cfg.Disk,
			maxVCPUs:       cfg.Cpus,
			maxMemoryMB:    cfg.Mem,
			sshIP:          cfg.ConcIP,
			sshPort:        cfg.SSHPort,
		}
	} else {
		canAdd = false
	}
	cmd.targetCh <- ovsAddResult{targetCh, canAdd}
}

func (ovs *overseer) processRemoveCommand(cmd *ovsRemoveCmd) {
	glog.Infof("Overseer: removing %s", cmd.instance)
	target := ovs.instances[cmd.instance]
	if target == nil {
		cmd.errCh <- fmt.Errorf("Instance does not exist")
		return
	}

	ovs.diskSpaceAllocated -= target.maxDiskUsageMB
	if ovs.diskSpaceAllocated < 0 {
		ovs.diskSpaceAllocated = 0
	}

	ovs.vcpusAllocated -= target.maxVCPUs
	if ovs.vcpusAllocated < 0 {
		ovs.vcpusAllocated = 0
	}

	ovs.memoryAllocated -= target.maxMemoryMB
	if ovs.memoryAllocated < 0 {
		ovs.memoryAllocated = 0
	}

	delete(ovs.instances, cmd.instance)
	cmd.errCh <- nil
}

func (ovs *overseer) processStatusCommand(cmd *ovsStatusCmd) {
	glog.Info("Overseer: Received Status Command")
	if !ovs.ac.conn.isConnected() {
		return
	}
	cns := getStats(ovs.instancesDir)
	ovs.updateAvailableResources(cns)
	ovs.sendStatusCommand(cns, ovs.computeStatus())
}

func (ovs *overseer) processStatsStatusCommand(cmd *ovsStatsStatusCmd) {
	glog.Info("Overseer: Received StatsStatus Command")
	if !ovs.ac.conn.isConnected() {
		return
	}
	cns := getStats(ovs.instancesDir)
	ovs.updateAvailableResources(cns)
	status := ovs.computeStatus()
	ovs.sendStatusCommand(cns, status)
	ovs.sendStats(cns, status)
}

func (ovs *overseer) processStateChangeCommand(cmd *ovsStateChange) {
	glog.Infof("Overseer: Received State Change %v", *cmd)
	target := ovs.instances[cmd.instance]
	if target != nil {
		target.running = cmd.state
	}
}

func (ovs *overseer) processStatusUpdateCommand(cmd *ovsStatsUpdateCmd) {
	if glog.V(1) {
		glog.Infof("STATS Update for %s: Mem %d Disk %d Cpu %d",
			cmd.instance, cmd.memoryUsageMB,
			cmd.diskUsageMB, cmd.CPUUsage)
	}
	target := ovs.instances[cmd.instance]
	if target != nil {
		target.memoryUsageMB = cmd.memoryUsageMB
		target.diskUsageMB = cmd.diskUsageMB
		target.CPUUsage = cmd.CPUUsage
		target.volumes = cmd.volumes
	}
}

func (ovs *overseer) processTraceFrameCommand(cmd *ovsTraceFrame) {
	cmd.frame.SetEndStamp()
	ovs.traceFrames.PushBack(cmd.frame)
}

func (ovs *overseer) processCommand(cmd interface{}) {
	switch cmd := cmd.(type) {
	case *ovsGetCmd:
		ovs.processGetCommand(cmd)
	case *ovsAddCmd:
		ovs.processAddCommand(cmd)
	case *ovsRemoveCmd:
		ovs.processRemoveCommand(cmd)
	case *ovsStatusCmd:
		ovs.processStatusCommand(cmd)
	case *ovsStatsStatusCmd:
		ovs.processStatsStatusCommand(cmd)
	case *ovsStateChange:
		ovs.processStateChangeCommand(cmd)
	case *ovsStatsUpdateCmd:
		ovs.processStatusUpdateCommand(cmd)
	case *ovsTraceFrame:
		ovs.processTraceFrameCommand(cmd)
	default:
		panic("Unknown Overseer Command")
	}
}

func (ovs *overseer) runOverseer() {

	statsTimer := time.After(ovs.statsInterval)
DONE:
	for {
		select {
		case cmd, ok := <-ovs.ovsCh:
			if !ok {
				break DONE
			}
			ovs.processCommand(cmd)
		case cmd := <-ovs.ovsInstanceCh:
			ovs.processCommand(cmd)
		case <-statsTimer:
			if !ovs.ac.conn.isConnected() {
				statsTimer = time.After(ovs.statsInterval)
				continue
			}

			cns := getStats(ovs.instancesDir)
			ovs.updateAvailableResources(cns)
			status := ovs.computeStatus()
			ovs.sendStatusCommand(cns, status)
			ovs.sendStats(cns, status)
			ovs.sendTraceReport()
			statsTimer = time.After(ovs.statsInterval)
			if glog.V(1) {
				glog.Infof("Consumed: Disk %d Mem %d CPUs %d",
					ovs.diskSpaceAllocated, ovs.memoryAllocated, ovs.vcpusAllocated)
			}
		}
	}

	close(ovs.childDoneCh)

DRAIN:

	// Here we have the problem that we have multiple go routines writing to the
	// same channel.   We cannot therefore use the closure of this channel as
	// a signal that all writes are done.  Instead we need to use waitgroups.
	// But here's the catch.  We need to keep reading on the ovsInstanceCh channel
	// until the childWg indicates that all instances have exitted, otherwise some
	// of them might block trying to write to ovsInstanceCh.

	for {
		select {
		case <-ovs.ovsInstanceCh:
		case <-func() chan struct{} {
			ch := make(chan struct{})
			go func() {
				ovs.childWg.Wait()
				close(ch)
			}()
			return ch
		}():
			break DRAIN
		}
	}

	glog.Info("All instance go routines have exitted")
	ovs.parentWg.Done()

	glog.Info("Overseer exitting")
}

func startOverseerFull(instancesDir string, wg *sync.WaitGroup, ac *agentClient,
	statsInterval time.Duration, di deviceInfo) chan<- interface{} {

	instances := make(map[string]*ovsInstanceState)
	ovsCh := make(chan interface{})
	ovsInstanceCh := make(chan interface{})
	toMonitor := make([]chan<- interface{}, 0, 1024)
	childDoneCh := make(chan struct{})
	childWg := new(sync.WaitGroup)

	vcpusAllocated := 0
	diskSpaceAllocated := 0
	memoryAllocated := 0

	_ = filepath.Walk(instancesDir, func(path string, info os.FileInfo, err error) error {
		if path == instancesDir {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		glog.Infof("Reconnecting to existing instance %s", path)
		instance := filepath.Base(path)

		// BUG(markus): We should garbage collect corrupt instances

		cfg, err := loadVMConfig(path)
		if err != nil {
			glog.Warning("Unable to load state of running instance %s: %v", instance, err)
			return nil
		}

		vcpusAllocated += cfg.Cpus
		diskSpaceAllocated += cfg.Disk
		memoryAllocated += cfg.Mem

		target := startInstance(instance, cfg, childWg, childDoneCh, ac, ovsInstanceCh)
		instances[instance] = &ovsInstanceState{
			cmdCh:          target,
			running:        ovsPending,
			diskUsageMB:    -1,
			CPUUsage:       -1,
			memoryUsageMB:  -1,
			maxDiskUsageMB: cfg.Disk,
			maxVCPUs:       cfg.Cpus,
			maxMemoryMB:    cfg.Mem,
			sshIP:          cfg.ConcIP,
			sshPort:        cfg.SSHPort,
		}
		toMonitor = append(toMonitor, target)

		return filepath.SkipDir
	})

	ovs := &overseer{
		instancesDir:       instancesDir,
		instances:          instances,
		ovsCh:              ovsCh,
		ovsInstanceCh:      ovsInstanceCh,
		parentWg:           wg,
		childWg:            childWg,
		childDoneCh:        childDoneCh,
		ac:                 ac,
		vcpusAllocated:     vcpusAllocated,
		diskSpaceAllocated: diskSpaceAllocated,
		memoryAllocated:    memoryAllocated,
		traceFrames:        list.New(),
		statsInterval:      statsInterval,
		di:                 di,
	}
	ovs.parentWg.Add(1)
	glog.Info("Starting Overseer")
	glog.Infof("Allocated: Disk %d Mem %d CPUs %d",
		diskSpaceAllocated, memoryAllocated, vcpusAllocated)
	go ovs.runOverseer()
	ovs = nil
	instances = nil

	// I know this looks weird but there is method here.  After we launch the overseer go routine
	// we can no longer access instances from this go routine otherwise we will have a data race.
	// For this reason we make a copy of the instance command channels that can be safely used
	// in this go routine.  The monitor commands cannot be sent from the overseer as it is not
	// allowed to send information to the instance go routines.  Doing so would incur the risk of
	// deadlock.  So we copy.  'A little copying is better than a little dependency', and so forth.

	for _, v := range toMonitor {
		v <- &insMonitorCmd{}
	}

	return ovsCh
}

func startOverseer(wg *sync.WaitGroup, ac *agentClient) chan<- interface{} {
	return startOverseerFull(instancesDir, wg, ac, time.Second*statsPeriod,
		realDeviceInfo{})
}
