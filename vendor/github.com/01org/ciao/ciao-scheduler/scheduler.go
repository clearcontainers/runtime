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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/01org/ciao/clogger/gloginterface"
	"github.com/01org/ciao/osprepare"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

var cert = flag.String("cert", "/etc/pki/ciao/cert-Scheduler-localhost.pem", "Server certificate")
var cacert = flag.String("cacert", "/etc/pki/ciao/CAcert-server-localhost.pem", "CA certificate")
var cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
var heartbeat = flag.Bool("heartbeat", false, "Emit status heartbeat text")
var logDir = "/var/lib/ciao/logs/scheduler"
var configURI = flag.String("configuration-uri", "file:///etc/ciao/configuration.yaml",
	"Cluster configuration URI")

type ssntpSchedulerServer struct {
	// user config overrides ------------------------------------------
	heartbeat  bool
	cpuprofile string

	// ssntp ----------------------------------------------------------
	config *ssntp.Config
	ssntp  ssntp.Server

	// scheduler internal state ---------------------------------------

	// Command & Status Reporting node(s)
	controllerMap   map[string]*controllerStat
	controllerList  []*controllerStat // 1 controllerMaster at front of list
	controllerMutex sync.RWMutex      // Rlock traversing map, Lock modifying map

	// Compute Nodes
	cnMap      map[string]*nodeStat
	cnList     []*nodeStat
	cnMutex    sync.RWMutex // Rlock traversing map, Lock modifying map
	cnMRU      *nodeStat
	cnMRUIndex int
	//cnInactiveMap      map[string]nodeStat

	// Network Nodes
	nnMap      map[string]*nodeStat
	nnList     []*nodeStat
	nnMutex    sync.RWMutex // Rlock traversing map, Lock modifying map
	nnMRU      *nodeStat
	nnMRUIndex int
}

func newSsntpSchedulerServer() *ssntpSchedulerServer {
	return &ssntpSchedulerServer{
		controllerMap: make(map[string]*controllerStat),
		cnMap:         make(map[string]*nodeStat),
		cnMRUIndex:    -1,
		nnMap:         make(map[string]*nodeStat),
		nnMRUIndex:    -1,
	}
}

type nodeStat struct {
	mutex       sync.Mutex
	status      ssntp.Status
	uuid        string
	memTotalMB  int
	memAvailMB  int
	diskTotalMB int
	diskAvailMB int
	load        int
	cpus        int
	isNetNode   bool
	networks    []payloads.NetworkStat
}

type controllerStatus uint8

func (s controllerStatus) String() string {
	switch s {
	case controllerMaster:
		return "MASTER"
	case controllerBackup:
		return "BACKUP"
	}

	return ""
}

const (
	controllerMaster controllerStatus = iota
	controllerBackup
)

type controllerStat struct {
	mutex  sync.Mutex
	status controllerStatus
	uuid   string
}

func (sched *ssntpSchedulerServer) sendNodeConnectionEvent(nodeUUID, controllerUUID string, nodeType payloads.Resource, connected bool) (int, error) {
	/* connect */
	if connected == true {
		payload := payloads.NodeConnected{
			Connected: payloads.NodeConnectedEvent{
				NodeUUID: nodeUUID,
				NodeType: nodeType,
			},
		}

		b, err := yaml.Marshal(&payload)
		if err != nil {
			return 0, err
		}

		return sched.ssntp.SendEvent(controllerUUID, ssntp.NodeConnected, b)
	}

	/* disconnect */
	payload := payloads.NodeDisconnected{
		Disconnected: payloads.NodeConnectedEvent{
			NodeUUID: nodeUUID,
			NodeType: nodeType,
		},
	}

	b, err := yaml.Marshal(&payload)
	if err != nil {
		return 0, err
	}

	return sched.ssntp.SendEvent(controllerUUID, ssntp.NodeDisconnected, b)
}

func (sched *ssntpSchedulerServer) sendNodeConnectedEvents(nodeUUID string, nodeType payloads.Resource) {
	sched.controllerMutex.RLock()
	defer sched.controllerMutex.RUnlock()

	for _, c := range sched.controllerMap {
		sched.sendNodeConnectionEvent(nodeUUID, c.uuid, nodeType, true)
	}
}

func (sched *ssntpSchedulerServer) sendNodeDisconnectedEvents(nodeUUID string, nodeType payloads.Resource) {
	sched.controllerMutex.RLock()
	defer sched.controllerMutex.RUnlock()

	for _, c := range sched.controllerMap {
		sched.sendNodeConnectionEvent(nodeUUID, c.uuid, nodeType, false)
	}
}

// Add state for newly connected Controller
// This function is symmetric with disconnectController().
func connectController(sched *ssntpSchedulerServer, uuid string) {
	sched.controllerMutex.Lock()
	defer sched.controllerMutex.Unlock()

	if sched.controllerMap[uuid] != nil {
		glog.Warningf("Unexpected reconnect from controller %s\n", uuid)
		return
	}

	var controller controllerStat
	controller.uuid = uuid

	// TODO: smarter clustering than "assume master, unless another is master"
	if len(sched.controllerList) == 0 || sched.controllerList[0].status == controllerBackup {
		// master at front of the list
		controller.status = controllerMaster
		sched.controllerList = append([]*controllerStat{&controller}, sched.controllerList...)
	} else { // already have a master
		// backup controllers at the end of the list
		controller.status = controllerBackup
		sched.controllerList = append(sched.controllerList, &controller)
	}

	sched.controllerMap[controller.uuid] = &controller
}

// Undo previous state additions for departed Controller
// This function is symmetric with connectController().
func disconnectController(sched *ssntpSchedulerServer, uuid string) {
	sched.controllerMutex.Lock()
	defer sched.controllerMutex.Unlock()

	controller := sched.controllerMap[uuid]
	if controller == nil {
		glog.Warningf("Unexpected disconnect from controller %s\n", uuid)
		return
	}

	// delete from map, remove from list
	delete(sched.controllerMap, uuid)
	for i, c := range sched.controllerList {
		if c != controller {
			continue
		}

		sched.controllerList = append(sched.controllerList[:i], sched.controllerList[i+1:]...)
	}

	if controller.status == controllerBackup {
		return
	} // else promote a new master

	for i, c := range sched.controllerList {
		c.mutex.Lock()
		if c.status == controllerBackup {
			c.status = controllerMaster
			//TODO: inform the Controller it is master
			c.mutex.Unlock()

			// move to front of list
			front := sched.controllerList[:i]
			back := sched.controllerList[i+1:]
			sched.controllerList = append([]*controllerStat{c}, front...)
			sched.controllerList = append(sched.controllerList, back...)
			break
		}
		c.mutex.Unlock()
	}
}

// Add state for newly connected Compute Node
// This function is symmetric with disconnectComputeNode().
func connectComputeNode(sched *ssntpSchedulerServer, uuid string) {
	sched.cnMutex.Lock()
	defer sched.cnMutex.Unlock()

	if sched.cnMap[uuid] != nil {
		glog.Warningf("Unexpected reconnect from compute node %s\n", uuid)
		return
	}

	var node nodeStat
	node.status = ssntp.CONNECTED
	node.uuid = uuid
	node.isNetNode = false
	sched.cnList = append(sched.cnList, &node)
	sched.cnMap[uuid] = &node

	sched.sendNodeConnectedEvents(uuid, payloads.ComputeNode)
}

// Undo previous state additions for departed Compute Node
// This function is symmetric with connectComputeNode().
func disconnectComputeNode(sched *ssntpSchedulerServer, uuid string) {
	sched.cnMutex.Lock()
	defer sched.cnMutex.Unlock()

	node := sched.cnMap[uuid]
	if node == nil {
		glog.Warningf("Unexpected disconnect from compute node %s\n", uuid)
		return
	}

	//TODO: consider moving to cnInactiveMap?
	delete(sched.cnMap, uuid)

	for i, n := range sched.cnList {
		if n != node {
			continue
		}

		sched.cnList = append(sched.cnList[:i], sched.cnList[i+1:]...)
	}

	if node == sched.cnMRU {
		sched.cnMRU = nil
		sched.cnMRUIndex = -1
	}

	sched.sendNodeDisconnectedEvents(uuid, payloads.ComputeNode)
}

// Add state for newly connected Network Node
// This function is symmetric with disconnectNetworkNode().
func connectNetworkNode(sched *ssntpSchedulerServer, uuid string) {
	sched.nnMutex.Lock()
	defer sched.nnMutex.Unlock()

	if sched.nnMap[uuid] != nil {
		glog.Warningf("Unexpected reconnect from network compute node %s\n", uuid)
		return
	}

	var node nodeStat
	node.status = ssntp.CONNECTED
	node.uuid = uuid
	node.isNetNode = true
	sched.nnList = append(sched.nnList, &node)
	sched.nnMap[uuid] = &node

	sched.sendNodeConnectedEvents(uuid, payloads.NetworkNode)
}

// Undo previous state additions for departed Network Node
// This function is symmetric with connectNetworkNode().
func disconnectNetworkNode(sched *ssntpSchedulerServer, uuid string) {
	sched.nnMutex.Lock()
	defer sched.nnMutex.Unlock()

	node := sched.nnMap[uuid]
	if node == nil {
		glog.Warningf("Unexpected disconnect from network compute node %s\n", uuid)
		return
	}

	//TODO: consider moving to nnInactiveMap?
	delete(sched.nnMap, uuid)

	for i, n := range sched.nnList {
		if n != node {
			continue
		}

		sched.nnList = append(sched.nnList[:i], sched.nnList[i+1:]...)
	}

	if node == sched.nnMRU {
		sched.nnMRU = nil
		sched.nnMRUIndex = -1
	}

	sched.sendNodeDisconnectedEvents(uuid, payloads.NetworkNode)
}
func (sched *ssntpSchedulerServer) ConnectNotify(uuid string, role ssntp.Role) {
	if role.IsController() {
		connectController(sched, uuid)
	}
	if role.IsAgent() {
		connectComputeNode(sched, uuid)
	}
	if role.IsNetAgent() {
		connectNetworkNode(sched, uuid)
	}

	glog.V(2).Infof("Connect (role 0x%x, uuid=%s)\n", role, uuid)
}

func (sched *ssntpSchedulerServer) DisconnectNotify(uuid string, role ssntp.Role) {
	if role.IsController() {
		disconnectController(sched, uuid)
	}
	if role.IsAgent() {
		disconnectComputeNode(sched, uuid)
	}
	if role.IsNetAgent() {
		disconnectNetworkNode(sched, uuid)
	}

	glog.V(2).Infof("Connect (role 0x%x, uuid=%s)\n", role, uuid)
}

func (sched *ssntpSchedulerServer) updateNodeStat(node *nodeStat, status ssntp.Status, frame *ssntp.Frame) {
	payload := frame.Payload

	node.mutex.Lock()
	defer node.mutex.Unlock()

	node.status = status
	switch node.status {
	case ssntp.READY:
		//pull in client's READY status frame transmitted statistics
		var stats payloads.Ready
		err := yaml.Unmarshal(payload, &stats)
		if err != nil {
			glog.Errorf("Bad READY yaml for node %s\n", node.uuid)
			return
		}
		node.memTotalMB = stats.MemTotalMB
		node.memAvailMB = stats.MemAvailableMB
		node.diskTotalMB = stats.DiskTotalMB
		node.diskAvailMB = stats.DiskAvailableMB
		node.load = stats.Load
		node.cpus = stats.CpusOnline
		node.networks = stats.Networks

		//any changes to the payloads.Ready struct should be
		//accompanied by a change here
	}
}

func (sched *ssntpSchedulerServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
	// for now only pay attention to READY status

	role, err := sched.ssntp.ClientRole(uuid)
	if err != nil {
		glog.Errorf("STATUS ignored from disconnected client %s", uuid)
		return
	}

	glog.V(2).Infof("STATUS %v from %s (%s)\n", status, uuid, role.String())

	if role.IsAgent() {
		var cn *nodeStat
		sched.cnMutex.RLock()
		defer sched.cnMutex.RUnlock()
		if sched.cnMap[uuid] != nil {
			cn = sched.cnMap[uuid]
			sched.updateNodeStat(cn, status, frame)
		}
	}

	if role.IsNetAgent() {
		var nn *nodeStat
		sched.nnMutex.RLock()
		defer sched.nnMutex.RUnlock()
		if sched.nnMap[uuid] != nil {
			nn = sched.nnMap[uuid]
			sched.updateNodeStat(nn, status, frame)
		}
	}
}

type workResources struct {
	instanceUUID string
	memReqMB     int
	diskReqMB    int
	networkNode  bool
	physNets     []string
}

func (sched *ssntpSchedulerServer) getWorkloadResources(work *payloads.Start) (workload workResources, err error) {
	// loop the array to find resources
	for idx := range work.Start.RequestedResources {
		reqType := work.Start.RequestedResources[idx].Type
		reqValue := work.Start.RequestedResources[idx].Value
		reqString := work.Start.RequestedResources[idx].ValueString

		// memory:
		if reqType == payloads.MemMB {
			workload.memReqMB = reqValue
		}

		// network node
		if reqType == payloads.NetworkNode {
			wantsNetworkNode := reqValue

			// validate input: requested resource values are always integers
			if wantsNetworkNode != 0 && wantsNetworkNode != 1 {
				return workload, fmt.Errorf("invalid start payload resource demand: network_node (%d) is not 0 or 1", wantsNetworkNode)
			}

			// convert to more natural bool for local struct
			if wantsNetworkNode == 1 {
				workload.networkNode = true
			} else { //wantsNetworkNode == 0
				workload.networkNode = false
			}

		}

		// network node physical networks
		if workload.networkNode {
			if reqType == payloads.PhysicalNetwork {
				workload.physNets = append(workload.physNets, reqString)
			}
		}

		// etc...
	}

	// volumes
	for _, volume := range work.Start.Storage {
		if volume.Local {
			workload.diskReqMB += volume.Size * 1024
		}
	}

	// validate the found resources
	if workload.memReqMB <= 0 {
		return workload, fmt.Errorf("invalid start payload resource demand: mem_mb (%d) <= 0, must be > 0", workload.memReqMB)
	}
	if workload.diskReqMB < 0 {
		return workload, fmt.Errorf("invalid start payload local disk demand: disk MB (%d) < 0, must be >= 0", workload.diskReqMB)
	}

	// note the uuid
	workload.instanceUUID = work.Start.InstanceUUID

	return workload, nil
}

func networkDemandsSatisfied(node *nodeStat, workload *workResources) bool {
	if !node.isNetNode {
		return true
	}

	var matchedNetworksCount int
	var requestedNetworksCount int

	for _, requestedNetwork := range workload.physNets {
		requestedNetworksCount++
		for _, availableNetwork := range node.networks {
			if requestedNetwork == availableNetwork.NodeIP {
				matchedNetworksCount++
				break
			}
		}
	}
	if requestedNetworksCount != matchedNetworksCount {
		return false
	}

	return true
}

// Check resource demands are satisfiable by the referenced, locked nodeStat object
func (sched *ssntpSchedulerServer) workloadFits(node *nodeStat, workload *workResources) bool {
	// simple scheduling policy == first fit
	if node.memAvailMB >= workload.memReqMB &&
		node.diskAvailMB >= workload.diskReqMB &&
		node.status == ssntp.READY &&
		networkDemandsSatisfied(node, workload) {

		return true
	}
	return false
}

func (sched *ssntpSchedulerServer) sendStartFailureError(clientUUID string, instanceUUID string, reason payloads.StartFailureReason, restart bool) {
	error := payloads.ErrorStartFailure{
		InstanceUUID: instanceUUID,
		Reason:       reason,
		Restart:      restart,
	}

	payload, err := yaml.Marshal(&error)
	if err != nil {
		glog.Errorf("Unable to Marshall Status %v", err)
		return
	}

	glog.Warningf("Unable to dispatch: %v\n", reason)
	sched.ssntp.SendError(clientUUID, ssntp.StartFailure, payload)
}

func (sched *ssntpSchedulerServer) getCommandConcentratorUUID(command ssntp.Command, payload []byte) (string, error) {
	switch command {
	default:
		return "", fmt.Errorf("unsupported ssntp.Command type \"%s\"", command)
	case ssntp.AssignPublicIP:
		var cmd payloads.CommandAssignPublicIP
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.AssignIP.ConcentratorUUID, err
	case ssntp.ReleasePublicIP:
		var cmd payloads.CommandReleasePublicIP
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.ReleaseIP.ConcentratorUUID, err
	}
}

func (sched *ssntpSchedulerServer) getEventConcentratorUUID(event ssntp.Event, payload []byte) (string, error) {
	switch event {
	default:
		return "", fmt.Errorf("unsupported ssntp.Event type \"%s\"", event)
	case ssntp.TenantAdded:
		var ev payloads.EventTenantAdded
		err := yaml.Unmarshal(payload, &ev)
		return ev.TenantAdded.ConcentratorUUID, err
	case ssntp.TenantRemoved:
		var ev payloads.EventTenantRemoved
		err := yaml.Unmarshal(payload, &ev)
		return ev.TenantRemoved.ConcentratorUUID, err
	}
}

func (sched *ssntpSchedulerServer) fwdCmdToCNCI(command ssntp.Command, payload []byte) (dest ssntp.ForwardDestination) {
	// since the scheduler is the primary ssntp server, it needs to
	// unwrap CNCI directed command payloads and forward to the right CNCI

	concentratorUUID, err := sched.getCommandConcentratorUUID(command, payload)
	if err != nil || concentratorUUID == "" {
		glog.Errorf("Bad %s command yaml. Unable to forward to CNCI.\n", command)
		dest.SetDecision(ssntp.Discard)
		return
	}

	glog.V(2).Infof("Forwarding %s command to CNCI Agent %s\n", command.String(), concentratorUUID)
	dest.AddRecipient(concentratorUUID)

	return dest
}

func (sched *ssntpSchedulerServer) fwdEventToCNCI(event ssntp.Event, payload []byte) (dest ssntp.ForwardDestination) {
	// since the scheduler is the primary ssntp server, it needs to
	// unwrap CNCI directed event payloads and forward to the right CNCI

	concentratorUUID, err := sched.getEventConcentratorUUID(event, payload)
	if err != nil || concentratorUUID == "" {
		glog.Errorf("Bad %s event yaml. Unable to forward to CNCI.\n", event)
		dest.SetDecision(ssntp.Discard)
		return
	}

	glog.V(2).Infof("Forwarding %s command to CNCI Agent%s\n", event.String(), concentratorUUID)
	dest.AddRecipient(concentratorUUID)

	return dest
}

func getWorkloadAgentUUID(sched *ssntpSchedulerServer, command ssntp.Command, payload []byte) (string, string, error) {
	switch command {
	default:
		return "", "", fmt.Errorf("unsupported ssntp.Command type \"%s\"", command)
	case ssntp.RESTART:
		var cmd payloads.Restart
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.Restart.InstanceUUID, cmd.Restart.WorkloadAgentUUID, err
	case ssntp.STOP:
		var cmd payloads.Stop
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.Stop.InstanceUUID, cmd.Stop.WorkloadAgentUUID, err
	case ssntp.DELETE:
		var cmd payloads.Delete
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.Delete.InstanceUUID, cmd.Delete.WorkloadAgentUUID, err
	case ssntp.EVACUATE:
		var cmd payloads.Evacuate
		err := yaml.Unmarshal(payload, &cmd)
		return "", cmd.Evacuate.WorkloadAgentUUID, err

	case ssntp.AttachVolume:
		var cmd payloads.AttachVolume
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.Attach.InstanceUUID, cmd.Attach.WorkloadAgentUUID, err
	case ssntp.DetachVolume:
		var cmd payloads.DetachVolume
		err := yaml.Unmarshal(payload, &cmd)
		return cmd.Detach.InstanceUUID, cmd.Detach.WorkloadAgentUUID, err
	}
}

func (sched *ssntpSchedulerServer) fwdCmdToComputeNode(command ssntp.Command, payload []byte) (dest ssntp.ForwardDestination, instanceUUID string) {
	// some commands require no scheduling choice, rather the specified
	// agent/launcher needs the command instead of the scheduler
	instanceUUID, cnDestUUID, err := getWorkloadAgentUUID(sched, command, payload)
	if err != nil || cnDestUUID == "" {
		glog.Errorf("Bad %s command yaml from Controller, WorkloadAgentUUID == %s\n", command.String(), cnDestUUID)
		dest.SetDecision(ssntp.Discard)
		return
	}

	glog.V(2).Infof("Forwarding controller %s command to %s\n", command.String(), cnDestUUID)
	dest.AddRecipient(cnDestUUID)

	return
}

// Decrement resource claims for the referenced locked nodeStat object
func (sched *ssntpSchedulerServer) decrementResourceUsage(node *nodeStat, workload *workResources) {
	node.memAvailMB -= workload.memReqMB
}

// Find suitable compute node, returning referenced to a locked nodeStat if found
func pickComputeNode(sched *ssntpSchedulerServer, controllerUUID string, workload *workResources, restart bool) (node *nodeStat) {
	sched.cnMutex.RLock()
	defer sched.cnMutex.RUnlock()

	if len(sched.cnList) == 0 {
		glog.Errorf("No compute nodes connected, unable to start workload")
		sched.sendStartFailureError(controllerUUID, workload.instanceUUID, payloads.NoComputeNodes, restart)
		return nil
	}

	/* First try nodes after the MRU */
	if sched.cnMRUIndex != -1 && sched.cnMRUIndex < len(sched.cnList)-1 {
		for i, node := range sched.cnList[sched.cnMRUIndex+1:] {
			node.mutex.Lock()
			if node == sched.cnMRU {
				node.mutex.Unlock()
				continue
			}

			if sched.workloadFits(node, workload) == true {
				sched.cnMRUIndex = sched.cnMRUIndex + 1 + i
				sched.cnMRU = node
				return node // locked nodeStat
			}
			node.mutex.Unlock()
		}
	}

	/* Then try the whole list, including the MRU */
	for i, node := range sched.cnList {
		node.mutex.Lock()
		if sched.workloadFits(node, workload) == true {
			sched.cnMRUIndex = i
			sched.cnMRU = node
			return node // locked nodeStat
		}
		node.mutex.Unlock()
	}

	sched.sendStartFailureError(controllerUUID, workload.instanceUUID, payloads.FullCloud, restart)
	return nil
}

// Find suitable net node, returning referenced to a locked nodeStat if found
func pickNetworkNode(sched *ssntpSchedulerServer, controllerUUID string, workload *workResources, restart bool) (node *nodeStat) {
	sched.nnMutex.RLock()
	defer sched.nnMutex.RUnlock()

	if len(sched.nnList) == 0 {
		glog.Errorf("No network nodes connected, unable to start network workload")
		sched.sendStartFailureError(controllerUUID, workload.instanceUUID, payloads.NoNetworkNodes, restart)
		return nil
	}

	/* First try nodes after the MRU */
	if sched.nnMRUIndex != -1 && sched.nnMRUIndex < len(sched.nnList)-1 {
		for i, node := range sched.nnList[sched.nnMRUIndex+1:] {
			node.mutex.Lock()
			if node == sched.nnMRU {
				node.mutex.Unlock()
				continue
			}

			if sched.workloadFits(node, workload) == true {
				sched.nnMRUIndex = sched.nnMRUIndex + 1 + i
				sched.nnMRU = node
				return node // locked nodeStat
			}
			node.mutex.Unlock()
		}
	}

	/* Then try the whole list, including the MRU */
	for i, node := range sched.nnList {
		node.mutex.Lock()
		if sched.workloadFits(node, workload) == true {
			sched.nnMRUIndex = i
			sched.nnMRU = node
			return node // locked nodeStat
		}
		node.mutex.Unlock()
	}

	sched.sendStartFailureError(controllerUUID, workload.instanceUUID, payloads.NoNetworkNodes, restart)
	return nil
}

func startWorkload(sched *ssntpSchedulerServer, controllerUUID string, payload []byte) (dest ssntp.ForwardDestination, instanceUUID string) {
	var work payloads.Start
	err := yaml.Unmarshal(payload, &work)
	if err != nil {
		glog.Errorf("Bad START workload yaml from Controller %s: %s\n", controllerUUID, err)
		dest.SetDecision(ssntp.Discard)
		return dest, ""
	}

	workload, err := sched.getWorkloadResources(&work)
	if err != nil {
		glog.Errorf("Bad START workload resource list from Controller %s: %s\n", controllerUUID, err)
		dest.SetDecision(ssntp.Discard)
		return dest, ""
	}

	instanceUUID = workload.instanceUUID

	var targetNode *nodeStat

	if workload.networkNode {
		targetNode = pickNetworkNode(sched, controllerUUID, &workload, work.Start.Restart)
	} else { //workload.network_node == false
		targetNode = pickComputeNode(sched, controllerUUID, &workload, work.Start.Restart)
	}

	if targetNode != nil {
		//TODO: mark the targetNode as unavailable until next stats / READY checkin?
		//	or is subtracting mem demand sufficiently speculative enough?
		//	Goal is to have spread, not schedule "too many" workloads back
		//	to back on the same targetNode, but also not add latency to dispatch and
		//	hopefully not queue when all nodes have just started a workload.
		sched.decrementResourceUsage(targetNode, &workload)

		dest.AddRecipient(targetNode.uuid)
		targetNode.mutex.Unlock()
	} else {
		// TODO Queue the frame ?
		dest.SetDecision(ssntp.Discard)
	}

	return dest, instanceUUID
}

func (sched *ssntpSchedulerServer) CommandForward(controllerUUID string, command ssntp.Command, frame *ssntp.Frame) (dest ssntp.ForwardDestination) {
	payload := frame.Payload
	instanceUUID := ""

	sched.controllerMutex.RLock()
	defer sched.controllerMutex.RUnlock()
	if sched.controllerMap[controllerUUID] == nil {
		glog.Warningf("Ignoring %s command from unknown Controller %s\n", command, controllerUUID)
		dest.SetDecision(ssntp.Discard)
		return
	}
	controller := sched.controllerMap[controllerUUID]
	controller.mutex.Lock()
	if controller.status != controllerMaster {
		glog.Warningf("Ignoring %s command from non-master Controller %s\n", command, controllerUUID)
		dest.SetDecision(ssntp.Discard)
		controller.mutex.Unlock()
		return
	}
	controller.mutex.Unlock()

	start := time.Now()

	glog.V(2).Infof("Command %s from %s\n", command, controllerUUID)

	switch command {
	// the main command with scheduler processing
	case ssntp.START:
		dest, instanceUUID = startWorkload(sched, controllerUUID, payload)
	case ssntp.RESTART:
		fallthrough
	case ssntp.STOP:
		fallthrough
	case ssntp.DELETE:
		fallthrough
	case ssntp.AttachVolume:
		fallthrough
	case ssntp.DetachVolume:
		fallthrough
	case ssntp.EVACUATE:
		dest, instanceUUID = sched.fwdCmdToComputeNode(command, payload)
	case ssntp.AssignPublicIP:
		fallthrough
	case ssntp.ReleasePublicIP:
		dest = sched.fwdCmdToCNCI(command, payload)
	default:
		dest.SetDecision(ssntp.Discard)
	}

	elapsed := time.Since(start)
	glog.V(2).Infof("%s command processed for instance %s in %s\n", command, instanceUUID, elapsed)

	return
}

func (sched *ssntpSchedulerServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	// Currently all commands are handled by CommandForward, the SSNTP command forwader,
	// or directly by role defined forwarding rules.
	glog.V(2).Infof("COMMAND %v from %s\n", command, uuid)
}

func (sched *ssntpSchedulerServer) EventForward(uuid string, event ssntp.Event, frame *ssntp.Frame) (dest ssntp.ForwardDestination) {
	payload := frame.Payload

	start := time.Now()

	switch event {
	case ssntp.TenantAdded:
		fallthrough
	case ssntp.TenantRemoved:
		dest = sched.fwdEventToCNCI(event, payload)
	}

	elapsed := time.Since(start)
	glog.V(2).Infof("%s event processed for instance %s in %s\n", event.String(), uuid, elapsed)

	return dest
}

func (sched *ssntpSchedulerServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
	// Currently all events are handled by EventForward, the SSNTP command forwader,
	// or directly by role defined forwarding rules.
	glog.V(2).Infof("EVENT %v from %s\n", event, uuid)
}

func (sched *ssntpSchedulerServer) ErrorNotify(uuid string, error ssntp.Error, frame *ssntp.Frame) {
	glog.V(2).Infof("ERROR %v from %s\n", error, uuid)
}

func setLimits() {
	var rlim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		glog.Warningf("Getrlimit failed %v", err)
		return
	}

	glog.Infof("Initial nofile limits: cur %d max %d", rlim.Cur, rlim.Max)

	if rlim.Cur < rlim.Max {
		oldCur := rlim.Cur
		rlim.Cur = rlim.Max
		err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
		if err != nil {
			glog.Warningf("Setrlimit failed %v", err)
			rlim.Cur = oldCur
		}
	}

	glog.Infof("Updated nofile limits: cur %d max %d", rlim.Cur, rlim.Max)
}

func heartBeatControllers(sched *ssntpSchedulerServer) (s string) {
	// show the first two controller's
	controllerMax := 2
	i := 0

	sched.controllerMutex.RLock()
	defer sched.controllerMutex.RUnlock()

	if len(sched.controllerList) == 0 {
		return " -no Controller- \t\t\t\t\t"
	}

	// first show any master, which is at front of list
	controller := sched.controllerList[0]
	controller.mutex.Lock()
	if controller.status == controllerMaster {
		s += fmt.Sprintf("controller-%s:", controller.uuid[:8])
		s += controller.status.String()
		controller.mutex.Unlock()

		i++
		if i <= controllerMax && len(sched.controllerList) > i {
			s += ", "
		} else {
			s += "\t"
		}
	}

	// second show any backup(s)
	for _, controller := range sched.controllerList[i:] {
		if i == controllerMax {
			break
		}

		controller.mutex.Lock()
		if controller.status == controllerMaster {
			controller.mutex.Unlock()
			glog.Errorf("multiple controller masters")
			return "ERROR multiple controller masters"
		}

		s += fmt.Sprintf("controller-%s:", controller.uuid[:8])
		s += controller.status.String()
		controller.mutex.Unlock()

		i++
		if i < controllerMax && len(sched.controllerList) > i {
			s += ", "
		} else {
			s += "\t"
		}
	}

	// finish with some whitespace ahead of compute nodes
	if i < controllerMax {
		s += "\t\t\t"
	} else {
		s += "\t"
	}

	return s
}

func heartBeatComputeNodes(sched *ssntpSchedulerServer) (s string) {
	// show the first four compute nodes
	cnMax := 4
	i := 0

	sched.cnMutex.RLock()
	defer sched.cnMutex.RUnlock()

	for _, node := range sched.cnList {

		node.mutex.Lock()
		s += fmt.Sprintf("node-%s:", node.uuid[:8])
		s += node.status.String()
		if node == sched.cnMRU {
			s += "*"
		}
		s += ":" + fmt.Sprintf("%d/%d,%d",
			node.memAvailMB,
			node.memTotalMB,
			node.load)
		node.mutex.Unlock()

		i++
		if i == cnMax {
			break
		}
		if i <= cnMax && len(sched.cnList) > i {
			s += ", "
		}
	}

	if i == 0 {
		s += " -no Compute Nodes-"
	}

	return s
}

const heartBeatHeaderFreq = 22

func heartBeat(sched *ssntpSchedulerServer, iter int) string {
	var beatTxt string

	time.Sleep(time.Duration(1) * time.Second)

	sched.controllerMutex.RLock()
	sched.cnMutex.RLock()
	if len(sched.controllerList) == 0 && len(sched.cnList) == 0 {
		sched.controllerMutex.RUnlock()
		sched.cnMutex.RUnlock()
		return "** idle / disconnected **\n"
	}
	sched.controllerMutex.RUnlock()
	sched.cnMutex.RUnlock()

	iter++
	if iter%heartBeatHeaderFreq == 0 {
		//output a column indication occasionally
		beatTxt = "Controllers\t\t\t\t\tCompute Nodes\n"
	}

	beatTxt += heartBeatControllers(sched) + heartBeatComputeNodes(sched)

	return beatTxt
}

func heartBeatLoop(sched *ssntpSchedulerServer) {
	iter := 0
	for {
		log.Printf("%s\n", heartBeat(sched, iter))
	}
}

func toggleDebug(sched *ssntpSchedulerServer) {
	if len(sched.cpuprofile) != 0 {
		f, err := os.Create(sched.cpuprofile)
		if err != nil {
			glog.Warningf("unable to initialize cpuprofile (%s)", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	/* glog's --logtostderr and -v=2 are probably really what you want..
	sched.config.Trace = os.Stdout
	sched.config.Error = os.Stdout
	sched.config.DebugInterface = false
	*/

	if sched.heartbeat {
		go heartBeatLoop(sched)
	}
}

func setSSNTPForwardRules(sched *ssntpSchedulerServer) {
	sched.config.ForwardRules = []ssntp.FrameForwardRule{
		{ // all STATS commands go to all Controllers
			Operand: ssntp.STATS,
			Dest:    ssntp.Controller,
		},
		{ // all TraceReport events go to all Controllers
			Operand: ssntp.TraceReport,
			Dest:    ssntp.Controller,
		},
		{ // all InstanceDeleted events go to all Controllers
			Operand: ssntp.InstanceDeleted,
			Dest:    ssntp.Controller,
		},
		{ // all InstanceStopped events go to all Controllers
			Operand: ssntp.InstanceStopped,
			Dest:    ssntp.Controller,
		},
		{ // all ConcentratorInstanceAdded events go to all Controllers
			Operand: ssntp.ConcentratorInstanceAdded,
			Dest:    ssntp.Controller,
		},
		{ // all StartFailure errors go to all Controllers
			Operand: ssntp.StartFailure,
			Dest:    ssntp.Controller,
		},
		{ // all StopFailure errors go to all Controllers
			Operand: ssntp.StopFailure,
			Dest:    ssntp.Controller,
		},
		{ // all RestartFailure errors go to all Controllers
			Operand: ssntp.RestartFailure,
			Dest:    ssntp.Controller,
		},
		{ // all DeleteFailure errors go to all Controllers
			Operand: ssntp.DeleteFailure,
			Dest:    ssntp.Controller,
		},
		{ // all PublicIPAssigned events go to all Controllers
			Operand: ssntp.PublicIPAssigned,
			Dest:    ssntp.Controller,
		},
		{ // all AssignPublicIPFailure events go to all Controllers
			Operand: ssntp.AssignPublicIPFailure,
			Dest:    ssntp.Controller,
		},
		{ // all PublicIPUnassigned events go to all Controllers
			Operand: ssntp.PublicIPUnassigned,
			Dest:    ssntp.Controller,
		},
		{ // all UnassignPublicIPFailure events go to all Controllers
			Operand: ssntp.UnassignPublicIPFailure,
			Dest:    ssntp.Controller,
		},
		{ // all START command are processed by the Command forwarder
			Operand:        ssntp.START,
			CommandForward: sched,
		},
		{ // all RESTART command are processed by the Command forwarder
			Operand:        ssntp.RESTART,
			CommandForward: sched,
		},
		{ // all STOP command are processed by the Command forwarder
			Operand:        ssntp.STOP,
			CommandForward: sched,
		},
		{ // all DELETE command are processed by the Command forwarder
			Operand:        ssntp.DELETE,
			CommandForward: sched,
		},
		{ // all EVACUATE command are processed by the Command forwarder
			Operand:        ssntp.EVACUATE,
			CommandForward: sched,
		},
		{ // all TenantAdded events are processed by the Event forwarder
			Operand:      ssntp.TenantAdded,
			EventForward: sched,
		},
		{ // all TenantRemoved events are processed by the Event forwarder
			Operand:      ssntp.TenantRemoved,
			EventForward: sched,
		},
		{ // all AttachVolume command are processed by the Command forwarder
			Operand:        ssntp.AttachVolume,
			CommandForward: sched,
		},
		{ // all DetachVolume command are processed by the Command forwarder
			Operand:        ssntp.DetachVolume,
			CommandForward: sched,
		},
		{ // all AttachVolumeFailure errors go to all Controllers
			Operand: ssntp.AttachVolumeFailure,
			Dest:    ssntp.Controller,
		},
		{ // all DetachVolumeFailure errors go to all Controllers
			Operand: ssntp.DetachVolumeFailure,
			Dest:    ssntp.Controller,
		},
		{ // all AssignPublicIP commands are processed by the Command forwarder
			Operand:        ssntp.AssignPublicIP,
			CommandForward: sched,
		},
		{ // all ReleasePublicIP commands are processed by the Command forwarder
			Operand:        ssntp.ReleasePublicIP,
			CommandForward: sched,
		},
	}
}

func initLogger() error {
	logDirFlag := flag.Lookup("log_dir")
	if logDirFlag == nil {
		return fmt.Errorf("log_dir does not exist")
	}
	if logDirFlag.Value.String() == "" {
		if err := logDirFlag.Value.Set(logDir); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(logDirFlag.Value.String(), 0755); err != nil {
		return fmt.Errorf("Unable to create log directory (%s) %v", logDir, err)
	}
	return nil
}

func configSchedulerServer() (sched *ssntpSchedulerServer) {
	setLimits()

	sched = newSsntpSchedulerServer()
	sched.cpuprofile = *cpuprofile
	sched.heartbeat = *heartbeat

	toggleDebug(sched)

	sched.config = &ssntp.Config{
		CAcert:    *cacert,
		Cert:      *cert,
		ConfigURI: *configURI,
	}

	setSSNTPForwardRules(sched)

	return sched
}

func main() {
	flag.Parse()

	if err := initLogger(); err != nil {
		fmt.Printf("Unable to initialise logs: %v", err)
		return
	}

	glog.Info("Starting Scheduler")

	logger := gloginterface.CiaoGlogLogger{}
	osprepare.Bootstrap(context.TODO(), logger)
	osprepare.InstallDeps(context.TODO(), schedDeps, logger)

	sched := configSchedulerServer()
	if sched == nil {
		glog.Errorf("unable to configure scheduler")
		return
	}

	sched.ssntp.Serve(sched.config, sched)
}
